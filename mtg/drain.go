package mtg

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
)

const (
	outputsDrainingKey = "outputs-draining-checkpoint"
)

func (grp *Group) drainOutputsFromNetwork(ctx context.Context, filter map[string]bool, batch int) {
	logger.Verbosef("Group.drainOutputsFromNetwork(%d)\n", batch)

	for {
		checkpoint, err := grp.readDrainingCheckpoint(ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		outputs, err := grp.readSafeOutputsAsUnspent(ctx, grp.members, uint8(grp.threshold), checkpoint, batch)
		logger.Verbosef("Group.readSafeOutputsAsUnspent(%d) => %d %v\n", checkpoint, len(outputs), err)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		checkpoint = grp.processSafeOutputs(ctx, filter, checkpoint, outputs)
		grp.writeDrainingCheckpoint(ctx, checkpoint)
		if len(outputs) < batch/2 {
			break
		}
	}
}

func (grp *Group) processSafeOutputs(ctx context.Context, filter map[string]bool, checkpoint uint64, outputs []*UnifiedOutput) uint64 {
	for _, utxo := range outputs {
		checkpoint = utxo.Sequence
		key := fmt.Sprintf("ACT:%s:%d", utxo.OutputId, utxo.Sequence)
		if filter[key] || utxo.Sequence < grp.epoch {
			continue
		}
		filter[key] = true
		grp.processSafeOutput(ctx, utxo)
	}
	return checkpoint
}

func (grp *Group) processSafeOutput(ctx context.Context, output *UnifiedOutput) {
	logger.Verbosef("Group.processSafeOutput(%v) \n", output)
	actionState := ActionStateInitial

	appId, _ := DecodeMixinExtraHEX(output.Extra)
	if output.RequestId == "" {
		d, err := grp.readOutputDepositUntilSufficient(ctx, output.OutputId)
		if err != nil {
			panic(err)
		}
		if d != nil {
			appId = grp.FindAppByEntry(DepositEntry{
				Destination: d.Destination,
				Tag:         d.Tag,
			}.UniqueKey())
		}
	}
	if appId == "" {
		appId = grp.GroupId
	}
	output.AppId = appId

	appId, err := grp.checkChange(ctx, output)
	if err != nil {
		panic(err)
	}
	if appId != "" {
		output.AppId = appId
		actionState = ActionStateDone
	}
	grp.store.WriteAction(ctx, output, actionState)
}

func (grp *Group) checkChange(ctx context.Context, output *UnifiedOutput) (string, error) {
	// we must always ensure there are at most 2 outputs,
	// and the last one is the change output
	if output.OutputIndex != 1 {
		return "", nil
	}
	req, err := grp.readTransactionUntilSufficient(ctx, output.TransactionHash.String())
	if err != nil {
		return "", err
	}
	rb, err := hex.DecodeString(req.RawTransaction)
	if err != nil {
		return "", err
	}
	ver, err := common.UnmarshalVersionedTransaction(rb)
	if err != nil {
		return "", err
	}

	var appId string
	for _, input := range ver.Inputs {
		output, err := grp.store.ReadOutputByHashAndIndex(ctx, input.Hash.String(), input.Index)
		if err != nil || output == nil {
			return "", err
		}
		if appId == "" {
			appId = output.AppId
		}
		if output.AppId != appId {
			return "", fmt.Errorf("invalid app id: %s", output.OutputId)
		}
	}
	return appId, nil
}

func (grp *Group) readDrainingCheckpoint(ctx context.Context) (uint64, error) {
	val, err := grp.store.ReadProperty(ctx, outputsDrainingKey)
	if err != nil || len(val) == 0 {
		return 0, err
	}
	return strconv.ParseUint(val, 10, 64)
}

func (grp *Group) writeDrainingCheckpoint(ctx context.Context, ckpt uint64) error {
	return grp.store.WriteProperty(ctx, outputsDrainingKey, fmt.Sprint(ckpt))
}

func (grp *Group) readSafeOutputsAsUnspent(ctx context.Context, members []string, threshold uint8, offset uint64, limit int) ([]*UnifiedOutput, error) {
	params := make(map[string]string)
	if offset > 0 {
		params["offset"] = fmt.Sprint(offset)
	}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	if threshold < 1 {
		threshold = 1
	}
	if int(threshold) > len(members) {
		return nil, errors.New("invalid members")
	}
	params["members"] = mixinnet.HashMembers(members)
	params["threshold"] = fmt.Sprint(threshold)
	params["order"] = "ASC"

	var utxos []*UnifiedOutput
	if err := grp.mixin.Get(ctx, "/safe/outputs", params, &utxos); err != nil {
		return nil, err
	}
	for _, o := range utxos {
		o.State = SafeUtxoStateUnspent
	}
	return utxos, nil
}

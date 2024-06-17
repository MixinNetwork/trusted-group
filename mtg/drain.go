package mtg

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go/v2"
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
		outputs, err := grp.readSafeOutputs(ctx, grp.members, uint8(grp.threshold), checkpoint, batch)
		logger.Verbosef("Group.readSafeOutputs(%d) => %d %v\n", checkpoint, len(outputs), err)
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

	appId, traceId, _ := DecodeMixinExtra(output.Extra)
	if output.RequestId == "" {
		d, err := grp.readOutputDepositUntilSufficient(ctx, output.OutputId)
		if err != nil {
			panic(err)
		}
		if d != nil {
			appId = grp.FindAppByEntry(DepositEntry{
				Destination: d.Destination,
				Tag:         d.Tag,
			}.String())
		}
	}
	if appId == "" {
		appId = grp.GroupId
	}
	switch output.State {
	case SafeUtxoStateUnspent:
	case SafeUtxoStateSigned, SafeUtxoStateSpent:
		req, err := grp.readMultisigUntilSufficient(ctx, output.SignedBy)
		if err != nil {
			panic(err)
		}
		traceId = req.RequestID
	}
	output.AppId = appId
	output.TraceId = traceId

	tx, err := grp.store.ReadTransactionByHash(ctx, output.TransactionHash)
	if err != nil {
		panic(err)
	}
	if tx != nil {
		if tx.Compaction {
			err = grp.store.restoreAction(ctx, output, tx)
			if err != nil {
				panic(err)
			}
			return
		}
		if output.OutputIndex == 1 {
			switch tx.State {
			case TransactionStateSnapshot:
				output.AppId = tx.AppId
				output.TraceId = tx.TraceId
			default:
				req, err := grp.readTransactionUntilSufficient(ctx, output.TransactionHash.String())
				if err != nil {
					panic(err)
				}
				if req != nil && req.State == mixin.SafeUtxoStateSpent {
					output.AppId = tx.AppId
					output.TraceId = tx.TraceId
				}
			}
		}
	}
	grp.store.writeAction(ctx, output, ActionStateInitial)
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

func (grp *Group) readSafeOutputs(ctx context.Context, members []string, threshold uint8, offset uint64, limit int) ([]*UnifiedOutput, error) {
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
	return utxos, nil
}

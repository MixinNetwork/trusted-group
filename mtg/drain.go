package mtg

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/gofrs/uuid/v5"
)

const (
	outputsDrainingKey = "outputs-draining-checkpoint"
)

func (grp *Group) drainOutputsFromNetwork(ctx context.Context, filter map[string]bool, batch int) {
	logger.Verbosef("Group.drainOutputsFromNetwork(%d, %s)\n", batch)

	for {
		checkpoint, err := grp.readDrainingCheckpoint(ctx)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		outputs, err := grp.readSafeOutputs(ctx, grp.members, uint8(grp.threshold), checkpoint, batch)
		logger.Verbosef("Group.readSafeOutputs(%s, %s) => %d %v\n", checkpoint, len(outputs), err)
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
	for _, out := range outputs {
		checkpoint = out.Sequence
		key := fmt.Sprintf("OUT:%s:%d", out.OutputId, out.Sequence)
		if filter[key] || out.Sequence < grp.epoch {
			continue
		}
		filter[key] = true
		grp.processSafeOutput(ctx, out.Output())
	}

	for _, utxo := range outputs {
		key := fmt.Sprintf("ACT:%s:%d", utxo.OutputId, utxo.Sequence)
		if filter[key] || utxo.Sequence < grp.epoch {
			continue
		}
		filter[key] = true
		exist, err := grp.readOldTransaction(utxo)
		if err != nil {
			panic(utxo.TransactionHash)
		} else if exist {
			continue
		}
		grp.writeAction(utxo, ActionStateInitial)
	}
	return checkpoint
}

func (grp *Group) readOldTransaction(utxo *UnifiedOutput) (bool, error) {
	switch utxo.Type {
	case OutputTypeSafeOutput:
		tx, err := grp.store.ReadTransactionByHash(utxo.TransactionHash)
		return tx != nil, err
	default:
		panic(utxo.Type)
	}
}

func (grp *Group) processSafeOutput(ctx context.Context, out *Output) {
	logger.Verbosef("Group.processSafeOutput(%v)", out)
	t, err := grp.mixin.SafeReadMultisigRequests(ctx, out.SignedBy)
	if err != nil {
		panic(err)
	}
	ver, extra := decodeTransactionWithExtra(t.RawTransaction)
	if ver == nil {
		panic(t.RawTransaction)
	}
	out.SignedTx = t.RawTransaction
	if grp.checkCompactTransactionRequest(ctx, ver, extra) {
		amount := ver.Outputs[0].Amount.String()
		receivers, threshold := grp.GetMembers(), grp.GetThreshold()
		err := grp.buildTransaction(ctx, out.AssetID, receivers, threshold, amount, CompactionTransactionMemo, extra.T.String(), extra.G, time.Unix(0, 0), nil)
		logger.Printf("Group.drainCompactTransaction(%s, %s, %s) => %v\n", extra.G, extra.T.String(), amount, err)
		if err != nil {
			panic(err)
		}
	}
	var groupId, traceId string
	if extra != nil {
		groupId, traceId = extra.G, extra.T.String()
	} else {
		groupId = uuid.Nil.String()
		traceId = mixin.UniqueConversationID(out.UTXOID, out.SignedBy)
	}

	// FIXME get trace id from other members could break the consensus
	// this in theory won't affect asset security though
	if out.State == OutputStateUnspent || (ver.AggregatedSignature == nil && len(ver.SignaturesMap) == 0) {
		grp.writeOutputOrPanic(out, traceId)
		return
	}
	tx := &Transaction{
		GroupId: groupId,
		TraceId: traceId,
		State:   TransactionStateSigned,
		Raw:     ver.Marshal(),
		Hash:    ver.PayloadHash(),
	}

	out.State = OutputStateSpent
	grp.writeOutputOrPanic(out, tx.TraceId)

	old, err := grp.store.ReadTransactionByTraceId(tx.TraceId)
	if err != nil {
		panic(err)
	}
	if old != nil && old.State >= TransactionStateSigned {
		return
	}
	grp.writeTansactionOrPanic(tx)
}

func (grp *Group) writeOutputOrPanic(out *Output, traceId string) {
	// FIXME some invalid memo could also be randomly decoded
	// thus result in incorrect group id
	p := DecodeMixinExtra(out.Memo)
	if p != nil && p.G != "" {
		out.GroupId = p.G
	} else if grp.grouper != nil {
		out.GroupId = grp.grouper(out)
	}
	logger.Verbosef("Group.writeOutputOrPanic(%v, %s)", out, traceId)
	err := grp.store.WriteOutput(out, traceId)
	if err != nil {
		panic(err)
	}
}

func (grp *Group) readDrainingCheckpoint(ctx context.Context) (uint64, error) {
	val, err := grp.store.ReadProperty([]byte(outputsDrainingKey))
	if err != nil || len(val) == 0 {
		return 0, err
	}
	return uint64(binary.BigEndian.Uint64(val)), nil
}

func (grp *Group) writeDrainingCheckpoint(ctx context.Context, ckpt uint64) error {
	val := make([]byte, 8)
	binary.BigEndian.PutUint64(val, ckpt)
	return grp.store.WriteProperty([]byte(outputsDrainingKey), val)
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

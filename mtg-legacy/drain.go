package mtg

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid/v5"
)

const (
	outputsOrderCreated = "created"
	outputsOrderUpdated = "updated"
	outputsDrainingKey  = "outputs-draining-checkpoint"
)

func (grp *Group) drainOutputsFromNetwork(ctx context.Context, filter map[string]bool, batch int, order string) {
	logger.Verbosef("Group.drainOutputsFromNetwork(%d, %s)\n", batch, order)
	if order != outputsOrderCreated && order != outputsOrderUpdated {
		panic(order)
	}

	for {
		checkpoint, err := grp.readDrainingCheckpoint(ctx, order)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		outputs, err := grp.readUnifiedOutputs(ctx, grp.members, uint8(grp.threshold), checkpoint, batch, order)
		logger.Verbosef("Group.readUnifiedOutputs(%s, %s) => %d %v\n", checkpoint, order, len(outputs), err)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		checkpoint = grp.processUnifiedOutputs(ctx, filter, checkpoint, outputs, order)
		grp.writeDrainingCheckpoint(ctx, order, checkpoint)
		if len(outputs) < batch/2 {
			break
		}
	}
}

func (grp *Group) processUnifiedOutputs(ctx context.Context, filter map[string]bool, checkpoint time.Time, outputs []*UnifiedOutput, order string) time.Time {
	for _, out := range outputs {
		if order == outputsOrderCreated {
			checkpoint = out.CreatedAt
		} else {
			checkpoint = out.UpdatedAt
		}
		key := fmt.Sprintf("OUT:%s:%d", out.UniqueId(), out.UpdatedAt.UnixNano())
		if filter[key] || out.UpdatedAt.Before(grp.epoch) {
			continue
		}
		filter[key] = true
		if out.Type == OutputTypeMultisig {
			grp.processMultisigOutput(ctx, out.AsMultisig())
		} else if out.Type == OutputTypeCollectible {
			grp.processCollectibleOutput(out.AsCollectible())
		}
	}

	for _, utxo := range outputs {
		key := fmt.Sprintf("ACT:%s:%d", utxo.UniqueId(), utxo.UpdatedAt.UnixNano())
		if filter[key] || utxo.UpdatedAt.Before(grp.epoch) {
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
	if utxo.Type == OutputTypeMultisig {
		tx, err := grp.store.ReadTransactionByHash(utxo.TransactionHash)
		return tx != nil, err
	} else if utxo.Type == OutputTypeCollectible {
		tx, err := grp.store.ReadCollectibleTransactionByHash(utxo.TransactionHash)
		return tx != nil, err
	}
	panic(utxo.Type)
}

func (grp *Group) processMultisigOutput(ctx context.Context, out *Output) {
	logger.Verbosef("Group.processMultisigOutput(%v)", out)
	ver, extra := decodeTransactionWithExtra(out.SignedTx)
	if out.SignedTx != "" && ver == nil {
		panic(out.SignedTx)
	}
	// FIXME do more consensus check to unlock transactions
	if ver != nil && ver.Version < common.TxVersionReferences &&
		ver.AggregatedSignature == nil && len(ver.SignaturesMap) == 0 {
		req, err := grp.createMultisigUntilSufficient(ctx, mixin.MultisigActionUnlock, out.SignedTx)
		if err != nil {
			panic(err)
		}
		err = grp.mixin.UnlockMultisig(ctx, req.RequestID, grp.pin)
		if err != nil {
			panic(err)
		}
		return
	}
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

func (grp *Group) processCollectibleOutput(out *CollectibleOutput) {
	logger.Verbosef("Group.processCollectibleOutput(%v)", out)
	ver, extra := decodeCollectibleTransactionWithExtra(out.SignedTx)
	if out.SignedTx != "" && ver == nil {
		panic(out.SignedTx)
	}
	if out.State == OutputStateUnspent {
		grp.writeCollectibleOutputOrPanic(out, "", nil)
		return
	}
	tx := &CollectibleTransaction{
		TraceId: extra.T.String(),
		State:   TransactionStateInitial,
		Raw:     ver.Marshal(),
		Hash:    ver.PayloadHash(),
		NFO:     ver.Extra,
	}
	if ver.AggregatedSignature != nil || len(ver.SignaturesMap) > 0 {
		out.State = OutputStateSpent
		tx.State = TransactionStateSigned
	}
	grp.writeCollectibleOutputOrPanic(out, tx.TraceId, tx)
}

func (grp *Group) writeCollectibleOutputOrPanic(out *CollectibleOutput, traceId string, tx *CollectibleTransaction) {
	logger.Verbosef("Group.writeCollectibleOutputOrPanic(%v, %s, %v)", out, traceId, tx)
	err := grp.store.WriteCollectibleOutput(out, traceId)
	if err != nil {
		panic(err)
	}
	if traceId == "" {
		return
	}
	old, err := grp.store.ReadCollectibleTransaction(traceId)
	if err != nil {
		panic(err)
	}
	if old != nil && old.State >= TransactionStateSigned {
		return
	}
	err = grp.store.WriteCollectibleTransaction(traceId, tx)
	if err != nil {
		panic(err)
	}
}

func (grp *Group) readDrainingCheckpoint(ctx context.Context, order string) (time.Time, error) {
	key := fmt.Sprintf("%s-by-%s", outputsDrainingKey, order)
	val, err := grp.store.ReadProperty([]byte(key))
	if err != nil || len(val) == 0 {
		return grp.epoch, err
	}
	ts := int64(binary.BigEndian.Uint64(val))
	return time.Unix(0, ts), nil
}

func (grp *Group) writeDrainingCheckpoint(ctx context.Context, order string, ckpt time.Time) error {
	val := make([]byte, 8)
	ts := uint64(ckpt.UnixNano())
	binary.BigEndian.PutUint64(val, ts)
	key := fmt.Sprintf("%s-by-%s", outputsDrainingKey, order)
	return grp.store.WriteProperty([]byte(key), val)
}

func (grp *Group) readUnifiedOutputs(ctx context.Context, members []string, threshold uint8, offset time.Time, limit int, order string) ([]*UnifiedOutput, error) {
	params := make(map[string]string)
	if !offset.IsZero() {
		params["offset"] = offset.UTC().Format(time.RFC3339Nano)
	}
	if limit > 0 {
		params["limit"] = fmt.Sprint(limit)
	}
	if order == outputsOrderCreated || order == outputsOrderUpdated {
		params["order"] = order
	}
	if threshold < 1 || int(threshold) > len(members) {
		return nil, fmt.Errorf("invalid members %v %d", members, threshold)
	}
	params["members"] = mixin.HashMembers(members)
	params["threshold"] = fmt.Sprint(threshold)

	var outputs []*UnifiedOutput
	err := grp.mixin.Get(ctx, "/outputs", params, &outputs)
	if err != nil {
		return nil, err
	}
	return outputs, nil
}

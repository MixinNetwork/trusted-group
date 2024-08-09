package mtg

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
)

const (
	ActionStateInitial    ActionState = 10
	ActionStateDone       ActionState = 11
	ActionStateRestorable ActionState = 12
)

type ActionState int

type Action struct {
	OutputId        string
	TransactionHash string
	ActionState     ActionState
	Sequence        uint64
	restoreSequence uint64

	UnifiedOutput
}

var actionCols = []string{"output_id", "transaction_hash", "action_state", "sequence", "restore_sequence"}

var actionJoinCols = []string{"actions.output_id", "actions.transaction_hash", "action_state", "actions.sequence", "restore_sequence", "request_id", "output_index", "asset_id", "kernel_asset_id", "amount", "mask", "keys", "senders_threshold", "senders", "receivers_threshold", "receivers", "extra", "state", "created_at", "updated_at", "signers", "signed_by", "signed_tx", "trace_id", "app_id"}

func (a *Action) values() []any {
	return []any{a.OutputId, a.TransactionHash, a.ActionState, a.Sequence, a.restoreSequence}
}

func actionFromRow(row Row) (*Action, error) {
	var a Action
	err := row.Scan(&a.OutputId, &a.TransactionHash, &a.ActionState, &a.Sequence, &a.restoreSequence)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func actionJoinFromRow(row Row) (*Action, error) {
	var a Action
	var keys, senders, receivers, signers string
	err := row.Scan(&a.OutputId, &a.TransactionHash, &a.ActionState, &a.Sequence, &a.restoreSequence, &a.RequestId, &a.OutputIndex, &a.AssetId, &a.KernelAssetId, &a.Amount, &a.Mask, &keys, &a.SendersThreshold, &senders, &a.ReceiversThreshold, &receivers, &a.Extra, &a.State, &a.CreatedAt, &a.UpdatedAt, &signers, &a.SignedBy, &a.SignedTx, &a.TraceId, &a.AppId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	a.Keys = strings.Split(keys, ",")
	a.Senders = strings.Split(senders, ",")
	a.Receivers = strings.Split(receivers, ",")
	a.Signers = strings.Split(signers, ",")
	return &a, err
}

func (a *Action) ToOutput() *UnifiedOutput {
	hash, err := crypto.HashFromString(a.TransactionHash)
	if err != nil {
		panic(err)
	}
	return &UnifiedOutput{
		OutputId:           a.OutputId,
		RequestId:          a.RequestId,
		TransactionHash:    hash,
		OutputIndex:        a.OutputIndex,
		AssetId:            a.AssetId,
		KernelAssetId:      a.KernelAssetId,
		Amount:             a.Amount,
		Mask:               a.Mask,
		Keys:               a.Keys,
		SendersHash:        a.SendersHash,
		SendersThreshold:   a.SendersThreshold,
		Senders:            a.Senders,
		ReceiversHash:      a.ReceiversHash,
		ReceiversThreshold: a.ReceiversThreshold,
		Receivers:          a.Receivers,
		Extra:              a.Extra,
		State:              a.State,
		Sequence:           a.Sequence,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
		Signers:            a.Signers,
		SignedBy:           a.SignedBy,
	}
}

func ReplayCheck(a *Action, txs1, txs2 []*Transaction, asset1, asset2 string) {
	if asset1 != asset2 {
		err := fmt.Errorf("action %s compaction asset %s => %s", a.OutputId, asset1, asset2)
		panic(err)
	}
	b1 := SerializeTransactions(txs1)
	b2 := SerializeTransactions(txs2)
	if !bytes.Equal(b1, b2) {
		err := fmt.Errorf("action %s serialization %x => %x", a.OutputId, b1, b2)
		panic(err)
	}
}

func (grp *Group) checkCompactionTransaction(ctx context.Context, out *UnifiedOutput) (*Transaction, bool) {
	if out.RequestId == "" {
		d, err := grp.readOutputDepositUntilSufficient(ctx, out.OutputId)
		if err != nil {
			panic(err)
		}
		appId := grp.FindAppByEntry(DepositEntry{
			Destination: d.Destination,
			Tag:         d.Tag,
		}.UniqueKey())
		if appId != out.AppId {
			panic(out.OutputId)
		}
		return nil, false
	}

	appId, err := grp.checkMTGTransaction(ctx, out.TransactionHash.String())
	if err != nil {
		panic(err)
	}
	if appId == "" {
		return nil, false
	}
	tx, err := grp.store.ReadTransactionByHash(ctx, out.TransactionHash)
	if err != nil {
		panic(err)
	}
	return tx, true
}

// actions queue is all the utxos ordered by their sequence
func (grp *Group) handleActionsQueue(ctx context.Context) error {
	as, err := grp.store.ListActions(ctx, ActionStateInitial, 16)
	if err != nil {
		return fmt.Errorf("store.ListInitialActions() => %v", err)
	}
	for _, a := range as {
		tx, isMTG := grp.checkCompactionTransaction(ctx, a.ToOutput())
		if isMTG && tx == nil {
			return nil
		}
		if tx != nil && tx.compaction {
			return grp.store.RestoreAction(ctx, a, tx)
		}

		wkr := grp.FindWorker(a.AppId)
		if wkr == nil {
			err = grp.store.FinishAction(ctx, a.OutputId, ActionStateDone, nil)
			if err != nil {
				return fmt.Errorf("store.FinishAction(%s) => %v", a.OutputId, err)
			}
			continue
		}

		if a.restoreSequence > a.Sequence {
			a.Sequence = a.restoreSequence
		}
		txs, compactionAsset := wkr.ProcessOutput(ctx, a)
		if grp.debug {
			txs2, compactionAsset2 := wkr.ProcessOutput(ctx, a)
			ReplayCheck(a, txs, txs2, compactionAsset, compactionAsset2)
		}

		state := ActionStateDone
		if compactionAsset != "" && len(txs) == 0 {
			t, err := grp.buildCompactionTransaction(ctx, compactionAsset, a)
			if err != nil {
				return fmt.Errorf("group.buildCompactionTransaction(%s %v) => %v", compactionAsset, a, err)
			}
			state = ActionStateRestorable
			txs = []*Transaction{t}
		} else if compactionAsset != "" {
			return fmt.Errorf("invalid compactionAsset: %s", compactionAsset)
		}

		err = grp.checkTransactions(ctx, a, txs)
		if err != nil {
			return fmt.Errorf("group.checkTransactions(%v) => %v", a, err)
		}

		err = grp.store.FinishAction(ctx, a.OutputId, state, txs)
		if err != nil {
			return fmt.Errorf("store.FinishAction(%s %d) => %v", a.OutputId, state, err)
		}
	}
	return nil
}

func (grp *Group) checkTransactions(ctx context.Context, act *Action, txs []*Transaction) error {
	totalAmount := make(map[string]common.Integer)
	for _, t := range txs {
		t.AppId = act.AppId
		t.Sequence = act.Sequence

		err := t.check(ctx, act)
		if err != nil {
			return err
		}

		amount, ok := totalAmount[t.AssetId]
		if !ok {
			amount = common.NewInteger(0)
		}
		totalAmount[t.AssetId] = amount.Add(common.NewIntegerFromString(t.Amount))
	}

	for asset, amount := range totalAmount {
		outputs := grp.ListOutputsForAsset(ctx, act.AppId, asset, act.Sequence, SafeUtxoStateUnspent, 0)
		total := common.NewInteger(0)
		for _, os := range outputs {
			total = total.Add(common.NewIntegerFromString(os.Amount.String()))
		}
		if total.Cmp(amount) < 0 {
			return fmt.Errorf("insufficient balance for asset %s: %s %s", asset, total.String(), amount.String())
		}
	}
	return nil
}

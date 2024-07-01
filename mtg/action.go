package mtg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/MixinNetwork/mixin/common"
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

// actions queue is all the utxos ordered by their sequence
func (grp *Group) handleActionsQueue(ctx context.Context) error {
	as, err := grp.store.ListActions(ctx, ActionStateInitial, 16)
	if err != nil {
		return fmt.Errorf("store.ListInitialActions() => %v", err)
	}
	for _, a := range as {
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
		state := ActionStateDone
		txs, compactionAsset := wkr.ProcessOutput(ctx, a)
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
		outputs, err := grp.ListOutputsForAsset(ctx, act.AppId, asset, act.Sequence, SafeUtxoStateUnspent, 0)
		if err != nil {
			return err
		}
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

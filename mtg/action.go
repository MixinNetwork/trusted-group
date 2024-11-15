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
	ActionState     ActionState
	restoreSequence uint64

	UnifiedOutput
	group    *Group
	consumed map[string]uint64
}

var actionCols = []string{"output_id", "transaction_hash", "action_state", "sequence", "restore_sequence"}

var actionJoinCols = []string{"actions.output_id", "actions.transaction_hash", "action_state", "actions.sequence", "restore_sequence", "request_id", "output_index", "asset_id", "kernel_asset_id", "amount", "senders_threshold", "senders", "receivers_threshold", "extra", "state", "created_at", "updated_at", "signers", "signed_by", "trace_id", "app_id"}

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
	var senders, signers string
	err := row.Scan(&a.OutputId, &a.TransactionHash, &a.ActionState, &a.Sequence, &a.restoreSequence, &a.TransactionRequestId, &a.OutputIndex, &a.AssetId, &a.KernelAssetId, &a.Amount, &a.SendersThreshold, &senders, &a.ReceiversThreshold, &a.Extra, &a.State, &a.SequencerCreatedAt, &a.updatedAt, &signers, &a.SignedBy, &a.TraceId, &a.AppId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	a.Senders = strings.Split(senders, ",")
	a.Signers = strings.Split(signers, ",")
	return &a, err
}

func (a *Action) TestAttachActionToGroup(g *Group) {
	a.group = g
	a.consumed = make(map[string]uint64)
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

func (grp *Group) checkCompactionTransaction(ctx context.Context, action *Action) (*Transaction, bool) {
	ver, err := grp.ReadKernelTransactionUntilSufficient(ctx, action.TransactionHash)
	if err != nil {
		panic(err)
	}
	if ver.DepositData() != nil {
		d, err := grp.readOutputDepositUntilSufficient(ctx, action.OutputId)
		if err != nil {
			panic(err)
		}
		appId := grp.FindAppByEntry(DepositEntry{
			Destination: d.Destination,
			Tag:         d.Tag,
		}.UniqueKey())
		if appId == "" {
			appId = grp.GroupId
		}
		if appId != action.AppId {
			panic(action.OutputId)
		}
		return nil, false
	}
	appId, _ := DecodeMixinExtraHEX(action.Extra)
	if appId == "" {
		appId = grp.GroupId
	}
	if appId != action.AppId {
		panic(action.OutputId)
	}

	appId, err = grp.checkMTGTransaction(ctx, ver)
	if err != nil {
		panic(err)
	}
	if appId == "" {
		return nil, false
	}
	hash, err := crypto.HashFromString(action.TransactionHash)
	if err != nil {
		panic(err)
	}
	tx, err := grp.store.ReadTransactionByHash(ctx, hash)
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
		tx, isMTG := grp.checkCompactionTransaction(ctx, a)
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
		a.group = grp
		a.consumed = make(map[string]uint64)
		txs, compactionAsset := wkr.ProcessOutput(ctx, a)
		if grp.debug {
			a.consumed = make(map[string]uint64)
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

		err = a.attachTxsConsumed(ctx, txs)
		if err != nil {
			return fmt.Errorf("group.attachTxsConsumed(%v) => %v", a, err)
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
		outputs := grp.ListOutputsForAsset(ctx, act.AppId, asset, 0, act.Sequence, SafeUtxoStateUnspent, 0)
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

func (action *Action) attachTxsConsumed(ctx context.Context, txs []*Transaction) error {
	for _, tx := range txs {
		if len(tx.consumedIds) == 0 {
			panic(fmt.Sprintf("tx %s has empty consumedIds", tx.TraceId))
		}
		if len(tx.consumed) > 0 {
			if len(tx.consumed) != len(tx.consumedIds) {
				panic(tx.TraceId)
			}
			continue
		}
		outputs, err := action.group.store.listOutputs(ctx, tx.consumedIds)
		if err != nil {
			return err
		}
		for _, o := range outputs {
			if o.State != SafeUtxoStateUnspent {
				panic(fmt.Sprintf("invalid output %s state %s for tx %s", o.OutputId, o.State, tx.TraceId))
			}
			if o.Sequence <= action.Sequence && o.Sequence >= action.consumed[tx.AssetId] {
				action.consumed[tx.AssetId] = o.Sequence
			} else {
				panic(fmt.Sprintf("invalid outputs sequence %d for action sequence %d or asset %s consumed %d", o.Sequence, action.Sequence, tx.AssetId, action.consumed[tx.AssetId]))
			}
		}
		tx.consumed = outputs
	}
	return nil
}

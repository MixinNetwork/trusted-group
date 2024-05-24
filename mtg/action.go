package mtg

import (
	"context"
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
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

	RequestId          string
	OutputIndex        int
	AssetId            string
	KernelAssetId      string
	Amount             decimal.Decimal
	Mask               string
	Keys               string
	SendersThreshold   int64
	Senders            string
	ReceiversThreshold int64
	Receivers          string
	Extra              string
	State              SafeUtxoState
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Signers            string
	SignedBy           string
	SignedTx           string
	TraceId            string
	AppId              string
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
	err := row.Scan(&a.OutputId, &a.TransactionHash, &a.ActionState, &a.Sequence, &a.restoreSequence, &a.RequestId, &a.OutputIndex, &a.AssetId, &a.KernelAssetId, &a.Amount, &a.Mask, &a.Keys, &a.SendersThreshold, &a.Senders, &a.ReceiversThreshold, &a.Receivers, &a.Extra, &a.State, &a.CreatedAt, &a.UpdatedAt, &a.Signers, &a.SignedBy, &a.SignedTx, &a.TraceId, &a.AppId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

// actions queue is all the utxos ordered by their sequence
func (grp *Group) handleActionsQueue(ctx context.Context) error {
	as, err := grp.store.ListActions(ctx, ActionStateInitial, 16)
	if err != nil {
		return err
	}
	for _, a := range as {
		wkr := grp.FindWorker(a.AppId)
		if wkr == nil {
			err = grp.store.finishAction(ctx, a.OutputId, ActionStateDone, nil)
			if err != nil {
				panic(err)
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
				return err
			}
			state = ActionStateRestorable
			txs = []*Transaction{t}
		} else if compactionAsset != "" {
			panic(compactionAsset)
		}

		// TODO should check the asset balance all together
		for _, t := range txs {
			err = t.check(ctx)
			if err != nil {
				panic(err)
			}
		}

		err = grp.store.finishAction(ctx, a.OutputId, state, txs)
		if err != nil {
			panic(err)
		}
	}
	return nil
}

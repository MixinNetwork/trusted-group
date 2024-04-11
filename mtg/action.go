package mtg

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/shopspring/decimal"
)

const (
	ActionStateInitial     ActionState = 10
	ActionStateDone        ActionState = 11
	ActionStateRrestorable ActionState = 12
)

type ActionState int

type Action struct {
	OutputId           string
	RequestId          string
	TransactionHash    string
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
	Sequence           uint64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Signers            string
	SignedBy           string

	TraceId         string
	GroupId         string
	ActionState     ActionState
	SignedTx        string
	RestoreSequence uint64
}

func (a *Action) Output() *UnifiedOutput {
	h, err := crypto.HashFromString(a.TransactionHash)
	if err != nil {
		panic(err)
	}
	return &UnifiedOutput{
		OutputId:           a.OutputId,
		RequestId:          a.RequestId,
		TransactionHash:    h,
		OutputIndex:        a.OutputIndex,
		AssetId:            a.AssetId,
		KernelAssetId:      a.KernelAssetId,
		Amount:             a.Amount,
		Mask:               a.Mask,
		Keys:               strings.Split(a.Keys, ","),
		SendersThreshold:   a.SendersThreshold,
		Senders:            strings.Split(a.Senders, ","),
		ReceiversThreshold: a.ReceiversThreshold,
		Receivers:          strings.Split(a.Receivers, ","),
		Extra:              a.Extra,
		State:              a.State,
		Sequence:           a.Sequence,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
		Signers:            strings.Split(a.Signers, ","),
		SignedBy:           a.SignedBy,
	}
}

var actionCols = []string{"output_id", "request_id", "transaction_hash", "output_index", "asset_id", "kernel_asset_id", "amount", "mask", "keys", "senders_threshold", "senders", "receivers_threshold", "receivers", "extra", "state", "sequence", "created_at", "updated_at", "signers", "signed_by", "trace_id", "group_id", "action_state", "signed_tx", "restore_sequence"}

func (a *Action) values() []any {
	return []any{a.OutputId, a.RequestId, a.TransactionHash, a.OutputIndex, a.AssetId, a.KernelAssetId, a.Amount, a.Mask, a.Keys, a.SendersThreshold, a.Senders, a.ReceiversThreshold, a.Receivers, a.Extra, a.State, a.Sequence, a.CreatedAt, a.UpdatedAt, a.Signers, a.SignedBy, a.TraceId, a.GroupId, a.ActionState, a.SignedTx, a.RestoreSequence}
}

func actionFromRow(row *sql.Row) (*Action, error) {
	var a Action
	err := row.Scan(&a.OutputId, &a.RequestId, &a.TransactionHash, &a.OutputIndex, &a.AssetId, &a.KernelAssetId, &a.Amount, &a.Mask, &a.Keys, &a.SendersThreshold, &a.Senders, &a.ReceiversThreshold, &a.Receivers, &a.Extra, &a.State, &a.Sequence, &a.CreatedAt, &a.UpdatedAt, &a.Signers, &a.SignedBy, &a.TraceId, &a.GroupId, &a.ActionState, &a.SignedTx, &a.RestoreSequence)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

// actions queue is all the utxos ordered by their created time
// this queue can only be queried after all utxos are drained from api
func (grp *Group) handleActionsQueue(ctx context.Context) error {
	as, err := grp.store.ListActions(ctx, 16)
	if err != nil {
		return err
	}
	for _, a := range as {
		var totalTs []*Transaction
		for _, wkr := range grp.workers {
			ts := wkr.ProcessOutput(ctx, a)
			totalTs = append(totalTs, ts...)
		}

		hasCompaction := false
		for _, t := range totalTs {
			err = t.check()
			if err != nil {
				panic(err)
			}
			t.ActionId = a.OutputId
			t.Sequence = a.Sequence

			if a.RestoreSequence > 0 {
				t.Sequence = a.RestoreSequence
			}
			if t.Memo == CompactionTransactionMemo {
				hasCompaction = true
				totalTs = []*Transaction{t}
				break
			}
		}
		if !hasCompaction {
			a.ActionState = ActionStateDone
		}
		err = grp.store.writeAction(ctx, a, totalTs)
		if err != nil {
			panic(err)
		}
	}
	return nil
}

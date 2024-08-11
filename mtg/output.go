package mtg

import (
	"database/sql"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/shopspring/decimal"
)

const (
	OutputTypeSafeOutput = "kernel_output"

	SafeUtxoStateUnspent SafeUtxoState = "unspent"
	SafeUtxoStateSigned  SafeUtxoState = "signed"
	SafeUtxoStateSpent   SafeUtxoState = "spent"
)

type SafeUtxoState string

type UnifiedOutput struct {
	Type               string          `json:"type"`
	OutputId           string          `json:"output_id"`
	requestId          string          `json:"request_id,omitempty"`
	TransactionHash    crypto.Hash     `json:"transaction_hash"`
	OutputIndex        int             `json:"output_index"`
	AssetId            string          `json:"asset_id"`
	KernelAssetId      string          `json:"kernel_asset_id"`
	Amount             decimal.Decimal `json:"amount"`
	Mask               string          `json:"mask"`
	Keys               []string        `json:"keys"`
	SendersHash        string          `json:"senders_hash"`
	SendersThreshold   int64           `json:"senders_threshold"`
	Senders            []string        `json:"senders"`
	ReceiversHash      string          `json:"receivers_hash"`
	ReceiversThreshold int64           `json:"receivers_threshold"`
	Receivers          []string        `json:"receivers"`
	Extra              string          `json:"extra"`
	State              SafeUtxoState   `json:"state"`
	Sequence           uint64          `json:"sequence"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	Signers            []string        `json:"signers"`
	SignedBy           string          `json:"signed_by"`

	SignedTx string
	TraceId  string
	AppId    string
}

var outputCols = []string{"output_id", "request_id", "transaction_hash", "output_index", "asset_id", "kernel_asset_id", "amount", "mask", "keys", "senders_threshold", "senders", "receivers_threshold", "receivers", "extra", "state", "sequence", "created_at", "updated_at", "signers", "signed_by", "signed_tx", "trace_id", "app_id"}

func (o *UnifiedOutput) values() []any {
	return []any{o.OutputId, o.requestId, o.TransactionHash.String(), o.OutputIndex, o.AssetId, o.KernelAssetId, o.Amount, o.Mask, strings.Join(o.Keys, ","), o.SendersThreshold, strings.Join(o.Senders, ","), o.ReceiversThreshold, strings.Join(o.Receivers, ","), o.Extra, o.State, o.Sequence, o.CreatedAt, o.UpdatedAt, strings.Join(o.Signers, ","), o.SignedBy, o.SignedTx, o.TraceId, o.AppId}
}

func outputFromRow(row Row) (*UnifiedOutput, error) {
	var o UnifiedOutput
	var hash, keys, senders, receivers, signers string
	err := row.Scan(&o.OutputId, &o.requestId, &hash, &o.OutputIndex, &o.AssetId, &o.KernelAssetId, &o.Amount, &o.Mask, &keys, &o.SendersThreshold, &senders, &o.ReceiversThreshold, &receivers, &o.Extra, &o.State, &o.Sequence, &o.CreatedAt, &o.UpdatedAt, &signers, &o.SignedBy, &o.SignedTx, &o.TraceId, &o.AppId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	h, _ := crypto.HashFromString(hash)
	o.TransactionHash = h
	o.Keys = strings.Split(keys, ",")
	o.Senders = strings.Split(senders, ",")
	o.Receivers = strings.Split(receivers, ",")
	o.Signers = strings.Split(signers, ",")
	return &o, err
}

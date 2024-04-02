package mtg

import (
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

const (
	OutputStateUnspent = 10
	OutputStateSigned  = 11
	OutputStateSpent   = 12

	OutputTypeSafeOutput = "kernel_output"

	SafeUtxoStateUnspent SafeUtxoState = "unspent"
	SafeUtxoStateSigned  SafeUtxoState = "signed"
	SafeUtxoStateSpent   SafeUtxoState = "spent"
)

type SafeUtxoState string

type UnifiedOutput struct {
	Type               string          `json:"type"`
	OutputId           string          `json:"output_id"`
	RequestId          string          `json:"request_id,omitempty"`
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
}

func (o *UnifiedOutput) OutputState() int {
	switch o.State {
	case SafeUtxoStateUnspent:
		return OutputStateUnspent
	case SafeUtxoStateSigned:
		return OutputStateSigned
	case SafeUtxoStateSpent:
		return OutputStateSpent
	}
	panic(o.State)
}

func (o *UnifiedOutput) Output() *Output {
	return &Output{
		UTXOID:           o.OutputId,
		TraceID:          o.RequestId,
		AssetID:          o.AssetId,
		TransactionHash:  o.TransactionHash,
		OutputIndex:      o.OutputIndex,
		Amount:           o.Amount,
		SendersThreshold: o.SendersThreshold,
		Senders:          o.Senders,
		Threshold:        o.ReceiversThreshold,
		Members:          o.Receivers,
		Memo:             o.Extra,
		State:            o.OutputState(),
		Sequence:         o.Sequence,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
		SignedBy:         o.SignedBy,
	}
}

type Output struct {
	GroupId          string
	UTXOID           string
	TraceID          string
	AssetID          string
	TransactionHash  crypto.Hash
	OutputIndex      int
	Amount           decimal.Decimal
	SendersThreshold int64
	Senders          []string
	Threshold        int64
	Members          []string
	Memo             string
	State            int
	Sequence         uint64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	SignedBy         string
}

func (out *Output) StateName() SafeUtxoState {
	switch out.State {
	case OutputStateUnspent:
		return SafeUtxoStateUnspent
	case OutputStateSigned:
		return SafeUtxoStateSigned
	case OutputStateSpent:
		return SafeUtxoStateSpent
	}
	panic(out.State)
}

func (o *Output) Unified() *UnifiedOutput {
	var kernelAssetId string
	if uuid.FromStringOrNil(o.AssetID).String() == o.AssetID {
		kernelAssetId = crypto.Sha256Hash([]byte(o.AssetID)).String()
	}
	return &UnifiedOutput{
		Type:               OutputTypeSafeOutput,
		OutputId:           o.UTXOID,
		RequestId:          o.TraceID,
		AssetId:            o.AssetID,
		KernelAssetId:      kernelAssetId,
		TransactionHash:    o.TransactionHash,
		OutputIndex:        o.OutputIndex,
		Amount:             o.Amount,
		SendersThreshold:   o.SendersThreshold,
		Senders:            o.Senders,
		ReceiversThreshold: o.Threshold,
		Receivers:          o.Members,
		Extra:              o.Memo,
		State:              o.StateName(),
		Sequence:           o.Sequence,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
		SignedBy:           o.SignedBy,
	}
}

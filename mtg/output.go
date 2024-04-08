package mtg

import (
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

func (o *UnifiedOutput) Action(state ActionState, traceId, groupId string) *Action {
	return &Action{
		OutputId:           o.OutputId,
		RequestId:          o.RequestId,
		TransactionHash:    o.TransactionHash.String(),
		OutputIndex:        o.OutputIndex,
		AssetId:            o.AssetId,
		KernelAssetId:      o.KernelAssetId,
		Amount:             o.Amount,
		Mask:               o.Mask,
		Keys:               strings.Join(o.Keys, ","),
		SendersThreshold:   o.SendersThreshold,
		Senders:            strings.Join(o.Senders, ","),
		ReceiversThreshold: o.ReceiversThreshold,
		Receivers:          strings.Join(o.Receivers, ","),
		Extra:              o.Extra,
		State:              o.State,
		Sequence:           o.Sequence,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
		Signers:            strings.Join(o.Signers, ","),
		SignedBy:           o.SignedBy,

		TraceId:     traceId,
		GroupId:     groupId,
		ActionState: state,
	}
}

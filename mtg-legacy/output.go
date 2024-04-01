package mtg

import (
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

const (
	OutputStateUnspent = 10
	OutputStateSigned  = 11
	OutputStateSpent   = 12

	OutputTypeMultisig    = "multisig_utxo"
	OutputTypeCollectible = "non_fungible_output"
)

type UnifiedOutput struct {
	Type            string          `json:"type"`
	UserId          string          `json:"user_id"`
	TransactionHash crypto.Hash     `json:"transaction_hash"`
	OutputIndex     int             `json:"output_index"`
	Amount          decimal.Decimal `json:"amount"`
	Memo            string          `json:"memo"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	SignedBy        string          `json:"signed_by"`
	SignedTx        string          `json:"signed_tx"`
	State           string          `json:"state"`

	UnifiedOutputId           string   `json:"output_id"`
	UnifiedTokenId            string   `json:"token_id"`
	UnifiedSendersThreshold   int64    `json:"senders_threshold"`
	UnifiedSenders            []string `json:"senders"`
	UnifiedReceiversThreshold int64    `json:"receivers_threshold"`
	UnifiedReceivers          []string `json:"receivers"`

	UnifiedUTXOID    string   `json:"utxo_id"`
	UnifiedAssetId   string   `json:"asset_id"`
	UnifiedThreshold int64    `json:"threshold"`
	UnifiedMembers   []string `json:"members"`
	UnifiedSender    string   `json:"sender"`
}

type Output struct {
	GroupId         string
	UserID          string
	UTXOID          string
	AssetID         string
	TransactionHash crypto.Hash
	OutputIndex     int
	Sender          string
	Amount          decimal.Decimal
	Threshold       uint8
	Members         []string
	Memo            string
	State           int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	SignedBy        string
	SignedTx        string
}

func (out *Output) StateName() string {
	switch out.State {
	case OutputStateUnspent:
		return mixin.UTXOStateUnspent
	case OutputStateSigned:
		return mixin.UTXOStateSigned
	case OutputStateSpent:
		return mixin.UTXOStateSpent
	}
	panic(out.State)
}

func (o *Output) Unified() *UnifiedOutput {
	return &UnifiedOutput{
		Type:             OutputTypeMultisig,
		UserId:           o.UserID,
		TransactionHash:  o.TransactionHash,
		OutputIndex:      o.OutputIndex,
		Amount:           o.Amount,
		Memo:             o.Memo,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
		SignedBy:         o.SignedBy,
		SignedTx:         o.SignedTx,
		State:            o.StateName(),
		UnifiedUTXOID:    o.UTXOID,
		UnifiedAssetId:   o.AssetID,
		UnifiedSender:    o.Sender,
		UnifiedThreshold: int64(o.Threshold),
		UnifiedMembers:   o.Members,
	}
}

func (o *UnifiedOutput) UniqueId() string {
	switch o.Type {
	case OutputTypeMultisig:
		return o.UnifiedUTXOID
	case OutputTypeCollectible:
		return o.UnifiedOutputId
	}
	panic(o.Type)
}

func (o *UnifiedOutput) AsMultisig() *Output {
	if o.Type != OutputTypeMultisig {
		panic(o.Type)
	}
	out := &Output{
		UserID:          o.UserId,
		UTXOID:          o.UnifiedUTXOID,
		AssetID:         o.UnifiedAssetId,
		TransactionHash: o.TransactionHash,
		OutputIndex:     o.OutputIndex,
		Sender:          o.UnifiedSender,
		Amount:          o.Amount,
		Threshold:       uint8(o.UnifiedThreshold),
		Members:         o.UnifiedMembers,
		Memo:            o.Memo,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
		SignedBy:        o.SignedBy,
		SignedTx:        o.SignedTx,
	}
	switch o.State {
	case mixin.UTXOStateUnspent:
		out.State = OutputStateUnspent
	case mixin.UTXOStateSigned:
		out.State = OutputStateSigned
	case mixin.UTXOStateSpent:
		out.State = OutputStateSpent
	}
	return out
}

func (o *UnifiedOutput) AsCollectible() *CollectibleOutput {
	if o.Type != OutputTypeCollectible {
		panic(o.Type)
	}
	out := &CollectibleOutput{
		Type:               o.Type,
		UserId:             o.UserId,
		OutputId:           o.UnifiedOutputId,
		TokenId:            o.UnifiedTokenId,
		TransactionHash:    o.TransactionHash,
		OutputIndex:        o.OutputIndex,
		Amount:             o.Amount,
		SendersThreshold:   o.UnifiedSendersThreshold,
		Senders:            o.UnifiedSenders,
		ReceiversThreshold: o.UnifiedReceiversThreshold,
		Receivers:          o.UnifiedReceivers,
		Memo:               o.Memo,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
		SignedBy:           o.SignedBy,
		SignedTx:           o.SignedTx,
	}
	switch o.State {
	case mixin.UTXOStateUnspent:
		out.State = OutputStateUnspent
	case mixin.UTXOStateSigned:
		out.State = OutputStateSigned
	case mixin.UTXOStateSpent:
		out.State = OutputStateSpent
	}
	return out
}

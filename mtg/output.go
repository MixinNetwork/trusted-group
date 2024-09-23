package mtg

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

const (
	OutputTypeSafeOutput = "kernel_output"

	SafeUtxoStateUnspent  SafeUtxoState = "unspent"
	SafeUtxoStateAssigned SafeUtxoState = "assigned"
	SafeUtxoStateSigned   SafeUtxoState = "signed"
	SafeUtxoStateSpent    SafeUtxoState = "spent"
)

type SafeUtxoState string

type UnifiedOutput struct {
	Type                 string          `json:"type"`
	OutputId             string          `json:"output_id"`
	TransactionRequestId string          `json:"request_id,omitempty"`
	TransactionHash      string          `json:"transaction_hash"`
	OutputIndex          int             `json:"output_index"`
	AssetId              string          `json:"asset_id"`
	KernelAssetId        string          `json:"kernel_asset_id"`
	Amount               decimal.Decimal `json:"amount"`
	SendersHash          string          `json:"senders_hash"`
	SendersThreshold     int64           `json:"senders_threshold"`
	Senders              []string        `json:"senders"`
	ReceiversHash        string          `json:"receivers_hash"`
	ReceiversThreshold   int64           `json:"receivers_threshold"`
	Extra                string          `json:"extra"`
	State                SafeUtxoState   `json:"state"`
	Sequence             uint64          `json:"sequence"`
	Signers              []string        `json:"signers"`
	SignedBy             string          `json:"signed_by"`
	SequencerCreatedAt   time.Time       `json:"created_at"`

	updatedAt time.Time
	TraceId   string
	AppId     string
}

var outputCols = []string{"output_id", "request_id", "transaction_hash", "output_index", "asset_id", "kernel_asset_id", "amount", "senders_threshold", "senders", "receivers_threshold", "extra", "state", "sequence", "created_at", "updated_at", "signers", "signed_by", "trace_id", "app_id"}

func (o *UnifiedOutput) values() []any {
	return []any{o.OutputId, o.TransactionRequestId, o.TransactionHash, o.OutputIndex, o.AssetId, o.KernelAssetId, o.Amount, o.SendersThreshold, strings.Join(o.Senders, ","), o.ReceiversThreshold, o.Extra, o.State, o.Sequence, o.SequencerCreatedAt, o.updatedAt, strings.Join(o.Signers, ","), o.SignedBy, o.TraceId, o.AppId}
}

func outputFromRow(row Row) (*UnifiedOutput, error) {
	var o UnifiedOutput
	var senders, signers string
	err := row.Scan(&o.OutputId, &o.TransactionRequestId, &o.TransactionHash, &o.OutputIndex, &o.AssetId, &o.KernelAssetId, &o.Amount, &o.SendersThreshold, &senders, &o.ReceiversThreshold, &o.Extra, &o.State, &o.Sequence, &o.SequencerCreatedAt, &o.updatedAt, &signers, &o.SignedBy, &o.TraceId, &o.AppId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	o.Senders = strings.Split(senders, ",")
	o.Signers = strings.Split(signers, ",")
	return &o, err
}

func (o *UnifiedOutput) checkId() bool {
	h := md5.New()
	oid := fmt.Sprintf("%s:%d", o.TransactionHash, o.OutputIndex)
	n, err := io.WriteString(h, oid)
	if err != nil || n != len(oid) {
		panic(err)
	}
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	id, err := uuid.FromBytes(sum)
	if err != nil {
		panic(err)
	}
	return id.String() == o.OutputId
}

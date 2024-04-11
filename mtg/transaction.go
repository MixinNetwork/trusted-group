package mtg

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

const (
	TransactionStateInitial  = 10
	TransactionStateSigning  = 11
	TransactionStateSigned   = 12
	TransactionStateSnapshot = 13

	OutputsBatchSize          = 36
	CompactionTransactionMemo = "COMPACTION"
	StorageAssetId            = "c94ac88f-4671-3976-b60a-09064f1811e8"
	StorageReceiverId         = "773e5e77-4107-45c2-b648-8fc722ed77f5"
)

type Transaction struct {
	GroupId    string
	TraceId    string
	ActionId   string
	State      int
	AssetId    string
	Receivers  []string
	Threshold  int
	Amount     string
	Memo       string
	Raw        []byte
	Hash       crypto.Hash
	References []crypto.Hash
	Sequence   uint64
	UpdatedAt  time.Time
}

var transactionCols = []string{"group_id", "trace_id", "action_id", "state", "asset_id", "receivers", "threshold", "amount", "memo", "raw", "hash", "references", "sequence", "updated_at"}

func (t *Transaction) values() []any {
	return []any{t.GroupId, t.TraceId, t.ActionId, t.State, t.AssetId, t.Receivers, t.Threshold, t.Amount, t.Memo, t.Raw, t.Hash, t.References, t.Sequence, t.UpdatedAt}
}

func transactionFromRow(row *sql.Row) (*Transaction, error) {
	var t Transaction
	err := row.Scan(&t.GroupId, &t.TraceId, &t.ActionId, &t.State, &t.AssetId, &t.Receivers, &t.Threshold, &t.Amount, &t.Memo, &t.Raw, &t.Hash, &t.References, &t.Sequence, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

func (t *Transaction) check() error {
	if len(t.References) > 2 {
		return fmt.Errorf("invalid references length: %d", len(t.References))
	}
	for i, r := range t.References {
		if !r.HasValue() {
			return fmt.Errorf("invalid reference: %d %v", i, r)
		}
	}
	if t.Threshold < 1 || t.Threshold > 128 {
		return fmt.Errorf("invalid receivers threshold %d/%d", t.Threshold, len(t.Receivers))
	}
	amt, err := decimal.NewFromString(t.Amount)
	min, _ := decimal.NewFromString("0.00000001")
	if err != nil || amt.Cmp(min) < 0 {
		return fmt.Errorf("invalid amount %s", t.Amount)
	}

	for _, r := range t.Receivers {
		id, _ := uuid.FromString(r)
		if id.String() == uuid.Nil.String() {
			return fmt.Errorf("invalid receiver %s", r)
		}
	}

	s := encodeMixinExtra(t.GroupId, t.TraceId, t.Memo)
	if len(s) >= common.ExtraSizeGeneralLimit {
		return fmt.Errorf("invalid extra length: %d", len(s))
	}

	return nil
}

func (grp *Group) buildCompactTransaction(ctx context.Context, source *Transaction, outputs []*Action) error {
	var total common.Integer
	traceId := mixin.UniqueConversationID(CompactionTransactionMemo, source.TraceId)
	for _, out := range outputs {
		// if out.GroupId != source.GroupId {
		// 	panic(source)
		// }
		total = total.Add(common.NewIntegerFromString(out.Amount.String()))
		traceId = mixin.UniqueConversationID(traceId, out.OutputId)
	}
	newTx := &Transaction{
		GroupId:   source.GroupId,
		TraceId:   traceId,
		State:     TransactionStateInitial,
		AssetId:   source.AssetId,
		Receivers: grp.GetMembers(),
		Threshold: grp.GetThreshold(),
		Amount:    total.String(),
		Memo:      CompactionTransactionMemo,
	}
	logger.Printf("Group.buildCompactTransaction(%s, %s, %s) => %s\n", source.GroupId, source.TraceId, total, traceId)
	// TODO write compact tx and delete all initial txs related to the action
	return grp.writeCompactTansactionOrPanic(ctx, newTx)
}

func (grp *Group) checkCompactTransaction(ver *common.VersionedTransaction, extra *mixinExtraPack) bool {
	// FIXME should check the keys with messenger api
	return ver != nil && extra != nil && extra.M == CompactionTransactionMemo && len(ver.Inputs) == OutputsBatchSize &&
		len(ver.Outputs) == 1 && len(ver.Outputs[0].Keys) == len(grp.GetMembers()) &&
		ver.Outputs[0].Script.String() == common.NewThresholdScript(uint8(grp.GetThreshold())).String()
}

func (grp *Group) signTransaction(ctx context.Context, tx *Transaction) ([]byte, bool, error) {
	outputs, err := grp.ListOutputsForTransaction(ctx, tx)
	if err != nil {
		panic(err)
	}
	if len(outputs) == 0 {
		outputs, err = grp.ListOutputsForAsset(ctx, tx, SafeUtxoStateUnspent, OutputsBatchSize)
	}
	if err != nil {
		panic(err)
	}
	logger.Verbosef("group.ListOutputsForTransaction(%s) => %d %v\n", tx.TraceId, len(outputs), err)
	if len(outputs) == 0 {
		return nil, false, fmt.Errorf("empty outputs %s", tx.Amount)
	}
	// FIXME do more compatcion transaction check
	if len(outputs) < OutputsBatchSize && tx.Memo == CompactionTransactionMemo {
		return nil, false, fmt.Errorf("insufficient compaction transaction outputs %v %d", tx, len(outputs))
	}

	ver, outputs, err := grp.buildRawTransaction(ctx, tx, outputs)
	logger.Verbosef("group.buildRawTransaction(%v) => %v %d %v\n", tx, ver, len(outputs), err)
	if err != nil {
		return nil, false, err
	}
	// FIXME do more compatcion transaction check
	if len(ver.Outputs) != 1 && tx.Memo == CompactionTransactionMemo {
		return nil, false, fmt.Errorf("expired compaction transaction %v", tx)
	}
	if ver.AggregatedSignature != nil || len(ver.SignaturesMap) > 0 {
		return ver.Marshal(), false, nil
	}
	isCompaction := false
	if tx.Memo == CompactionTransactionMemo {
		isCompaction = true
	}

	raw := hex.EncodeToString(ver.Marshal())
	req, err := grp.createMultisigUntilSufficient(ctx, tx.TraceId, raw)
	if err != nil {
		return nil, false, err
	}
	req, err = grp.signMultisigUntilSufficient(ctx, req)
	if err != nil {
		return nil, false, err
	}

	for _, out := range outputs {
		out.State = SafeUtxoStateSigned
		out.SignedBy = ver.PayloadHash().String()
		out.SignedTx = req.RawTransaction
	}

	rb, err := hex.DecodeString(req.RawTransaction)
	if err != nil {
		return nil, false, err
	}
	ver, _ = common.UnmarshalVersionedTransaction(rb)
	tx.Raw = rb
	tx.Hash = ver.PayloadHash()
	tx.UpdatedAt = time.Now()
	tx.State = TransactionStateSigning

	if ver.AggregatedSignature != nil || len(ver.SignaturesMap) > 0 {
		tx.State = TransactionStateSigned
	} else {
		p := DecodeMixinExtra(string(ver.Extra))
		if p.T.String() != tx.TraceId {
			panic(hex.EncodeToString(rb))
		}
		if p.G != tx.GroupId {
			panic(hex.EncodeToString(rb))
		}
	}

	err = grp.store.updateTxWithOutputs(ctx, tx, outputs)
	if err != nil {
		panic(err)
	}
	return rb, isCompaction, nil
}

func (grp *Group) createMultisigUntilSufficient(ctx context.Context, id, raw string) (*mixin.SafeTransactionRequest, error) {
	for {
		req, err := grp.mixin.SafeCreateTransactionRequest(ctx, &mixin.SafeTransactionRequestInput{
			RequestID:      id,
			RawTransaction: raw,
		})
		logger.Verbosef("group.SafeCreateTransactionRequest(%s, %s) => %v %v\n", id, raw, req, err)
		if err != nil && checkRetryableError(err) {
			time.Sleep(3 * time.Second)
			continue
		}
		return req, err
	}
}

func (grp *Group) signMultisigUntilSufficient(ctx context.Context, input *mixin.SafeTransactionRequest) (*mixin.SafeTransactionRequest, error) {
	key, err := mixinnet.KeyFromString(grp.SpendPrivateKey)
	if err != nil {
		return nil, err
	}
	ver, err := mixinnet.TransactionFromRaw(input.RawTransaction)
	if err != nil {
		return nil, err
	}
	err = mixin.SafeSignTransaction(ver, key, input.Views, uint16(grp.Index()))
	if err != nil {
		return nil, err
	}
	signedRaw, err := ver.Dump()
	if err != nil {
		return nil, err
	}
	for {
		req, err := grp.mixin.SafeSubmitTransactionRequest(ctx, &mixin.SafeTransactionRequestInput{
			RequestID:      input.RequestID,
			RawTransaction: signedRaw,
		})
		logger.Verbosef("group.SafeSignMultisigRequest(%s %s) => %v %v\n", input.RequestID, signedRaw, req, err)
		if err != nil && checkRetryableError(err) {
			time.Sleep(3 * time.Second)
			continue
		}
		return req, err
	}
}

func checkRetryableError(err error) bool {
	es := err.Error()
	switch {
	case strings.Contains(es, "Client.Timeout exceeded"):
	case strings.Contains(es, "Bad Gateway"):
	case strings.Contains(es, "Internal Server Error"):
	default:
		return false
	}
	return true
}

func (grp *Group) buildRawTransaction(ctx context.Context, tx *Transaction, outputs []*Action) (*common.VersionedTransaction, []*Action, error) {
	old, _ := decodeTransactionWithExtra(outputs[0].SignedTx)
	if old != nil && (old.AggregatedSignature != nil || len(old.SignaturesMap) > 0) {
		return old, nil, nil
	}
	ver := common.NewTransactionV5(crypto.Sha256Hash([]byte(tx.AssetId)))
	ver.Extra = []byte(encodeMixinExtra(tx.GroupId, tx.TraceId, tx.Memo))
	target := common.NewIntegerFromString(tx.Amount)

	var total common.Integer
	var consumed []*Action
	for _, out := range outputs {
		hash, err := crypto.HashFromString(out.TransactionHash)
		if err != nil {
			panic(err)
		}
		total = total.Add(common.NewIntegerFromString(out.Amount.String()))
		ver.AddInput(hash, uint(out.OutputIndex))
		consumed = append(consumed, out)
		if total.Cmp(target) >= 0 && len(consumed) >= grp.groupSize {
			break
		}
	}
	if total.Cmp(target) < 0 {
		if len(outputs) == OutputsBatchSize {
			// FIXME with reference
			err := grp.buildCompactTransaction(ctx, tx, outputs)
			if err != nil {
				panic(err)
			}
		}
		return nil, nil, fmt.Errorf("insufficient %d %s %s", len(outputs), total, tx.Amount)
	}

	keys, err := grp.mixin.SafeCreateGhostKeys(ctx, []*mixin.GhostInput{{
		Receivers: tx.Receivers,
		Index:     0,
		Hint:      tx.TraceId,
	}, {
		Receivers: grp.members,
		Index:     1,
		Hint:      tx.TraceId,
	}})
	if err != nil {
		return nil, nil, err
	}

	amount, err := decimal.NewFromString(tx.Amount)
	if err != nil {
		return nil, nil, err
	}
	ver.Outputs = append(ver.Outputs, newCommonOutput(&mixinnet.Output{
		Mask:   keys[0].Mask,
		Keys:   keys[0].Keys,
		Amount: mixinnet.IntegerFromDecimal(amount),
		Script: mixinnet.NewThresholdScript(uint8(tx.Threshold)),
	}))

	if diff := total.Sub(common.NewIntegerFromString(tx.Amount)); diff.Sign() > 0 {
		amount, err := decimal.NewFromString(diff.String())
		if err != nil {
			return nil, nil, err
		}
		ver.Outputs = append(ver.Outputs, newCommonOutput(&mixinnet.Output{
			Mask:   keys[1].Mask,
			Keys:   keys[1].Keys,
			Amount: mixinnet.IntegerFromDecimal(amount),
			Script: mixinnet.NewThresholdScript(uint8(grp.threshold)),
		}))
	}

	ver.References = tx.References
	return ver.AsVersioned(), consumed, nil
}

// all the transactions sent by the MTG is encoded by base64(msgpack(mep))
type mixinExtraPack struct {
	T uuid.UUID
	G string `msgpack:",omitempty"`
	M string `msgpack:",omitempty"`
}

func decodeTransactionWithExtra(s string) (*common.VersionedTransaction, *mixinExtraPack) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil, nil
	}
	tx, err := common.UnmarshalVersionedTransaction(raw)
	if err != nil {
		return nil, nil
	}
	p := DecodeMixinExtra(string(tx.Extra))
	return tx, p
}

func DecodeMixinExtra(memo string) *mixinExtraPack {
	extra, err := hex.DecodeString(memo)
	if err != nil {
		return nil
	}
	var p mixinExtraPack
	err = MsgpackUnmarshal(extra, &p)
	if err != nil || p.T.String() == uuid.Nil.String() {
		return nil
	}
	return &p
}

func encodeMixinExtra(groupId, traceId, memo string) string {
	id, err := uuid.FromString(traceId)
	if err != nil {
		panic(err)
	}
	p := &mixinExtraPack{T: id, G: groupId, M: memo}
	b := MsgpackMarshalPanic(p)
	s := base64.RawURLEncoding.EncodeToString(b)
	return s
}

func EncodeMixinExtra(groupId, traceId, memo string) string {
	s := encodeMixinExtra(groupId, traceId, memo)
	if len(s) >= common.ExtraSizeGeneralLimit {
		panic(memo)
	}
	return s
}

func newCommonOutput(out *mixinnet.Output) *common.Output {
	cout := &common.Output{
		Type:   common.OutputTypeScript,
		Amount: common.NewIntegerFromString(out.Amount.String()),
		Script: common.Script(out.Script),
		Mask:   crypto.Key(out.Mask),
	}
	for _, k := range out.Keys {
		ck := crypto.Key(k)
		cout.Keys = append(cout.Keys, &ck)
	}
	return cout
}

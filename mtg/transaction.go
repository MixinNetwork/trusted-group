package mtg

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

const (
	TransactionStateInitial  = 10
	TransactionStateSigning  = 11
	TransactionStateSigned   = 12
	TransactionStateSnapshot = 13

	OutputsBatchSize = 36
)

type Transaction struct {
	GroupId   string
	TraceId   string
	State     int
	AssetId   string
	Receivers []string
	Threshold int
	Amount    string
	Memo      string
	Raw       []byte
	Hash      crypto.Hash
	UpdatedAt time.Time
}

// the app should decide a unique trace id so that the MTG will not double spend
func (grp *Group) BuildTransaction(ctx context.Context, assetId string, receivers []string, threshold int, amount, memo string, traceId, groupId string) error {
	return grp.buildTransaction(ctx, assetId, receivers, threshold, amount, memo, traceId, groupId, grp.clock.Now())
}

func (grp *Group) buildCompactTransaction(ctx context.Context, source *Transaction, outputs []*Output) error {
	var total common.Integer
	traceId := mixin.UniqueConversationID("COMPACTION", source.TraceId)
	for _, out := range outputs {
		if out.GroupId != source.GroupId {
			panic(source)
		}
		total = total.Add(common.NewIntegerFromString(out.Amount.String()))
		traceId = mixin.UniqueConversationID(traceId, out.UTXOID)
	}
	logger.Printf("Group.buildCompactTransaction(%s, %s, %s) => %s\n", source.GroupId, source.TraceId, total, traceId)
	return grp.buildTransaction(ctx, source.AssetId, grp.GetMembers(), grp.GetThreshold(), total.String(), "COMPACTION", traceId, source.GroupId, time.Unix(0, 0))
}

func (grp *Group) buildTransaction(ctx context.Context, assetId string, receivers []string, threshold int, amount, memo string, traceId, groupId string, ts time.Time) error {
	if threshold <= 0 || threshold > len(receivers) {
		return fmt.Errorf("invalid receivers threshold %d/%d", threshold, len(receivers))
	}
	amt, err := decimal.NewFromString(amount)
	min, _ := decimal.NewFromString("0.00000001")
	if err != nil || amt.Cmp(min) < 0 {
		return fmt.Errorf("invalid amount %s", amount)
	}

	// TODO ensure valid memo and trace id
	EncodeMixinExtra(groupId, traceId, memo)

	for _, r := range receivers {
		id, _ := uuid.FromString(r)
		if id.String() == uuid.Nil.String() {
			return fmt.Errorf("invalid receiver %s", r)
		}
	}
	old, err := grp.store.ReadTransactionByTraceId(traceId)
	if err != nil {
		panic(err)
	} else if old != nil {
		return nil
	}
	tx := &Transaction{
		GroupId:   groupId,
		TraceId:   traceId,
		State:     TransactionStateInitial,
		AssetId:   assetId,
		Receivers: receivers,
		Threshold: threshold,
		Amount:    amount,
		Memo:      memo,
		UpdatedAt: ts,
	}
	err = grp.store.WriteTransaction(tx)
	if err != nil {
		panic(err)
	}
	return nil
}

func (grp *Group) signTransaction(ctx context.Context, tx *Transaction) ([]byte, error) {
	outputs, err := grp.store.ListOutputsForTransaction(tx.TraceId)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		outputs, err = grp.store.ListOutputsForAsset(tx.GroupId, mixin.UTXOStateUnspent, tx.AssetId, OutputsBatchSize)
	}
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		return nil, fmt.Errorf("empty outputs %s", tx.Amount)
	}

	ver, outputs, err := grp.buildRawTransaction(ctx, tx, outputs)
	if err != nil {
		return nil, err
	}
	if ver.AggregatedSignature != nil {
		return ver.Marshal(), nil
	}

	raw := hex.EncodeToString(ver.Marshal())
	req, err := grp.mixin.CreateMultisig(ctx, mixin.MultisigActionSign, raw)
	if err != nil {
		return nil, err
	}

	req, err = grp.mixin.SignMultisig(ctx, req.RequestID, grp.pin)
	if err != nil {
		return nil, err
	}

	for _, out := range outputs {
		out.State = OutputStateSigned
		out.SignedBy = ver.PayloadHash().String()
		out.SignedTx = req.RawTransaction
	}
	err = grp.store.WriteOutputs(outputs, tx.TraceId)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(req.RawTransaction)
}

func (grp *Group) buildRawTransaction(ctx context.Context, tx *Transaction, outputs []*Output) (*common.VersionedTransaction, []*Output, error) {
	old, _ := decodeTransactionWithExtra(outputs[0].SignedTx)
	if old != nil {
		return old, nil, nil
	}
	ver := common.NewTransactionV2(crypto.NewHash([]byte(tx.AssetId)))
	ver.Extra = []byte(EncodeMixinExtra(tx.GroupId, tx.TraceId, tx.Memo))
	target := common.NewIntegerFromString(tx.Amount)

	var total common.Integer
	var consumed []*Output
	for _, out := range outputs {
		total = total.Add(common.NewIntegerFromString(out.Amount.String()))
		ver.AddInput(crypto.Hash(out.TransactionHash), out.OutputIndex)
		consumed = append(consumed, out)
		if total.Cmp(target) >= 0 && len(consumed) >= grp.groupSize {
			break
		}
	}
	if total.Cmp(target) < 0 {
		if len(outputs) == OutputsBatchSize {
			err := grp.buildCompactTransaction(ctx, tx, outputs)
			if err != nil {
				panic(err)
			}
		}
		return nil, nil, fmt.Errorf("insufficient %d %s %s", len(outputs), total, tx.Amount)
	}

	keys, err := grp.mixin.BatchReadGhostKeys(ctx, []*mixin.GhostInput{{
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
	out := keys[0].DumpOutput(uint8(tx.Threshold), amount)
	ver.Outputs = append(ver.Outputs, newCommonOutput(out))

	if diff := total.Sub(common.NewIntegerFromString(tx.Amount)); diff.Sign() > 0 {
		amount, err := decimal.NewFromString(diff.String())
		if err != nil {
			return nil, nil, err
		}
		out := keys[1].DumpOutput(uint8(grp.threshold), amount)
		ver.Outputs = append(ver.Outputs, newCommonOutput(out))
	}

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
	if p == nil {
		return nil, nil
	}
	return tx, p
}

func DecodeMixinExtra(memo string) *mixinExtraPack {
	extra, err := base64.RawURLEncoding.DecodeString(memo)
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

func EncodeMixinExtra(groupId, traceId, memo string) string {
	id, err := uuid.FromString(traceId)
	if err != nil {
		panic(err)
	}
	p := &mixinExtraPack{T: id, G: groupId, M: memo}
	b := MsgpackMarshalPanic(p)
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) >= common.ExtraSizeLimit {
		panic(memo)
	}
	return s
}

func newCommonOutput(out *mixin.Output) *common.Output {
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

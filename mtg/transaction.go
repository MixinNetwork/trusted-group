package mtg

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
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

	OutputsBatchSize  = 36
	StorageAssetId    = "c94ac88f-4671-3976-b60a-09064f1811e8"
	StorageReceiverId = "773e5e77-4107-45c2-b648-8fc722ed77f5"
)

type Transaction struct {
	AppId         string
	OpponentAppId string
	TraceId       string
	State         int
	AssetId       string
	Receivers     []string
	Threshold     int
	Amount        string
	Memo          string
	Raw           []byte
	Hash          crypto.Hash
	References    []crypto.Hash
	Sequence      uint64
	Compaction    bool
	Storage       bool
	UpdatedAt     time.Time
}

var transactionCols = []string{"app_id", "opponent_app_id", "trace_id", "state", "asset_id", "receivers", "threshold", "amount", "memo", "raw", "hash", "refs", "sequence", "compaction", "storage", "updated_at"}

func (t *Transaction) values() []any {
	var refs []string
	for _, r := range t.References {
		refs = append(refs, r.String())
	}
	hash := t.Hash.String()
	if !t.Hash.HasValue() {
		hash = ""
	}
	return []any{t.AppId, t.OpponentAppId, t.TraceId, t.State, t.AssetId, strings.Join(t.Receivers, ","), t.Threshold, t.Amount, t.Memo, t.Raw, hash, strings.Join(refs, ","), t.Sequence, t.Compaction, t.Storage, t.UpdatedAt}
}

func transactionFromRow(row Row) (*Transaction, error) {
	var t Transaction
	var hash, rs, refs string
	err := row.Scan(&t.AppId, &t.OpponentAppId, &t.TraceId, &t.State, &t.AssetId, &rs, &t.Threshold, &t.Amount, &t.Memo, &t.Raw, &hash, &refs, &t.Sequence, &t.Compaction, &t.Storage, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	var h crypto.Hash
	if hash != "" {
		h, err = crypto.HashFromString(hash)
		if err != nil {
			return nil, err
		}
	}
	t.Hash = h

	t.Receivers = strings.Split(rs, ",")
	for _, r := range strings.Split(refs, ",") {
		if r == "" {
			continue
		}
		ref, err := crypto.HashFromString(r)
		if err != nil {
			return nil, err
		}
		t.References = append(t.References, ref)
	}
	return &t, err
}

func (grp *Group) BuildTransaction(traceId, appId, opponentAppId, assetId, amount, memo string, receivers []string, threshold int, sequence uint64) *Transaction {
	sort.Strings(receivers)
	return &Transaction{
		TraceId:       traceId,
		AppId:         appId,
		OpponentAppId: opponentAppId,
		State:         TransactionStateInitial,
		AssetId:       assetId,
		Amount:        amount,
		Receivers:     receivers,
		Threshold:     threshold,
		Memo:          memo,
		Sequence:      sequence,
	}
}

func (grp *Group) BuildTransactionWithReference(traceId, appId, opponentAppId, assetId, amount, memo string, receivers []string, threshold int, sequence uint64, reference crypto.Hash) *Transaction {
	if !reference.HasValue() {
		panic(reference)
	}
	t := grp.BuildTransaction(traceId, appId, opponentAppId, assetId, amount, memo, receivers, threshold, sequence)
	t.References = []crypto.Hash{reference}
	return t
}

func (grp *Group) BuildStorageTransaction(ctx context.Context, extra []byte, sequence uint64) (*Transaction, error) {
	sTraceId := crypto.Blake3Hash(extra).String()
	sTraceId = mixin.UniqueConversationID(sTraceId, sTraceId)
	old, err := grp.store.ReadTransactionByTraceId(ctx, sTraceId)
	if err != nil || old != nil {
		return old, err
	}
	if len(extra) > common.ExtraSizeStorageCapacity {
		return nil, fmt.Errorf("too large extra %d > %d", len(extra), common.ExtraSizeStorageCapacity)
	}

	step := common.NewIntegerFromString(common.ExtraStoragePriceStep)
	amount := step.Mul(len(extra)/common.ExtraSizeStorageStep + 1)
	t := grp.BuildTransaction(sTraceId, "", "", StorageAssetId, amount.String(), string(extra), []string{StorageReceiverId}, 64, sequence)
	t.Storage = true

	total, err := grp.CheckAssetBalanceAt(ctx, grp.GroupId, StorageAssetId, sequence)
	if err != nil {
		return nil, err
	}
	if common.NewIntegerFromString(total.String()).Cmp(amount) < 0 {
		return t, fmt.Errorf("insufficient balance")
	}

	return t, grp.store.writeStorageTransaction(ctx, t)
}

func (t *Transaction) check(ctx context.Context) error {
	// TODO do CheckAssetBalanceAt all these transactions together
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
		if CheckTestEnvironment(ctx) {
			continue
		}
		id, _ := uuid.FromString(r)
		if id.String() == uuid.Nil.String() {
			return fmt.Errorf("invalid receiver %s", r)
		}
	}

	s := encodeMixinExtra(t.OpponentAppId, t.TraceId, t.Memo)
	if len(s) >= common.ExtraSizeGeneralLimit {
		return fmt.Errorf("invalid extra length: %d", len(s))
	}

	return nil
}

func (grp *Group) buildCompactionTransaction(ctx context.Context, asset string, act *Action) (*Transaction, error) {
	compaction, err := grp.ListOutputsForAsset(ctx, act.AppId, asset, act.Sequence, SafeUtxoStateUnspent, OutputsBatchSize)
	if err != nil {
		return nil, err
	}
	total := decimal.NewFromInt(0)
	for _, out := range compaction {
		total = total.Add(out.Amount)
	}

	hash, err := crypto.HashFromString(act.TransactionHash)
	if err != nil {
		return nil, err
	}

	tx := &Transaction{
		TraceId:       mixin.UniqueConversationID(act.OutputId, "compaction"),
		AppId:         act.AppId,
		OpponentAppId: act.AppId,
		State:         TransactionStateInitial,
		AssetId:       asset,
		Receivers:     grp.GetMembers(),
		Threshold:     grp.GetThreshold(),
		Amount:        total.String(),
		Memo:          "",
		References:    []crypto.Hash{hash},
		Compaction:    true,
		Sequence:      act.Sequence,
	}
	return tx, nil
}

func (grp *Group) signTransaction(ctx context.Context, tx *Transaction) ([]byte, error) {
	outputs, err := grp.ListOutputsForTransaction(ctx, tx.TraceId, tx.Sequence)
	if err != nil {
		panic(err)
	}
	if len(outputs) == 0 {
		outputs, err = grp.ListOutputsForAsset(ctx, tx.AppId, tx.AssetId, tx.Sequence, SafeUtxoStateUnspent, OutputsBatchSize)
	}
	if err != nil {
		panic(err)
	}
	logger.Verbosef("group.ListOutputsForTransaction(%s) => %d %v\n", tx.TraceId, len(outputs), err)
	if len(outputs) == 0 {
		return nil, fmt.Errorf("empty outputs %s", tx.Amount)
	}
	if tx.Compaction && len(outputs) < OutputsBatchSize {
		return nil, fmt.Errorf("insufficient compaction transaction outputs %v %d", tx, len(outputs))
	}

	old, err := grp.readMultisigUntilSufficient(ctx, tx.TraceId)
	if err != nil {
		return nil, err
	}
	if old != nil {
		rb, _ := hex.DecodeString(old.RawTransaction)
		ver, _ := common.UnmarshalVersionedTransaction(rb)
		var consumed []*UnifiedOutput
		for _, input := range ver.Inputs {
			output, err := grp.store.readOutputByHashAndIndex(ctx, input.Hash.String(), input.Index)
			if err != nil {
				return nil, err
			}
			if output == nil {
				return nil, fmt.Errorf("output not found: %s:%d", input.Hash.String(), input.Index)
			}
		}

		if tx.Compaction && len(ver.Outputs) != 1 {
			return nil, fmt.Errorf("expired compaction transaction %v", tx)
		}
		var req *mixin.SafeMultisigRequest
		switch {
		case len(old.Signers) >= int(old.SendersThreshold) || slices.Contains(old.Signers, grp.mixin.ClientID):
			req = old
		default:
			req, err = grp.signMultisigUntilSufficient(ctx, old)
			if err != nil {
				return nil, err
			}
		}
		return grp.updateTxWithOutputs(ctx, tx, consumed, req)
	}

	ver, consumed, err := grp.buildRawTransaction(ctx, tx, outputs)
	logger.Verbosef("group.buildRawTransaction(%v) => %v %d %v\n", tx, ver, len(consumed), err)
	if err != nil {
		return nil, err
	}
	if tx.Compaction && len(ver.Outputs) != 1 {
		return nil, fmt.Errorf("expired compaction transaction %v", tx)
	}

	req, err := grp.readMultisigUntilSufficient(ctx, tx.TraceId)
	if err != nil {
		return nil, err
	}
	if req == nil {
		raw := hex.EncodeToString(ver.Marshal())
		req, err = grp.createMultisigUntilSufficient(ctx, tx.TraceId, raw)
		if err != nil {
			return nil, err
		}
	}
	req, err = grp.signMultisigUntilSufficient(ctx, req)
	if err != nil {
		return nil, err
	}

	return grp.updateTxWithOutputs(ctx, tx, outputs, req)
}

func (grp *Group) updateTxWithOutputs(ctx context.Context, tx *Transaction, outputs []*UnifiedOutput, req *mixin.SafeMultisigRequest) ([]byte, error) {
	for _, out := range outputs {
		out.TraceId = tx.TraceId
		out.State = SafeUtxoStateSigned
		out.SignedBy = req.TransactionHash
		out.SignedTx = req.RawTransaction
	}

	rb, err := hex.DecodeString(req.RawTransaction)
	if err != nil {
		return nil, err
	}
	ver, _ := common.UnmarshalVersionedTransaction(rb)
	tx.Raw = rb
	tx.Hash = ver.PayloadHash()
	tx.UpdatedAt = time.Now()
	tx.State = TransactionStateSigning

	if len(req.Signers) > 0 {
		tx.State = TransactionStateSigned
	} else {
		t, g, _ := DecodeMixinExtra(string(ver.Extra))
		if t != tx.TraceId {
			panic(hex.EncodeToString(rb))
		}
		if g != tx.OpponentAppId {
			panic(hex.EncodeToString(rb))
		}
	}

	err = grp.store.updateTxWithOutputs(ctx, tx, outputs)
	if err != nil {
		panic(err)
	}
	return rb, nil
}

func (grp *Group) createGhostKeysUntilSufficient(ctx context.Context, tx *Transaction) ([]*mixin.GhostKeys, error) {
	if CheckTestEnvironment(ctx) {
		var keys []*mixin.GhostKeys
		key1, err := testGetGhostKeys(tx, 0)
		if err != nil {
			return nil, err
		}
		key2, err := testGetGhostKeys(tx, 0)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key1)
		keys = append(keys, key2)
		return keys, nil
	}

	for {
		sort.Strings(tx.Receivers)
		keys, err := grp.mixin.SafeCreateGhostKeys(ctx, []*mixin.GhostInput{{
			Receivers: tx.Receivers,
			Index:     0,
			Hint:      mixin.UniqueConversationID(grp.mixin.ClientID, fmt.Sprintf("hint:%s;index:%d", tx.TraceId, 0)),
		}, {
			Receivers: grp.members,
			Index:     1,
			Hint:      mixin.UniqueConversationID(grp.mixin.ClientID, fmt.Sprintf("hint:%s;index:%d", tx.TraceId, 1)),
		}})
		logger.Verbosef("group.SafeCreateGhostKeys(%v) => %v %v\n", tx, keys, err)
		if err != nil && checkRetryableError(err) {
			time.Sleep(3 * time.Second)
			continue
		}
		return keys, err
	}
}

func (grp *Group) createMultisigUntilSufficient(ctx context.Context, id, raw string) (*mixin.SafeMultisigRequest, error) {
	for {
		req, err := grp.mixin.SafeCreateMultisigRequest(ctx, &mixin.SafeTransactionRequestInput{
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

func (grp *Group) getSpendPublicKeyUntilSufficient(ctx context.Context) (string, error) {
	for {
		me, err := grp.mixin.UserMe(ctx)
		logger.Verbosef("group.UserMe() => %v\n", err)
		if err != nil && checkRetryableError(err) {
			time.Sleep(3 * time.Second)
			continue
		}
		return me.SpendPublicKey, err
	}
}

func (grp *Group) signMultisigUntilSufficient(ctx context.Context, input *mixin.SafeMultisigRequest) (*mixin.SafeMultisigRequest, error) {
	spendPublicKey, err := grp.getSpendPublicKeyUntilSufficient(ctx)
	if err != nil {
		return nil, err
	}
	key, err := mixinnet.ParseKeyWithPub(grp.spendPrivateKey, spendPublicKey)
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
		req, err := grp.mixin.SafeSignMultisigRequest(ctx, &mixin.SafeTransactionRequestInput{
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

func (grp *Group) readMultisigUntilSufficient(ctx context.Context, id string) (*mixin.SafeMultisigRequest, error) {
	for {
		req, err := grp.mixin.SafeReadMultisigRequests(ctx, id)
		logger.Verbosef("group.SafeReadMultisigRequests(%s) => %v %v\n", id, req, err)
		if err != nil {
			if checkRetryableError(err) {
				time.Sleep(3 * time.Second)
				continue
			}
			if strings.Contains(err.Error(), "The endpoint is not found") {
				return nil, nil
			}
		}
		return req, err
	}
}

func (grp *Group) readTransactionUntilSufficient(ctx context.Context, id string) (*mixin.SafeTransactionRequest, error) {
	for {
		req, err := grp.mixin.SafeReadTransactionRequest(ctx, id)
		logger.Verbosef("group.SafeReadTransactionRequest(%s) => %v %v\n", id, req, err)
		if err != nil {
			if checkRetryableError(err) {
				time.Sleep(3 * time.Second)
				continue
			}
			if strings.Contains(err.Error(), "The endpoint is not found") {
				return nil, nil
			}
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

func (grp *Group) buildRawTransaction(ctx context.Context, tx *Transaction, outputs []*UnifiedOutput) (*common.VersionedTransaction, []*UnifiedOutput, error) {
	old := decodeTransaction(outputs[0].SignedTx)
	if old != nil && (old.AggregatedSignature != nil || len(old.SignaturesMap) > 0) {
		return old, nil, nil
	}
	ver := common.NewTransactionV5(crypto.Sha256Hash([]byte(tx.AssetId)))
	ver.Extra = []byte(tx.Memo)
	if !tx.Storage {
		ver.Extra = []byte(encodeMixinExtra(tx.OpponentAppId, tx.TraceId, tx.Memo))
	}
	target := common.NewIntegerFromString(tx.Amount)

	var total common.Integer
	var consumed []*UnifiedOutput
	for _, out := range outputs {
		total = total.Add(common.NewIntegerFromString(out.Amount.String()))
		ver.AddInput(out.TransactionHash, uint(out.OutputIndex))
		consumed = append(consumed, out)
		if total.Cmp(target) >= 0 && len(consumed) >= grp.groupSize {
			break
		}
	}
	if total.Cmp(target) < 0 {
		return nil, nil, fmt.Errorf("insufficient %d %s %s", len(outputs), total, tx.Amount)
	}

	keys, err := grp.createGhostKeysUntilSufficient(ctx, tx)
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

func decodeTransaction(s string) *common.VersionedTransaction {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	tx, err := common.UnmarshalVersionedTransaction(raw)
	if err != nil {
		return nil
	}
	return tx
}

func DecodeMixinExtra(memo string) (string, string, string) {
	extra, err := hex.DecodeString(memo)
	if err != nil {
		return "", "", ""
	}
	data, err := base64.RawURLEncoding.DecodeString(string(extra))
	if err != nil || len(data) < 32 {
		return "", "", ""
	}
	aid := uuid.FromBytesOrNil(data[0:16])
	tid := uuid.FromBytesOrNil(data[16:32])
	m := string(data[32:])
	return aid.String(), tid.String(), m
}

func encodeMixinExtra(appId, traceId, memo string) string {
	var gid uuid.UUID
	var err error
	if appId != "" {
		gid, err = uuid.FromString(appId)
		if err != nil {
			panic(err)
		}
	}
	tid, err := uuid.FromString(traceId)
	if err != nil {
		panic(err)
	}
	data := gid.Bytes()
	data = append(data, tid.Bytes()...)
	data = append(data, []byte(memo)...)
	s := base64.RawURLEncoding.EncodeToString(data)
	return s
}

func EncodeMixinExtra(appId, traceId, memo string) string {
	s := encodeMixinExtra(appId, traceId, memo)
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

func testGetGhostKeys(tx *Transaction, index int) (*mixin.GhostKeys, error) {
	k := &mixin.GhostKeys{}
	for _, r := range tx.Receivers {
		id := mixin.UniqueConversationID(r, tx.TraceId)
		id = mixin.UniqueConversationID(id, fmt.Sprint(index))
		bs := uuid.FromStringOrNil(id).Bytes()
		bs = append(bs, bs...)
		key, err := mixinnet.KeyFromSeed(hex.EncodeToString(bs))
		if err != nil {
			return nil, err
		}
		k.Keys = append(k.Keys, key)
	}

	id := mixin.UniqueConversationID(tx.TraceId, fmt.Sprint(index))
	bs := uuid.FromStringOrNil(id).Bytes()
	bs = append(bs, bs...)
	m, err := mixinnet.KeyFromSeed(hex.EncodeToString(bs))
	if err != nil {
		return nil, err
	}
	k.Mask = m
	return k, nil
}

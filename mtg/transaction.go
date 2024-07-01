package mtg

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
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

	OutputsBatchSize = 36
	StorageAssetId   = "c94ac88f-4671-3976-b60a-09064f1811e8"
)

type TransactionRecipient struct {
	MixAddress *mixin.MixAddress
	Amount     string
	UuidMember bool
}

type Transaction struct {
	AppId          string
	OpponentAppId  string
	TraceId        string
	State          int
	AssetId        string
	Receivers      []string
	Threshold      int
	Amount         string
	Memo           string
	Raw            []byte
	Hash           crypto.Hash
	References     []crypto.Hash
	Sequence       uint64
	Compaction     bool
	Storage        bool
	StorageTraceId string
	UpdatedAt      time.Time
}

var transactionCols = []string{"app_id", "opponent_app_id", "trace_id", "state", "asset_id", "receivers", "threshold", "amount", "memo", "raw", "hash", "refs", "sequence", "compaction", "storage", "storage_trace_id", "updated_at"}

func (t *Transaction) values() []any {
	var refs []string
	for _, r := range t.References {
		refs = append(refs, r.String())
	}
	var hash, raw sql.NullString
	if t.Hash.HasValue() {
		hash = sql.NullString{Valid: true, String: t.Hash.String()}
		raw = sql.NullString{Valid: true, String: hex.EncodeToString(t.Raw)}
	}
	return []any{t.AppId, t.OpponentAppId, t.TraceId, t.State, t.AssetId, strings.Join(t.Receivers, ","), t.Threshold, t.Amount, t.Memo, raw, hash, strings.Join(refs, ","), t.Sequence, t.Compaction, t.Storage, t.StorageTraceId, t.UpdatedAt}
}

func transactionFromRow(row Row) (*Transaction, error) {
	var t Transaction
	var rs, refs string
	var hash, raw sql.NullString
	err := row.Scan(&t.AppId, &t.OpponentAppId, &t.TraceId, &t.State, &t.AssetId, &rs, &t.Threshold, &t.Amount, &t.Memo, &raw, &hash, &refs, &t.Sequence, &t.Compaction, &t.Storage, &t.StorageTraceId, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if hash.Valid {
		h, err := crypto.HashFromString(hash.String)
		if err != nil {
			panic(hash.String)
		}
		r, err := hex.DecodeString(raw.String)
		if err != nil {
			panic(raw.String)
		}
		t.Hash = h
		t.Raw = r
	}

	t.Receivers = strings.Split(rs, ",")
	for _, r := range strings.Split(refs, ",") {
		ref, err := crypto.HashFromString(r)
		if err != nil {
			return nil, err
		}
		t.References = append(t.References, ref)
	}
	return &t, nil
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

func (grp *Group) BuildTransactionWithStorageTraceId(traceId, appId, opponentAppId, assetId, amount, memo string, receivers []string, threshold int, sequence uint64, storageTraceId string) *Transaction {
	if _, err := uuid.FromString(storageTraceId); err != nil {
		panic(err)
	}
	t := grp.BuildTransaction(traceId, appId, opponentAppId, assetId, amount, memo, receivers, threshold, sequence)
	t.StorageTraceId = storageTraceId
	return t
}

func (grp *Group) BuildStorageTransaction(ctx context.Context, appId string, extra []byte, sequence uint64) (*Transaction, error) {
	sTraceId := crypto.Blake3Hash(extra).String()
	sTraceId = UniqueId(sTraceId, sTraceId)
	old, err := grp.store.ReadTransactionByTraceId(ctx, sTraceId)
	if err != nil || old != nil {
		return old, err
	}
	if len(extra) > common.ExtraSizeStorageCapacity {
		return nil, fmt.Errorf("too large extra %d > %d", len(extra), common.ExtraSizeStorageCapacity)
	}

	step := common.NewIntegerFromString(common.ExtraStoragePriceStep)
	amount := step.Mul(len(extra)/common.ExtraSizeStorageStep + 1)

	total, err := grp.CheckAssetBalanceAt(ctx, appId, StorageAssetId, sequence)
	if err != nil {
		return nil, err
	}
	if common.NewIntegerFromString(total.String()).Cmp(amount) < 0 {
		return nil, fmt.Errorf("insufficient balance %s %s", total, amount)
	}

	addr := common.NewAddressFromSeed(make([]byte, 64))
	receivers := []string{addr.String()}
	t := grp.BuildTransaction(sTraceId, appId, appId, StorageAssetId, amount.String(), string(extra), receivers, 64, sequence)
	t.Storage = true
	return t, grp.store.WriteStorageTransaction(ctx, t)
}

func (t *Transaction) check(ctx context.Context) error {
	logger.Debugf("Group.checkTransaction(%v)\n", t)
	if _, err := uuid.FromString(t.AppId); err != nil {
		panic(err)
	}
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

	s := encodeMixinExtra(t.OpponentAppId, t.Memo)
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
	if len(compaction) != OutputsBatchSize {
		return nil, fmt.Errorf("insufficient outputs to build compaction transaction: %d", len(compaction))
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
		TraceId:       UniqueId(act.OutputId, "compaction"),
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
	logger.Printf("Group.signTransaction(%v)\n", tx)

	txs, err := grp.store.ListPreviousInitialTransactions(ctx, tx.AssetId, tx.Sequence)
	logger.Verbosef("store.ListPreviousInitialTransactions(%s %d) => %d %v\n", tx.AssetId, tx.Sequence, len(txs), err)
	if err != nil {
		return nil, err
	}
	if len(txs) > 0 {
		return nil, nil
	}

	if tx.StorageTraceId != "" {
		storageTx, err := grp.store.ReadTransactionByTraceId(ctx, tx.StorageTraceId)
		if err != nil || storageTx == nil || !storageTx.Storage {
			return nil, fmt.Errorf("store.ReadTransactionByTraceId(%s) => %v %v", tx.StorageTraceId, storageTx, err)
		}
		if storageTx.State != TransactionStateSnapshot {
			return nil, nil
		}
		t, err := grp.readTransactionUntilSufficient(ctx, tx.StorageTraceId)
		if err != nil {
			return nil, err
		}
		ref, err := crypto.HashFromString(t.TransactionHash)
		if err != nil {
			return nil, err
		}
		tx.References = []crypto.Hash{ref}
	}

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
	logger.Verbosef("Group.ListOutputsForTransaction(%s) => %d %v\n", tx.TraceId, len(outputs), err)
	if len(outputs) == 0 {
		return nil, fmt.Errorf("empty outputs %s", tx.Amount)
	}
	if tx.Compaction && len(outputs) < OutputsBatchSize {
		return nil, fmt.Errorf("insufficient compaction transaction outputs %v %d", tx, len(outputs))
	}

	ver, consumed, err := grp.buildRawTransaction(ctx, tx, outputs)
	logger.Verbosef("Group.buildRawTransaction(%v) => %v %d %v\n", tx, ver, len(consumed), err)
	if err != nil {
		return nil, err
	}
	if tx.Compaction && len(ver.Outputs) != 1 {
		return nil, fmt.Errorf("expired compaction transaction %v", tx)
	}

	raw := hex.EncodeToString(ver.Marshal())
	req, err := grp.createMultisigUntilSufficient(ctx, tx.TraceId, raw)
	if err != nil {
		return nil, err
	}
	req, err = grp.signMultisigUntilSufficient(ctx, req)
	if err != nil {
		return nil, err
	}

	return grp.updateTxWithOutputs(ctx, tx, consumed, req)
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
	} else if !tx.Storage {
		aid, _ := DecodeMixinExtra(string(ver.Extra))
		if aid != tx.OpponentAppId {
			panic(hex.EncodeToString(rb))
		}
	}

	err = grp.store.UpdateTxWithOutputs(ctx, tx, outputs)
	if err != nil {
		panic(err)
	}
	return rb, nil
}

func (grp *Group) createMultisigUntilSufficient(ctx context.Context, id, raw string) (*mixin.SafeMultisigRequest, error) {
	if CheckTestEnvironment(ctx) {
		rb, _ := hex.DecodeString(raw)
		ver, _ := common.UnmarshalVersionedTransaction(rb)
		hash := ver.PayloadHash()
		return &mixin.SafeMultisigRequest{
			RequestID:       id,
			RawTransaction:  raw,
			TransactionHash: hash.String(),
		}, nil
	}
	for {
		req, err := grp.mixin.SafeCreateMultisigRequest(ctx, &mixin.SafeTransactionRequestInput{
			RequestID:      id,
			RawTransaction: raw,
		})
		logger.Verbosef("Group.SafeCreateTransactionRequest(%s, %s) => %v %v\n", id, raw, req, err)
		if err != nil && CheckRetryableError(err) {
			time.Sleep(3 * time.Second)
			continue
		}
		return req, err
	}
}

func (grp *Group) signMultisigUntilSufficient(ctx context.Context, input *mixin.SafeMultisigRequest) (*mixin.SafeMultisigRequest, error) {
	if CheckTestEnvironment(ctx) {
		rb, _ := hex.DecodeString(input.RawTransaction)
		ver, _ := common.UnmarshalVersionedTransaction(rb)
		hash := ver.PayloadHash()
		return &mixin.SafeMultisigRequest{
			RequestID:       input.RequestID,
			RawTransaction:  input.RawTransaction,
			TransactionHash: hash.String(),
			Signers:         []string{grp.GroupId},
		}, nil
	}
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
		logger.Verbosef("Group.SafeSignMultisigRequest(%s %s) => %v %v\n", input.RequestID, signedRaw, req, err)
		if err != nil && CheckRetryableError(err) {
			time.Sleep(3 * time.Second)
			continue
		}
		return req, err
	}
}

func (grp *Group) buildRawTransaction(ctx context.Context, tx *Transaction, outputs []*UnifiedOutput) (*common.VersionedTransaction, []*UnifiedOutput, error) {
	inputs, tr, err := grp.getTransactionInputsAndRecipients(ctx, tx, outputs)
	if err != nil {
		return nil, nil, err
	}

	ver := common.NewTransactionV5(crypto.Sha256Hash([]byte(tx.AssetId)))
	for _, in := range inputs {
		ver.AddInput(in.TransactionHash, uint(in.OutputIndex))
	}

	keys, err := grp.createGhostKeysUntilSufficient(ctx, tx, tr)
	if err != nil {
		return nil, nil, err
	}
	for i, r := range tr {
		ver.Outputs = append(ver.Outputs, newCommonOutput(&mixinnet.Output{
			Type:   common.OutputTypeScript,
			Mask:   keys[i].Mask,
			Keys:   keys[i].Keys,
			Amount: mixinnet.IntegerFromString(r.Amount),
			Script: mixinnet.NewThresholdScript(uint8(r.MixAddress.Threshold)),
		}))
	}

	ver.References = tx.References
	ver.Extra = []byte(tx.Memo)
	if !tx.Storage {
		ver.Extra = []byte(encodeMixinExtra(tx.OpponentAppId, tx.Memo))
	}

	if l := ver.AsVersioned().GetExtraLimit(); len(ver.Extra) >= l {
		return nil, nil, fmt.Errorf("large extra %d > %d", len(ver.Extra), l)
	}
	return ver.AsVersioned(), inputs, nil
}

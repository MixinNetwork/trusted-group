package mtg

import (
	"bytes"
	"context"
	"database/sql"
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
	TransactionStateSigned   = 12
	TransactionStateSnapshot = 13

	OutputsBatchSize = 36
	StorageAssetId   = "c94ac88f-4671-3976-b60a-09064f1811e8"
	MixinFeeUserId   = "674d6776-d600-4346-af46-58e77d8df185"
)

type TransactionRecipient struct {
	MixAddress *mixin.MixAddress
	Amount     string
	UuidMember bool

	Destination string
	Tag         string
}

type Transaction struct {
	TraceId       string
	AppId         string
	OpponentAppId string
	ActionId      string
	State         int
	AssetId       string
	Receivers     []string
	Threshold     int
	Amount        string
	Memo          string
	Raw           []byte
	Hash          crypto.Hash
	Sequence      uint64
	UpdatedAt     time.Time

	compaction     bool
	storage        bool
	references     []crypto.Hash
	storageTraceId string
	Destination    sql.NullString
	Tag            sql.NullString
	WithdrawalHash sql.NullString
	requestId      sql.NullString
	consumed       []*UnifiedOutput
	consumedIds    []string
}

var transactionCols = []string{
	"trace_id", "app_id", "opponent_app_id", "action_id", "state", "asset_id", "receivers", "threshold", "amount", "memo", "raw", "hash", "refs", "sequence",
	"compaction", "storage", "storage_trace_id", "destination", "tag", "withdrawal_hash", "request_id", "updated_at",
}

func (t *Transaction) values() []any {
	var refs []string
	for _, r := range t.references {
		refs = append(refs, r.String())
	}
	var hash, raw sql.NullString
	if t.Hash.HasValue() {
		hash = sql.NullString{Valid: true, String: t.Hash.String()}
		raw = sql.NullString{Valid: true, String: hex.EncodeToString(t.Raw)}
	}
	return []any{
		t.TraceId, t.AppId, t.OpponentAppId, t.ActionId, t.State, t.AssetId, strings.Join(t.Receivers, ","), t.Threshold, t.Amount, t.Memo, raw, hash, strings.Join(refs, ","), t.Sequence,
		t.compaction, t.storage, t.storageTraceId, t.Destination, t.Tag, t.WithdrawalHash, t.requestId, t.UpdatedAt,
	}
}

func (t *Transaction) IsStorage() bool {
	return t.storage
}

func (t *Transaction) IsWithdrawal() bool {
	return t.Destination.Valid
}

func (t *Transaction) IsNormal() bool {
	return !t.IsStorage() && !t.IsWithdrawal()
}

func transactionFromRow(row Row) (*Transaction, error) {
	var t Transaction
	var rs, refs string
	var hash, raw sql.NullString
	err := row.Scan(
		&t.TraceId, &t.AppId, &t.OpponentAppId, &t.ActionId, &t.State, &t.AssetId, &rs, &t.Threshold, &t.Amount, &t.Memo, &raw, &hash, &refs, &t.Sequence,
		&t.compaction, &t.storage, &t.storageTraceId, &t.Destination, &t.Tag, &t.WithdrawalHash, &t.requestId, &t.UpdatedAt)
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

	t.Receivers = SplitIds(rs)
	for _, r := range SplitIds(refs) {
		ref, err := crypto.HashFromString(r)
		if err != nil {
			return nil, err
		}
		t.references = append(t.references, ref)
	}
	return &t, nil
}

func (act *Action) BuildTransaction(ctx context.Context, traceId, opponentAppId, assetId, amount, memo string, receivers []string, threshold int) *Transaction {
	rs := make([]string, len(receivers))
	copy(rs, receivers)
	sort.Strings(rs)

	tx := &Transaction{
		TraceId:       traceId,
		OpponentAppId: opponentAppId,
		ActionId:      act.OutputId,
		State:         TransactionStateInitial,
		AssetId:       assetId,
		Amount:        amount,
		Receivers:     receivers,
		Threshold:     threshold,
		Memo:          memo,
		AppId:         act.AppId,
		Sequence:      act.Sequence,
	}
	outputs := act.group.ListOutputsForAsset(ctx, tx.AppId, tx.AssetId, act.consumed[assetId], tx.Sequence, SafeUtxoStateUnspent, OutputsBatchSize)
	if len(outputs) == 0 {
		panic(tx.TraceId)
	}
	if ids := safeTransactionSequenceOrderHack[tx.TraceId]; len(ids) > 0 {
		hack, err := act.group.store.listOutputs(ctx, ids)
		if err != nil {
			panic(err)
		}
		outputs = hack
	}
	inputs, _, err := act.group.getTransactionInputsAndRecipients(ctx, tx, outputs)
	if err != nil {
		panic(err)
	}
	tx.consumed = inputs
	for _, o := range tx.consumed {
		tx.consumedIds = append(tx.consumedIds, o.OutputId)
		if o.Sequence > act.consumed[assetId] {
			act.consumed[assetId] = o.Sequence
		}
	}
	return tx
}

func (act *Action) BuildTransactionWithReference(ctx context.Context, traceId, opponentAppId, assetId, amount, memo string, receivers []string, threshold int, reference crypto.Hash) *Transaction {
	if !reference.HasValue() {
		panic(reference)
	}
	t := act.BuildTransaction(ctx, traceId, opponentAppId, assetId, amount, memo, receivers, threshold)
	t.references = []crypto.Hash{reference}
	return t
}

func (act *Action) BuildTransactionWithStorageTraceId(ctx context.Context, traceId, opponentAppId, assetId, amount, memo string, receivers []string, threshold int, storageTraceId string) *Transaction {
	if _, err := uuid.FromString(storageTraceId); err != nil {
		panic(err)
	}
	t := act.BuildTransaction(ctx, traceId, opponentAppId, assetId, amount, memo, receivers, threshold)
	t.storageTraceId = storageTraceId
	return t
}

func (act *Action) BuildStorageTransaction(ctx context.Context, extra []byte) *Transaction {
	if len(extra) > common.ExtraSizeStorageCapacity {
		panic(fmt.Errorf("too large extra %d > %d", len(extra), common.ExtraSizeStorageCapacity))
	}

	sTraceId := crypto.Blake3Hash(extra).String()
	sTraceId = UniqueId(sTraceId, sTraceId)
	addr := common.NewAddressFromSeed(make([]byte, 64))
	receivers := []string{addr.String()}
	amount := getStorageTransactionAmount(extra)
	t := act.BuildTransaction(ctx, sTraceId, act.group.GroupId, StorageAssetId, amount.String(), string(extra), receivers, 64)
	t.storage = true
	return t
}

func (act *Action) BuildWithdrawTransaction(ctx context.Context, traceId, assetId, amount, memo, destination, tag string) *Transaction {
	tx := &Transaction{
		TraceId:        traceId,
		AppId:          act.AppId,
		OpponentAppId:  act.group.GroupId,
		ActionId:       act.OutputId,
		State:          TransactionStateInitial,
		AssetId:        assetId,
		Amount:         amount,
		Memo:           memo,
		Sequence:       act.Sequence,
		Destination:    sql.NullString{Valid: true, String: destination},
		Tag:            sql.NullString{Valid: true, String: tag},
		WithdrawalHash: sql.NullString{Valid: false},
	}
	outputs := act.group.ListOutputsForAsset(ctx, tx.AppId, tx.AssetId, act.consumed[assetId], tx.Sequence, SafeUtxoStateUnspent, OutputsBatchSize)
	if len(outputs) == 0 {
		panic(tx.TraceId)
	}
	inputs, _, err := act.group.getTransactionInputsAndRecipients(ctx, tx, outputs)
	if err != nil {
		panic(err)
	}
	tx.consumed = inputs
	for _, o := range tx.consumed {
		tx.consumedIds = append(tx.consumedIds, o.OutputId)
		if o.Sequence > act.consumed[assetId] {
			act.consumed[assetId] = o.Sequence
		}
	}
	return tx
}

func getStorageTransactionAmount(extra []byte) common.Integer {
	step := common.NewIntegerFromString(common.ExtraStoragePriceStep)
	return step.Mul(len(extra)/common.ExtraSizeStorageStep + 1)
}

func (t *Transaction) getConsumedString() string {
	if len(t.consumedIds) == 0 {
		panic(t.TraceId)
	}
	if len(t.consumed) > 0 && len(t.consumed) != len(t.consumedIds) {
		panic(t.TraceId)
	}
	return strings.Join(t.consumedIds, ",")
}

func (t *Transaction) Equal(tx *Transaction) bool {
	return tx.TraceId == t.TraceId &&
		uuid.FromStringOrNil(tx.AppId).String() == uuid.FromStringOrNil(t.AppId).String() &&
		tx.OpponentAppId == t.OpponentAppId &&
		tx.State == t.State &&
		tx.AssetId == t.AssetId &&
		tx.Threshold == t.Threshold &&
		tx.ActionId == t.ActionId &&
		tx.Amount == t.Amount &&
		tx.Memo == t.Memo &&
		tx.Hash == t.Hash &&
		tx.Sequence == t.Sequence &&
		tx.compaction == t.compaction &&
		tx.storage == t.storage &&
		tx.storageTraceId == t.storageTraceId &&
		tx.getConsumedString() == t.getConsumedString() &&
		tx.Destination.String == t.Destination.String &&
		tx.Tag.String == t.Tag.String &&
		tx.WithdrawalHash.String == t.WithdrawalHash.String &&
		bytes.Equal(tx.Raw, t.Raw) &&
		slices.Equal(tx.Receivers, t.Receivers) &&
		slices.Equal(tx.references, t.references)
}

func (t *Transaction) check(_ context.Context, act *Action) error {
	logger.Debugf("Group.checkTransaction(%v)\n", t)
	if _, err := uuid.FromString(t.AppId); err != nil {
		panic(err)
	}
	if t.AppId != act.AppId || t.Sequence != act.Sequence || t.ActionId != act.OutputId {
		return fmt.Errorf("invalid action origin: %s %d %s", t.AppId, t.Sequence, t.ActionId)
	}
	if len(t.references) > 2 {
		return fmt.Errorf("invalid references length: %d", len(t.references))
	}
	for i, r := range t.references {
		if !r.HasValue() {
			return fmt.Errorf("invalid reference: %d %v", i, r)
		}
	}
	if !t.IsWithdrawal() {
		if t.Threshold < 1 || t.Threshold > 128 {
			return fmt.Errorf("invalid receivers threshold %d/%d", t.Threshold, len(t.Receivers))
		}
		for _, r := range t.Receivers {
			id, _ := uuid.FromString(r)
			if id.String() == uuid.Nil.String() {
				_, err := mixinnet.AddressFromString(r)
				if err != nil {
					return fmt.Errorf("invalid receiver %s", r)
				}
			}
		}
	}
	amt := decimal.RequireFromString(t.Amount)
	min := decimal.RequireFromString("0.00000001")
	if amt.Cmp(min) < 0 {
		return fmt.Errorf("invalid amount %s", t.Amount)
	}

	limit := common.ExtraSizeGeneralLimit
	if t.IsStorage() {
		limit = common.ExtraSizeStorageCapacity
	}
	s := encodeMixinExtra(t.OpponentAppId, []byte(t.Memo))

	if len(s) >= limit {
		return fmt.Errorf("invalid extra length: %d", len(s))
	}

	encoded := t.Serialize()
	decoded, _ := Deserialize(encoded)
	if !t.Equal(decoded) {
		panic(hex.EncodeToString(encoded))
	}
	return nil
}

func (grp *Group) buildCompactionTransaction(ctx context.Context, asset string, act *Action) (*Transaction, error) {
	// compaction transaction is special, this is the sole transaction for an action
	compaction := grp.ListOutputsForAsset(ctx, act.AppId, asset, act.consumed[asset], act.Sequence, SafeUtxoStateUnspent, OutputsBatchSize)
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

	traceId := UniqueId(act.OutputId, "compaction")
	tx := act.BuildTransaction(ctx, traceId, act.AppId, asset, total.String(), "", grp.GetMembers(), grp.GetThreshold())
	tx.references = []crypto.Hash{hash}
	tx.compaction = true
	return tx, nil
}

func (grp *Group) signTransaction(ctx context.Context, tx *Transaction) *common.VersionedTransaction {
	logger.Printf("Group.signTransaction(%v)\n", tx)

	txs, err := grp.store.ListPreviousInitialTransactions(ctx, tx.AssetId, tx.Sequence)
	logger.Verbosef("store.ListPreviousInitialTransactions(%s %d) => %d %v\n", tx.AssetId, tx.Sequence, len(txs), err)
	if err != nil {
		panic(err)
	}
	if len(txs) > 0 {
		return nil
	}

	if tx.storageTraceId != "" {
		storageTx, err := grp.store.ReadTransactionByTraceId(ctx, tx.storageTraceId)
		if err != nil || storageTx == nil || !storageTx.storage {
			panic(fmt.Errorf("store.ReadTransactionByTraceId(%s) => %v %v", tx.storageTraceId, storageTx, err))
		}
		if storageTx.State != TransactionStateSnapshot {
			return nil
		}
		t, err := grp.readTransactionUntilSufficient(ctx, tx.storageTraceId)
		if err != nil {
			panic(err)
		}
		if storageTx.Hash.String() != t.TransactionHash {
			panic(tx.TraceId)
		}
		tx.references = []crypto.Hash{storageTx.Hash}
	}

	outputs := grp.ListOutputsForTransaction(ctx, tx.TraceId, tx.Sequence)
	logger.Verbosef("Group.ListOutputsForTransaction(%s) => %d %v\n", tx.TraceId, len(outputs), err)
	if len(outputs) == 0 {
		panic(fmt.Errorf("empty outputs %s", tx.Amount))
	}
	if tx.compaction && len(outputs) < OutputsBatchSize {
		panic(fmt.Errorf("insufficient compaction transaction outputs %v %d", tx, len(outputs)))
	}

	ver, consumed, err := grp.buildRawTransaction(ctx, tx, outputs)
	logger.Verbosef("Group.buildRawTransaction(%v) => %v %d %v\n", tx, ver, len(consumed), err)
	if err != nil || len(outputs) != len(consumed) {
		panic(err)
	}
	if tx.compaction && len(ver.Outputs) != 1 {
		panic(fmt.Errorf("invalid compaction transaction %v", tx))
	}

	raw := hex.EncodeToString(ver.Marshal())
	req, err := grp.createMultisigUntilSufficient(ctx, tx.RequestID(), raw)
	if err != nil {
		panic(err)
	}
	if len(req.Signers) < int(req.SendersThreshold) && len(req.Views) > 0 {
		req, err = grp.signMultisigUntilSufficient(ctx, req)
		if err != nil {
			panic(err)
		}
	} else {
		rb, err := hex.DecodeString(req.RawTransaction)
		if err != nil {
			panic(err)
		}
		ver, err := common.UnmarshalVersionedTransaction(rb)
		if err != nil {
			panic(err)
		}
		if !CheckTestEnvironment(ctx) {
			if len(ver.SignaturesMap) != len(ver.Inputs) {
				panic(tx.TraceId)
			}
			for _, signatureMap := range ver.SignaturesMap {
				if len(signatureMap) < int(req.SendersThreshold) {
					panic(fmt.Errorf("invalid multisigs raw transaction: %s", req.RequestID))
				}
			}
		}
	}

	vn, err := grp.updateTxWithOutputs(ctx, tx, consumed, req)
	if err != nil {
		panic(err)
	}
	if vn.PayloadHash() != ver.PayloadHash() {
		panic(vn.PayloadHash().String())
	}
	return vn
}

func (grp *Group) updateTxWithOutputs(ctx context.Context, tx *Transaction, outputs []*UnifiedOutput, req *mixin.SafeMultisigRequest) (*common.VersionedTransaction, error) {
	for _, out := range outputs {
		out.TraceId = tx.TraceId
		out.State = SafeUtxoStateSigned
		out.SignedBy = req.TransactionHash
	}

	rb, err := hex.DecodeString(req.RawTransaction)
	if err != nil {
		return nil, err
	}
	ver, _ := common.UnmarshalVersionedTransaction(rb)
	tx.Raw = rb
	tx.Hash = ver.PayloadHash()
	tx.UpdatedAt = time.Now().UTC()
	tx.State = TransactionStateSigned
	tx.requestId = sql.NullString{Valid: true, String: req.RequestID}

	if tx.Hash.String() != req.TransactionHash {
		panic(req.TransactionHash)
	}
	if tx.IsNormal() {
		aid, _ := DecodeMixinExtraBase64(string(ver.Extra))
		if aid != tx.OpponentAppId {
			panic(hex.EncodeToString(rb))
		}
	}

	err = grp.store.UpdateTxWithOutputs(ctx, tx, outputs)
	if err != nil {
		panic(err)
	}
	return ver, nil
}

func (tx *Transaction) RequestID() string {
	if tx.requestId.Valid {
		return tx.requestId.String
	}
	return tx.TraceId
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
		if err != nil {
			return nil, err
		}
		if req.RevokedBy != "" {
			id = UniqueId(id, "next")
			continue
		}
		return req, nil
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
		h, err := crypto.HashFromString(in.TransactionHash)
		if err != nil {
			panic(in.TransactionHash)
		}
		ver.AddInput(h, uint(in.OutputIndex))
	}

	keys, err := grp.createGhostKeysUntilSufficient(ctx, tx, tr)
	if err != nil {
		return nil, nil, err
	}
	for i, r := range tr {
		if r.Destination == "" && r.MixAddress == nil {
			panic(r)
		}
		if r.Destination != "" {
			ver.Outputs = append(ver.Outputs, &common.Output{
				Type:   common.OutputTypeWithdrawalSubmit,
				Amount: common.NewIntegerFromString(r.Amount),
				Withdrawal: &common.WithdrawalData{
					Address: r.Destination,
					Tag:     r.Tag,
				},
			})
			continue
		}

		ver.Outputs = append(ver.Outputs, newCommonOutput(&mixinnet.Output{
			Type:   common.OutputTypeScript,
			Mask:   keys[i].Mask,
			Keys:   keys[i].Keys,
			Amount: mixinnet.IntegerFromString(r.Amount),
			Script: mixinnet.NewThresholdScript(uint8(r.MixAddress.Threshold)),
		}))
	}

	ver.References = tx.references
	ver.Extra = []byte(tx.Memo)
	if tx.IsNormal() {
		ver.Extra = []byte(EncodeMixinExtraBase64(tx.OpponentAppId, ver.Extra))
	}

	if l := ver.AsVersioned().GetExtraLimit(); len(ver.Extra) >= l {
		return nil, nil, fmt.Errorf("large extra %d > %d", len(ver.Extra), l)
	}
	return ver.AsVersioned(), inputs, nil
}

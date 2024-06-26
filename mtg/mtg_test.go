package mtg

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/gofrs/uuid/v5"
	"github.com/pelletier/go-toml"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const (
	USDTAssetId = "218bc6f4-7927-3f8e-8568-3a3725b74361"
	testSender  = "e14c1573-3aca-48b1-b437-766b4757b50d"
)

type Node struct {
	Group *Group
}

func (n *Node) ProcessOutput(ctx context.Context, a *Action) ([]*Transaction, string) {
	b, err := hex.DecodeString(a.Extra)
	if err != nil {
		panic(err)
	}
	b, err = base64.RawStdEncoding.DecodeString(string(b))
	if err != nil {
		return []*Transaction{}, ""
	}
	memo := string(b)
	items := strings.Split(memo, ",")

	var txs []*Transaction
	var storageTraceId string
	for _, tx := range items {
		if tx == "storage" {
			t, err := n.Group.BuildStorageTransaction(ctx, a.AppId, []byte("storage-memo"), a.Sequence)
			if err != nil {
				panic(err)
			}
			storageTraceId = t.TraceId
		} else {
			amt, err := decimal.NewFromString(tx)
			if err != nil {
				return []*Transaction{}, ""
			}

			balance, err := n.Group.CheckAssetBalanceAt(ctx, a.AppId, a.AssetId, a.Sequence)
			if err != nil {
				panic(err)
			}
			if balance.Cmp(amt) < 0 {
				return nil, USDTAssetId
			}

			amount := amt.String()
			id := UniqueId(amount, testSender)
			var t *Transaction
			if storageTraceId != "" {
				t = n.Group.BuildTransactionWithStorageTraceId(id, a.AppId, UniqueId(a.AppId, "opponent"), a.AssetId, amount, "", n.Group.GetMembers(), n.Group.GetThreshold(), a.Sequence, storageTraceId)
			} else {
				t = n.Group.BuildTransaction(id, a.AppId, UniqueId(a.AppId, "opponent"), a.AssetId, amount, "", n.Group.GetMembers(), n.Group.GetThreshold(), a.Sequence)
			}
			txs = append(txs, t)
		}
	}
	return txs, ""
}

func TestMTGCompaction(t *testing.T) {
	require := require.New(t)
	ctx, node := testBuildGroup(require)
	require.NotNil(node)
	defer teardownTestDatabase(node.Group.store)

	testDrainInitialOutputs(ctx, require, node.Group, "0.0037")

	as, err := node.Group.store.ListActions(ctx, ActionStateDone, 0)
	require.Nil(err)
	require.Len(as, OutputsBatchSize+1)
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	require.Nil(err)
	require.Len(as, 1)
	hash := as[0].TransactionHash

	wkr := node.Group.FindWorker(as[0].AppId)
	require.NotNil(wkr)
	err = node.Group.handleActionsQueue(ctx)
	require.Nil(err)
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	require.Nil(err)
	require.Len(as, 0)
	as, err = node.Group.store.ListActions(ctx, ActionStateRestorable, 0)
	require.Nil(err)
	require.Len(as, 1)

	out := testHandleCompactionTransaction(ctx, require, node.Group, hash)
	node.Group.processSafeOutput(ctx, out)
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	require.Nil(err)
	require.Len(as, 1)
	require.Equal(as[0].TransactionHash, hash)

	err = node.Group.handleActionsQueue(ctx)
	require.Nil(err)
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	require.Nil(err)
	require.Len(as, 0)
	ts, _, err := node.Group.store.ListTransactions(ctx, TransactionStateInitial, 0)
	require.Nil(err)
	require.Len(ts, 1)
}

func TestMTGCheckTxs(t *testing.T) {
	require := require.New(t)
	ctx, node := testBuildGroup(require)
	require.NotNil(node)
	defer teardownTestDatabase(node.Group.store)

	testDrainInitialOutputs(ctx, require, node.Group, "0.003,0.0008")

	as, err := node.Group.store.ListActions(ctx, ActionStateDone, 0)
	require.Nil(err)
	require.Len(as, OutputsBatchSize+1)
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	require.Nil(err)
	require.Len(as, 1)

	wkr := node.Group.FindWorker(as[0].AppId)
	require.NotNil(wkr)
	err = node.Group.handleActionsQueue(ctx)
	require.NotNil(err)
	require.True(strings.Contains(err.Error(), "insufficient balance"))
}

func TestMTGStorage(t *testing.T) {
	require := require.New(t)
	ctx, node := testBuildGroup(require)
	require.NotNil(node)
	defer teardownTestDatabase(node.Group.store)

	testDrainInitialOutputs(ctx, require, node.Group, "storage,0.0001")

	as, err := node.Group.store.ListActions(ctx, ActionStateDone, 0)
	require.Nil(err)
	require.Len(as, OutputsBatchSize+1)
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	require.Nil(err)
	require.Len(as, 1)

	wkr := node.Group.FindWorker(as[0].AppId)
	require.NotNil(wkr)
	err = node.Group.handleActionsQueue(ctx)
	require.Nil(err)

	txs, _, err := node.Group.store.ListTransactions(ctx, TransactionStateInitial, 0)
	require.Nil(err)
	require.Len(txs, 2)
	require.True(txs[0].Storage)
	require.Equal(txs[0].TraceId, txs[1].StorageTraceId)

	_, err = node.Group.signTransaction(ctx, txs[1])
	require.Nil(err)
	tx, err := node.Group.store.ReadTransactionByTraceId(ctx, txs[1].TraceId)
	require.Nil(err)
	require.Equal(TransactionStateInitial, tx.State)

	_, err = node.Group.signTransaction(ctx, txs[0])
	require.Nil(err)
	err = node.Group.store.finishTransaction(ctx, txs[0].TraceId)
	require.Nil(err)
	_, err = node.Group.signTransaction(ctx, txs[1])
	require.Nil(err)

	storage, err := node.Group.store.ReadTransactionByTraceId(ctx, txs[0].TraceId)
	require.Nil(err)
	tx, err = node.Group.store.ReadTransactionByTraceId(ctx, txs[1].TraceId)
	require.Nil(err)
	require.NotEqual(TransactionStateInitial, tx.State)
	require.Equal(storage.Hash.String(), tx.References[0].String())
}

func testHandleCompactionTransaction(ctx context.Context, require *require.Assertions, group *Group, hash string) *UnifiedOutput {
	ts, _, err := group.store.ListTransactions(ctx, TransactionStateInitial, 0)
	require.Nil(err)
	require.Len(ts, 1)

	tx := ts[0]
	require.True(tx.Compaction)
	outputs, err := group.ListOutputsForAsset(ctx, tx.AppId, tx.AssetId, tx.Sequence, SafeUtxoStateUnspent, OutputsBatchSize)
	require.Nil(err)
	require.Len(outputs, 36)
	ver, consumed, err := group.buildRawTransaction(ctx, tx, outputs)
	require.Nil(err)
	require.Len(consumed, 36)
	require.Len(ver.References, 1)
	require.Equal(ver.References[0].String(), hash)

	tx.Hash = ver.PayloadHash()
	tx.Raw = ver.Marshal()
	tx.State = TransactionStateSnapshot
	for _, out := range consumed {
		out.State = SafeUtxoStateSpent
		out.SignedBy = tx.Hash.String()
		out.SignedTx = hex.EncodeToString(tx.Raw)
	}
	err = group.store.updateTxWithOutputs(ctx, tx, consumed)
	require.Nil(err)

	return testBuildActionFromTx(require, group, tx)
}

func testBuildActionFromTx(require *require.Assertions, group *Group, tx *Transaction) *UnifiedOutput {
	extra := encodeMixinExtra(tx.AppId, tx.TraceId, tx.Memo)
	extra = hex.EncodeToString([]byte(extra))
	return testBuildOutput(group, require, tx.AssetId, tx.Amount, extra, SafeUtxoStateUnspent, tx.Sequence+100, tx.Hash.String())
}

func testDrainInitialOutputs(ctx context.Context, require *require.Assertions, group *Group, memo string) {
	count := OutputsBatchSize + 1
	start := 4655228

	out := testBuildOutput(group, require, StorageAssetId, "1", "", SafeUtxoStateUnspent, uint64(start), "")
	err := group.store.writeAction(ctx, out, ActionStateDone)
	require.Nil(err)

	for i := range count {
		extra := ""
		state := ActionStateDone
		if i+1 == count {
			extra = base64.RawStdEncoding.EncodeToString([]byte(memo))
			extra = hex.EncodeToString([]byte(extra))
			state = ActionStateInitial
		}
		out := testBuildOutput(group, require, USDTAssetId, "0.0001", extra, SafeUtxoStateUnspent, uint64(start+i+1), "")

		err := group.store.writeAction(ctx, out, state)
		require.Nil(err)
	}
}

func testBuildOutput(group *Group, require *require.Assertions, asset, amount string, extra string, state SafeUtxoState, sequence uint64, hash string) *UnifiedOutput {
	oid := UniqueId("output", fmt.Sprintf("%s:%s:%s:%s:%d", amount, extra, extra, state, sequence))
	rid := UniqueId("request", oid)
	h := crypto.Sha256Hash(uuid.FromStringOrNil(oid).Bytes())
	if hash != "" {
		hash, err := crypto.HashFromString(hash)
		require.Nil(err)
		h = hash
	}
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		panic(err)
	}

	return &UnifiedOutput{
		OutputId:           oid,
		RequestId:          rid,
		TransactionHash:    h,
		OutputIndex:        0,
		AssetId:            asset,
		Amount:             amt,
		SendersThreshold:   int64(1),
		Senders:            []string{testSender},
		ReceiversThreshold: int64(group.GetThreshold()),
		Receivers:          group.GetMembers(),
		Extra:              extra,
		State:              state,
		Sequence:           sequence,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		AppId:              group.GroupId,
	}
}

func testBuildGroup(require *require.Assertions) (context.Context, *Node) {
	logger.SetLevel(logger.INFO)
	ctx := context.Background()
	ctx = EnableTestEnvironment(ctx)

	f, _ := os.ReadFile("./example.toml")
	var conf Configuration
	err := toml.Unmarshal(f, &conf)
	require.Nil(err)

	root, err := os.MkdirTemp("", "mtg-test")
	require.Nil(err)
	conf.StoreDir = root
	if !(strings.HasPrefix(conf.StoreDir, "/tmp") || strings.HasPrefix(conf.StoreDir, "/var/folders")) {
		panic(root)
	}
	store, err := OpenSQLite3Store(conf.StoreDir + "/mtg.sqlite3")
	require.Nil(err)

	group, err := BuildGroup(ctx, store, &conf)
	require.Nil(err)
	group.groupSize = 1

	n := &Node{
		Group: group,
	}
	group.AttachWorker(group.GroupId, n)

	d := DepositEntry{
		Destination: "213",
		Tag:         "",
	}
	group.RegisterDepositEntry(group.GroupId, d)

	app := group.FindAppByEntry("")
	require.Equal("", app)
	app = group.FindAppByEntry(d.String())
	require.Equal(group.GroupId, app)

	ns, err := group.store.ListIterations(ctx)
	require.Nil(err)
	require.Len(ns, 5)

	return ctx, n
}

func teardownTestDatabase(store *SQLite3Store) {
	dropTablesDDL := `
		DROP TABLE IF EXISTS properties;
		DROP TABLE IF EXISTS iterations;
		DROP TABLE IF EXISTS actions;
		DROP TABLE IF EXISTS outputs;
		DROP TABLE IF EXISTS transactions;
	`
	_, err := store.db.Exec(dropTablesDDL)
	if err != nil {
		panic(err)
	}
}

package mtg

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/pelletier/go-toml"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const (
	USDTAssetId = "218bc6f4-7927-3f8e-8568-3a3725b74361"
	SOLAssetId  = "64692c23-8971-4cf4-84a7-4dd1271dd887"
	testSender  = "e14c1573-3aca-48b1-b437-766b4757b50d"

	testWithdrawalDestination = "73yoz7kK3zgh2ScD9aTJpXCrKHETi1xyEKfMTH95ugff"
	testWithdrawalAmouint     = "0.0049"
	testWithdrawalMemo        = "withdrawal-test"
)

type Node struct {
	Group *Group
}

var actionResult map[string]string

func (n *Node) ProcessOutput(ctx context.Context, a *Action) ([]*Transaction, string) {
	if actionResult[a.OutputId] != "" {
		data, err := hex.DecodeString(actionResult[a.OutputId])
		if err != nil {
			panic(err)
		}
		txs, err := DeserializeTransactions(data)
		if err != nil {
			panic(err)
		}
		return txs, ""
	}

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
		switch tx {
		case "storage":
			extra := []byte("storage-memo")
			enough := a.CheckAssetBalanceForStorageAt(ctx, extra)
			if !enough {
				panic(a.Sequence)
			}
			t := a.BuildStorageTransaction(ctx, extra)
			storageTraceId = t.TraceId
			txs = append(txs, t)
		case "withdrawal":
			tid := "cf0564ba-bf51-4e8c-b504-3beb6c5c65e3"
			t := a.BuildWithdrawTransaction(ctx, tid, SOLAssetId, testWithdrawalAmouint, testWithdrawalMemo, testWithdrawalDestination, "")
			txs = append(txs, t)
		default:
			amt := decimal.RequireFromString(tx)
			balance := a.CheckAssetBalanceAt(ctx, a.AssetId)
			if balance.Cmp(amt) < 0 {
				return nil, USDTAssetId
			}

			amount := amt.String()
			id := UniqueId(amount, testSender)
			var t *Transaction
			if storageTraceId != "" {
				t = a.BuildTransactionWithStorageTraceId(ctx, id, UniqueId(a.AppId, "opponent"), a.AssetId, amount, "", n.Group.GetMembers(), n.Group.GetThreshold(), storageTraceId)
			} else {
				t = a.BuildTransaction(ctx, id, UniqueId(a.AppId, "opponent"), a.AssetId, amount, "", n.Group.GetMembers(), n.Group.GetThreshold())
			}
			txs = append(txs, t)
		}
	}

	if actionResult[a.OutputId] == "" {
		data := SerializeTransactions(txs)
		actionResult[a.OutputId] = hex.EncodeToString(data)
	}

	return txs, ""
}

func TestMTGExtra(t *testing.T) {
	require := require.New(t)
	id := uuid.Must(uuid.NewV4()).String()
	memo := "123"

	extra := EncodeMixinExtraBase64(id, []byte(memo))
	a, m := DecodeMixinExtraHEX(hex.EncodeToString([]byte(extra)))
	require.Equal(id, a)
	require.Equal(memo, string(m))
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
	err = node.Group.handleActionsQueue(ctx)
	require.Nil(err)

	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	require.Nil(err)
	require.Len(as, 1)

	as, err = node.Group.store.ListActions(ctx, ActionStateRestorable, 0)
	require.Nil(err)
	require.Len(as, 0)
	err = node.Group.handleActionsQueue(ctx)
	require.Nil(err)
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
	require.True(strings.Contains(err.Error(), "insufficient outputs"))
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
	require.True(txs[1].storage)
	require.Equal(txs[1].TraceId, txs[0].storageTraceId)

	ver := node.Group.signTransaction(ctx, txs[0])
	require.Nil(ver)
	tx, err := node.Group.store.ReadTransactionByTraceId(ctx, txs[0].TraceId)
	require.Nil(err)
	require.Equal(TransactionStateInitial, tx.State)

	ver = node.Group.signTransaction(ctx, txs[1])
	require.NotNil(ver)
	err = node.Group.store.FinishTransaction(ctx, txs[1].TraceId)
	require.Nil(err)
	ver = node.Group.signTransaction(ctx, txs[0])
	require.NotNil(ver)

	storage, err := node.Group.store.ReadTransactionByTraceId(ctx, txs[1].TraceId)
	require.Nil(err)
	tx, err = node.Group.store.ReadTransactionByTraceId(ctx, txs[0].TraceId)
	require.Nil(err)
	require.NotEqual(TransactionStateInitial, tx.State)
	require.Equal(storage.Hash.String(), tx.references[0].String())
}

func TestMTGWithdrawal(t *testing.T) {
	require := require.New(t)
	ctx, node := testBuildGroup(require)
	require.NotNil(node)
	defer teardownTestDatabase(node.Group.store)

	d, err := decimal.NewFromString("0.005")
	require.Nil(err)
	err = node.Group.store.WriteAction(ctx, &UnifiedOutput{
		OutputId:             "7514b939-db92-3d31-abf4-7841f035e400",
		TransactionRequestId: "cf0564ba-bf51-4e8c-b504-3beb6c5c65e2",
		TransactionHash:      "01c43005fd06e0b8f06a0af04faf7530331603e352a11032afd0fd9dbd84e8ee",
		OutputIndex:          0,
		AssetId:              SOLAssetId,
		Amount:               d,
		SendersThreshold:     int64(1),
		Senders:              []string{testSender},
		ReceiversThreshold:   int64(node.Group.GetThreshold()),
		Extra:                "",
		State:                SafeUtxoStateUnspent,
		Sequence:             4655227,
		AppId:                node.Group.GroupId,
	}, ActionStateDone)
	require.Nil(err)

	testDrainInitialOutputs(ctx, require, node.Group, "withdrawal")
	as, err := node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	require.Nil(err)
	require.Len(as, 1)
	wkr := node.Group.FindWorker(as[0].AppId)
	require.NotNil(wkr)
	err = node.Group.handleActionsQueue(ctx)
	require.Nil(err)

	txs, _, err := node.Group.store.ListTransactions(ctx, TransactionStateInitial, 0)
	require.Nil(err)
	require.Len(txs, 1)
	tx := txs[0]
	require.Equal(testWithdrawalAmouint, tx.Amount)
	require.Equal(testWithdrawalMemo, tx.Memo)
	require.Equal(SOLAssetId, tx.AssetId)
	require.Equal(testWithdrawalDestination, tx.destination.String)
	require.Equal("", tx.tag.String)
	require.False(tx.withdrawalHash.Valid)
}

func testHandleCompactionTransaction(ctx context.Context, require *require.Assertions, group *Group, hash string) *UnifiedOutput {
	ts, _, err := group.store.ListTransactions(ctx, TransactionStateInitial, 0)
	require.Nil(err)
	require.Len(ts, 1)

	tx := ts[0]
	require.True(tx.compaction)
	outputs := group.ListOutputsForAsset(ctx, tx.AppId, tx.AssetId, 0, tx.Sequence, SafeUtxoStateAssigned, OutputsBatchSize)
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
	}
	err = group.store.UpdateTxWithOutputs(ctx, tx, consumed)
	require.Nil(err)

	return testBuildActionFromTx(require, group, tx)
}

func testBuildActionFromTx(require *require.Assertions, group *Group, tx *Transaction) *UnifiedOutput {
	extra := encodeMixinExtra(tx.AppId, []byte(tx.Memo))
	extra = hex.EncodeToString([]byte(extra))
	return testBuildOutput(group, require, tx.AssetId, tx.Amount, extra, SafeUtxoStateUnspent, tx.Sequence+100, tx.Hash.String())
}

func testDrainInitialOutputs(ctx context.Context, require *require.Assertions, group *Group, memo string) {
	count := OutputsBatchSize + 1
	start := 4655228

	out := testBuildOutput(group, require, StorageAssetId, "1", "", SafeUtxoStateUnspent, uint64(start), "")
	err := group.store.WriteAction(ctx, out, ActionStateDone)
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

		err := group.store.WriteAction(ctx, out, state)
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
	oid = mixin.UniqueConversationID(fmt.Sprintf("%s:%d", h, 0), "")
	amt := decimal.RequireFromString(amount)

	return &UnifiedOutput{
		OutputId:             oid,
		TransactionRequestId: rid,
		TransactionHash:      h.String(),
		OutputIndex:          0,
		AssetId:              asset,
		Amount:               amt,
		SendersThreshold:     int64(1),
		Senders:              []string{testSender},
		ReceiversThreshold:   int64(group.GetThreshold()),
		Extra:                extra,
		State:                state,
		Sequence:             sequence,
		AppId:                group.GroupId,
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
	group.EnableDebug()

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
	app = group.FindAppByEntry(d.UniqueKey())
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

func init() {
	actionResult = make(map[string]string)
}

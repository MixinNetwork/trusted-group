package mtg

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/pelletier/go-toml"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

const (
	XINAssetId = "c94ac88f-4671-3976-b60a-09064f1811e8"
	testSender = "e14c1573-3aca-48b1-b437-766b4757b50d"
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
	amt, err := decimal.NewFromString(string(b))
	if err != nil {
		return []*Transaction{}, ""
	}

	balance, err := n.Group.CheckAssetBalanceAt(ctx, "", XINAssetId, a.Sequence)
	if err != nil {
		panic(err)
	}
	if balance.Cmp(amt) < 0 {
		return nil, XINAssetId
	}

	amount := amt.String()
	id := mixin.UniqueConversationID(amount, testSender)
	t := &Transaction{
		TraceId:   id,
		AssetId:   XINAssetId,
		Amount:    amount,
		Receivers: n.Group.GetMembers(),
		Threshold: n.Group.GetThreshold(),
		Sequence:  a.Sequence,
	}
	return []*Transaction{t}, ""
}

func TestMTGCompaction(t *testing.T) {
	assert := assert.New(t)
	ctx, node := testBuildGroup(assert)
	assert.NotNil(node)
	defer teardownTestDatabase(node.Group.store)

	testDrainInitialOutputs(ctx, assert, node.Group, "0.0037")

	as, err := node.Group.store.ListActions(ctx, ActionStateDone, 0)
	assert.Nil(err)
	assert.Len(as, OutputsBatchSize)
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	assert.Nil(err)
	assert.Len(as, 1)
	hash := as[0].TransactionHash

	err = node.Group.handleActionsQueue(ctx)
	assert.Nil(err)
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	assert.Nil(err)
	assert.Len(as, 0)
	as, err = node.Group.store.ListActions(ctx, ActionStateRrestorable, 0)
	assert.Nil(err)
	assert.Len(as, 1)

	act := testHandleCompactionTransaction(ctx, assert, node.Group, hash)
	node.Group.processSafeOutput(ctx, act.Output())
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	assert.Nil(err)
	assert.Len(as, 1)
	assert.Equal(as[0].TransactionHash, hash)

	err = node.Group.handleActionsQueue(ctx)
	assert.Nil(err)
	as, err = node.Group.store.ListActions(ctx, ActionStateInitial, 0)
	assert.Nil(err)
	assert.Len(as, 0)
	ts, err := node.Group.store.ListTransactions(ctx, TransactionStateInitial, 0)
	assert.Nil(err)
	assert.Len(ts, 1)
}

func testHandleCompactionTransaction(ctx context.Context, assert *assert.Assertions, group *Group, hash string) *Action {
	ts, err := group.store.ListTransactions(ctx, TransactionStateInitial, 0)
	assert.Nil(err)
	assert.Len(ts, 1)

	tx := ts[0]
	assert.True(tx.Compaction)
	outputs, err := group.ListOutputsForAsset(ctx, tx.GroupId, tx.AssetId, tx.Sequence, SafeUtxoStateUnspent, OutputsBatchSize)
	assert.Nil(err)
	assert.Len(outputs, 36)
	ver, consumed, err := group.buildRawTransaction(ctx, tx, outputs)
	assert.Nil(err)
	assert.Len(consumed, 36)
	assert.Len(ver.References, 1)
	assert.Equal(ver.References[0].String(), hash)

	tx.Hash = ver.PayloadHash()
	tx.Raw = ver.Marshal()
	tx.State = TransactionStateSnapshot
	for _, out := range consumed {
		out.State = SafeUtxoStateSpent
		out.SignedBy = tx.Hash.String()
		out.SignedTx = hex.EncodeToString(tx.Raw)
	}
	err = group.store.updateTxWithOutputs(ctx, tx, consumed)
	assert.Nil(err)

	act := testBuildActionFromTx(assert, group, tx)
	return act
}

func testBuildActionFromTx(assert *assert.Assertions, group *Group, tx *Transaction) *Action {
	extra := encodeMixinExtra(tx.GroupId, tx.TraceId, tx.Memo)
	extra = hex.EncodeToString([]byte(extra))
	out := testBuildOutput(group, assert, tx.Amount, extra, SafeUtxoStateUnspent, tx.Sequence+100, tx.Hash.String())
	act := out.Action(ActionStateInitial, tx.TraceId, tx.GroupId)
	return act
}

func testDrainInitialOutputs(ctx context.Context, assert *assert.Assertions, group *Group, amount string) {
	count := OutputsBatchSize + 1
	start := 4655228

	for i := range count {
		extra := ""
		state := ActionStateDone
		if i+1 == count {
			extra = base64.RawStdEncoding.EncodeToString([]byte(amount))
			extra = hex.EncodeToString([]byte(extra))
			state = ActionStateInitial
		}
		out := testBuildOutput(group, assert, "0.0001", extra, SafeUtxoStateUnspent, uint64(start+i+1), "")

		act := out.Action(state, "", "")
		err := group.store.writeAction(ctx, act, []*Transaction{})
		assert.Nil(err)
	}
}

func testBuildOutput(group *Group, assert *assert.Assertions, amount string, extra string, state SafeUtxoState, sequence uint64, hash string) *UnifiedOutput {
	oid := mixin.UniqueConversationID("output", fmt.Sprintf("%s:%s:%s:%s:%d", amount, extra, extra, state, sequence))
	rid := mixin.UniqueConversationID("request", oid)
	h := crypto.Sha256Hash(uuid.FromStringOrNil(oid).Bytes())
	if hash != "" {
		hash, err := crypto.HashFromString(hash)
		assert.Nil(err)
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
		AssetId:            XINAssetId,
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
	}
}

func testBuildGroup(assert *assert.Assertions) (context.Context, *Node) {
	logger.SetLevel(logger.INFO)
	ctx := context.Background()
	ctx = EnableTestEnvironment(ctx)

	f, _ := os.ReadFile("./example.toml")
	var conf Configuration
	err := toml.Unmarshal(f, &conf)
	assert.Nil(err)

	store, err := OpenSQLite3Store(conf.StoreDir + "/mtg.sqlite3")
	assert.Nil(err)

	group, err := BuildGroup(ctx, store, &conf)
	assert.Nil(err)
	group.groupSize = 1

	n := &Node{
		Group: group,
	}
	group.AddWorker(n)

	ns, err := group.store.ListIterations(ctx)
	assert.Nil(err)
	assert.Len(ns, 5)

	return ctx, n
}

func teardownTestDatabase(store *SQLite3Store) {
	dropTablesDDL := `
		DROP TABLE IF EXISTS properties;
		DROP TABLE IF EXISTS iterations;
		DROP TABLE IF EXISTS actions;
		DROP TABLE IF EXISTS transactions;
	`
	_, err := store.db.Exec(dropTablesDDL)
	if err != nil {
		panic(err)
	}
}

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

func (n *Node) ProcessOutput(ctx context.Context, a *Action) []*Transaction {
	b, err := hex.DecodeString(a.Extra)
	if err != nil {
		panic(err)
	}
	b, err = base64.RawStdEncoding.DecodeString(string(b))
	if err != nil {
		return []*Transaction{}
	}
	amt, err := decimal.NewFromString(string(b))
	if err != nil {
		return []*Transaction{}
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
	compaction, err := n.Group.GetBalanceForTransaction(ctx, t)
	if err != nil {
		return nil
	}
	if compaction.Cmp(decimal.NewFromInt(0)) > 0 {
		t.Amount = compaction.String()
		t.Memo = CompactionTransactionMemo
	}
	return []*Transaction{t}
}

func TestMTG(t *testing.T) {
	assert := assert.New(t)
	ctx, node := testBuildGroup(assert)
	assert.NotNil(node)
	defer teardownTestDatabase(node.Group.store)

	testDrainOutputs(ctx, assert, node.Group)

	err := node.Group.handleActionsQueue(ctx)
	assert.Nil(err)

	ts, err := node.Group.store.ListTransactions(ctx, TransactionStateInitial, 0)
	assert.Nil(err)
	assert.Len(ts, 1)

	outputs, err := node.Group.ListOutputsForAsset(ctx, ts[0], SafeUtxoStateUnspent, OutputsBatchSize)
	assert.Nil(err)
	assert.Len(outputs, 36)
	_, consumed, err := node.Group.buildRawTransaction(ctx, ts[0], outputs)
	assert.Nil(err)
	assert.Len(consumed, 2)
}

func testDrainOutputs(ctx context.Context, assert *assert.Assertions, group *Group) {
	count := OutputsBatchSize + 1
	start := 4655228

	for i := range count {
		extra := ""
		state := ActionStateDone
		if i+1 == count {
			extra = base64.RawStdEncoding.EncodeToString([]byte("0.0002"))
			extra = hex.EncodeToString([]byte(extra))
			state = ActionStateInitial
		}
		out := testBuildOutput(group, "0.0001", extra, SafeUtxoStateUnspent, uint64(start+i+1))

		act := out.Action(state, "", "")
		err := group.store.writeAction(ctx, act, []*Transaction{})
		assert.Nil(err)
	}

	as, err := group.store.ListActions(ctx, ActionStateInitial, 0)
	assert.Nil(err)
	assert.Len(as, 1)
	as, err = group.store.ListActions(ctx, ActionStateDone, 0)
	assert.Nil(err)
	assert.Len(as, OutputsBatchSize)
}

func testBuildOutput(group *Group, amount string, extra string, state SafeUtxoState, sequence uint64) *UnifiedOutput {
	oid := mixin.UniqueConversationID("output", fmt.Sprintf("%s:%s:%s:%s:%d", amount, extra, extra, state, sequence))
	rid := mixin.UniqueConversationID("request", oid)
	hash := crypto.Sha256Hash(uuid.FromStringOrNil(oid).Bytes())
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		panic(err)
	}

	return &UnifiedOutput{
		OutputId:           oid,
		RequestId:          rid,
		TransactionHash:    hash,
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

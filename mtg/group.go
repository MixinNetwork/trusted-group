package mtg

import (
	"context"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/shopspring/decimal"
)

const (
	groupGenesisId   = "group-genesis-id"
	groupBootSynced  = "group-boot-synced"
	defaultKernelRPC = "https://kernel.mixin.dev"
)

type Worker interface {
	// process the action in a queue and return transactions
	// need to ensure enough balance with CheckAssetBalanceAt(ctx, a)
	// before return any transactions, otherwise the transactions
	// will be ignored when issuficient balance
	//
	// if we want to make a multi process worker, it's possible that
	// we pass some RPC handle to the process, or we could build a
	// whole state of the current sequence and send it to the process
	// i.e. ProcessOutput(StateAtSequence, Action) []*Transaction
	ProcessOutput(context.Context, *Action) ([]*Transaction, string)
}

type Group struct {
	mixin        *mixin.Client
	store        *SQLite3Store
	workers      map[string]Worker
	entries      map[string]string
	groupSize    int
	waitDuration time.Duration

	id              string
	GroupId         string
	rawMembers      []string
	threshold       int
	index           int
	epoch           uint64
	spendPrivateKey string
	debug           bool
	kernelRPC       string
}

func BuildGroup(ctx context.Context, store *SQLite3Store, conf *Configuration) (*Group, error) {
	if cg := conf.Genesis; len(cg.Members) < cg.Threshold || cg.Threshold < 1 {
		return nil, fmt.Errorf("invalid group threshold %d %d", len(cg.Members), cg.Threshold)
	}
	if !strings.Contains(strings.Join(conf.Genesis.Members, ","), conf.App.AppId) {
		return nil, fmt.Errorf("app %s not belongs to the group", conf.App.AppId)
	}

	client, err := mixin.NewFromKeystore(&mixin.Keystore{
		AppID:             conf.App.AppId,
		SessionID:         conf.App.SessionId,
		SessionPrivateKey: conf.App.SessionPrivateKey,
		ServerPublicKey:   conf.App.ServerPublicKey,
	})
	if err != nil {
		return nil, err
	}
	if !CheckTestEnvironment(ctx) {
		_, err := client.UserMe(ctx)
		if err != nil {
			return nil, err
		}
	}

	id := generateGenesisId(conf)
	grp := &Group{
		mixin:           client,
		store:           store,
		spendPrivateKey: conf.App.SpendPrivateKey,
		id:              id,
		GroupId:         UniqueId(id, conf.Project),
		groupSize:       conf.GroupSize,
		waitDuration:    time.Duration(conf.LoopWaitDuration),
		workers:         make(map[string]Worker),
		entries:         make(map[string]string),
		kernelRPC:       defaultKernelRPC,
		index:           -1,
	}
	if grp.groupSize <= 0 {
		grp.groupSize = OutputsBatchSize
	}

	oid, err := store.ReadProperty(ctx, groupGenesisId)
	if err != nil {
		return nil, err
	}
	if len(oid) > 0 && string(oid) != grp.id {
		return nil, fmt.Errorf("malformed group genesis id %s %s", string(oid), grp.id)
	}
	err = store.WriteProperty(ctx, groupGenesisId, grp.id)
	if err != nil {
		return nil, err
	}

	err = store.WriteProperty(ctx, groupBootSynced, "0")
	if err != nil {
		return nil, err
	}

	for _, id := range conf.Genesis.Members {
		err = grp.AddNode(ctx, id, conf.Genesis.Threshold, conf.Genesis.Epoch)
		if err != nil {
			return nil, err
		}
	}
	members, threshold, epoch, err := grp.ListActiveNodes(ctx)
	if err != nil {
		return nil, err
	}
	sort.Strings(members)

	grp.rawMembers = members
	grp.threshold = threshold
	grp.index = grp.calculateIndex()
	grp.epoch = epoch
	return grp, nil
}

func (grp *Group) GenesisId() string {
	return grp.id
}

func (grp *Group) GetMembers() []string {
	ms := make([]string, len(grp.rawMembers))
	n := copy(ms, grp.rawMembers)
	if len(grp.rawMembers) != n {
		panic(n)
	}
	if grp.debug {
		sort.Strings(ms)
		if !slices.Equal(ms, grp.rawMembers) {
			panic(ms)
		}
	}
	return ms
}

func (grp *Group) GetThreshold() int {
	return grp.threshold
}

func (grp *Group) Index() int {
	if grp.index < 0 {
		panic(grp.index)
	}
	return grp.index
}

func (grp *Group) EnableDebug() {
	grp.debug = true
}

func (grp *Group) SetKernelRPC(rpc string) {
	grp.kernelRPC = rpc
}

func (grp *Group) Synced(ctx context.Context) bool {
	v, err := grp.store.ReadProperty(ctx, groupBootSynced)
	if err != nil {
		panic(err)
	}
	return v == "1"
}

func (grp *Group) AttachWorker(appId string, wkr Worker) {
	if grp.FindWorker(appId) != nil {
		panic(appId)
	}
	grp.workers[appId] = wkr
}

func (grp *Group) RegisterDepositEntry(appId string, entry DepositEntry) {
	key := entry.UniqueKey()
	if grp.FindWorker(appId) == nil || grp.FindAppByEntry(key) != "" {
		panic(appId)
	}
	grp.entries[key] = appId
}

func (grp *Group) FindWorker(appId string) Worker {
	return grp.workers[appId]
}

func (grp *Group) FindAppByEntry(entry string) string {
	return grp.entries[entry]
}

func (grp *Group) calculateIndex() int {
	for i, id := range grp.GetMembers() {
		if grp.mixin.ClientID == id {
			return i
		}
	}
	panic(grp.mixin.ClientID)
}

func (grp *Group) Run(ctx context.Context) {
	logger.Printf("Group(%s, %d, %s).Run(v0.9.6)\n", mixinnet.HashMembers(grp.GetMembers()), grp.threshold, grp.GenesisId())
	filter := make(map[string]bool)
	for {
		time.Sleep(grp.waitDuration)
		// drain all the utxos in the order of sequence
		logger.Verbosef("Group.Run(drainOutputsFromNetwork) created\n")
		grp.drainOutputsFromNetwork(ctx, filter, 500)
		err := grp.store.WriteProperty(ctx, groupBootSynced, "1")
		if err != nil {
			panic(err)
		}

		// handle the utxos queue by sequence
		logger.Verbosef("Group.Run(handleActionsQueue)\n")
		err = grp.handleActionsQueue(ctx)
		if err != nil {
			panic(err)
		}

		// sign any possible transactions from BuildTransaction
		logger.Verbosef("Group.Run(signTransactions)\n")
		err = grp.signTransactions(ctx)
		if err != nil {
			panic(err)
		}

		// verify all transactions
		logger.Verbosef("Group.Run(publishTransactions)\n")
		err = grp.publishTransactions(ctx)
		if err != nil {
			panic(err)
		}

		// verify all withdrawal transactions
		logger.Verbosef("Group.Run(confirmWithdrawalTransactions)\n")
		err = grp.confirmWithdrawalTransactions(ctx)
		if err != nil {
			panic(err)
		}
	}
}

func (grp *Group) ListOutputsForAsset(ctx context.Context, appId, assetId string, consumedUntil, sequence uint64, state SafeUtxoState, limit int) []*UnifiedOutput {
	outputs, err := grp.store.ListOutputsForAsset(ctx, appId, assetId, consumedUntil, sequence, state, limit)
	if err != nil {
		panic(err)
	}
	return outputs
}

func (grp *Group) ListOutputsForTransaction(ctx context.Context, traceId string, sequence uint64) []*UnifiedOutput {
	outputs, err := grp.store.ListOutputsForTransaction(ctx, traceId, sequence)
	if err != nil {
		panic(err)
	}
	return outputs
}

func (grp *Group) ListOutputsByTransactionHash(ctx context.Context, hash string, sequence uint64) []*UnifiedOutput {
	outputs, err := grp.store.ListOutputsByTransactionHash(ctx, hash, sequence)
	if err != nil {
		panic(err)
	}
	return outputs
}

func (grp *Group) ListUnconfirmedWithdrawalTransactions(ctx context.Context, limit int) []*Transaction {
	txs, err := grp.store.ListUnconfirmedWithdrawalTransactions(ctx, limit)
	if err != nil {
		panic(err)
	}
	return txs
}

// this function or rpc should be used only in ProcessOutput
func (act *Action) CheckAssetBalanceAt(ctx context.Context, assetId string) decimal.Decimal {
	os, err := act.group.store.ListOutputsForAsset(ctx, act.AppId, assetId, act.consumed[assetId], act.Sequence, SafeUtxoStateUnspent, OutputsBatchSize)
	if err != nil {
		panic(err)
	}

	total := decimal.NewFromInt(0)
	for _, o := range os {
		total = total.Add(o.Amount)
	}
	return total
}

func (act *Action) CheckAssetBalanceForStorageAt(ctx context.Context, extra []byte) bool {
	if len(extra) > common.ExtraSizeStorageCapacity {
		panic(fmt.Errorf("too large extra %d > %d", len(extra), common.ExtraSizeStorageCapacity))
	}

	amount := getStorageTransactionAmount(extra)
	total := act.CheckAssetBalanceAt(ctx, StorageAssetId)
	return common.NewIntegerFromString(total.String()).Cmp(amount) > 0
}

func (grp *Group) signTransactionWithAsset(ctx context.Context, wg *sync.WaitGroup, asset string, txs []*Transaction) {
	logger.Verbosef("Group.signTransactionWithAsset(%s)", asset)
	defer wg.Done()

	for _, tx := range txs {
		ver := grp.signTransaction(ctx, tx)
		if ver == nil {
			break
		}
		logger.Verbosef("Group.signTransaction(%v) => %s", *tx, hex.EncodeToString(ver.Marshal()))
	}
}

func (grp *Group) signTransactions(ctx context.Context) error {
	_, assetTxMap, err := grp.store.ListTransactions(ctx, TransactionStateInitial, 0)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	for asset, txs := range assetTxMap {
		wg.Add(1)
		go grp.signTransactionWithAsset(ctx, &wg, asset, txs)
	}
	wg.Wait()
	return nil
}

func (grp *Group) publishTransactions(ctx context.Context) error {
	txs, _, err := grp.store.ListTransactions(ctx, TransactionStateSigned, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		snapshot, err := grp.snapshotTransaction(ctx, tx)
		if err != nil {
			return err
		} else if !snapshot {
			continue
		}
		err = grp.store.FinishTransaction(ctx, tx.TraceId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (grp *Group) snapshotTransaction(ctx context.Context, tx *Transaction) (bool, error) {
	req, err := grp.readTransactionUntilSufficient(ctx, tx.RequestID())
	logger.Verbosef("group.readTransactionUntilSufficient(%s, %s) => %v", tx.TraceId, tx.RequestID(), err)
	if err != nil || req == nil {
		return false, err
	}
	if req.TransactionHash != tx.Hash.String() {
		panic(tx.TraceId)
	}
	return req.State == mixin.SafeUtxoStateSpent, nil
}

func (grp *Group) confirmWithdrawalTransactions(ctx context.Context) error {
	txs, err := grp.store.ListUnconfirmedWithdrawalTransactions(ctx, 100)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		// TODO get withdrawal hash from api
		hash := ""
		err = grp.store.ConfirmWithdrawalTransaction(ctx, tx.TraceId, hash)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateGenesisId(conf *Configuration) string {
	sort.Slice(conf.Genesis.Members, func(i, j int) bool {
		return conf.Genesis.Members[i] < conf.Genesis.Members[j]
	})
	id := strings.Join(conf.Genesis.Members, "")
	id = fmt.Sprintf("%s:%d:%d", id, conf.Genesis.Threshold, conf.Genesis.Epoch)
	return crypto.Sha256Hash([]byte(id)).String()
}

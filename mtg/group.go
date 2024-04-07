package mtg

import (
	"context"
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
)

const (
	groupGenesisId  = "group-genesis-id"
	groupBootSynced = "group-boot-synced"
)

type Worker interface {
	// handle the output and a true return value interrupts workers loop
	ProcessOutput(context.Context, *Output) bool
}

type Group struct {
	mixin        *mixin.Client
	store        *SQLite3Store
	workers      []Worker
	grouper      func(*Output) string
	groupSize    int
	waitDuration time.Duration

	clock           *Clock
	id              string
	members         []string
	epoch           uint64
	threshold       int
	SpendPrivateKey string
}

func BuildGroup(ctx context.Context, conf *Configuration) (*Group, error) {
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
	_, err = client.UserMe(ctx)
	if err != nil {
		return nil, err
	}

	store, err := OpenSQLite3Store(conf.StoreDir + "/mtg.sqlite3")
	if err != nil {
		return nil, err
	}
	defer store.Close()

	grp := &Group{
		mixin:           client,
		store:           store,
		SpendPrivateKey: conf.App.SpendPrivateKey,
		id:              generateGenesisId(conf),
		groupSize:       conf.GroupSize,
		waitDuration:    time.Duration(conf.LoopWaitDuration),
	}
	if grp.groupSize <= 0 {
		grp.groupSize = OutputsBatchSize
	}

	clock, err := NewClock(ctx, store)
	if err != nil {
		return nil, err
	}
	grp.clock = clock

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
	grp.members = members
	grp.threshold = threshold
	grp.epoch = epoch
	return grp, nil
}

func (grp *Group) SetOutputGrouper(per func(out *Output) string) {
	grp.grouper = per
}

func (grp *Group) GenesisId() string {
	return grp.id
}

func (grp *Group) GetMembers() []string {
	return grp.members
}

func (grp *Group) GetThreshold() int {
	return grp.threshold
}

func (grp *Group) Synced(ctx context.Context) (bool, error) {
	v, err := grp.store.ReadProperty(ctx, groupBootSynced)
	return v == "1", err
}

func (grp *Group) AddWorker(wkr Worker) {
	grp.workers = append(grp.workers, wkr)
}

func (node *Group) Index() int {
	for i, id := range node.members {
		if node.mixin.ClientID == id {
			return i
		}
	}
	panic(node.mixin.ClientID)
}

func (grp *Group) Run(ctx context.Context) {
	logger.Printf("Group(%s, %d, %s).Run(v0.6.1)\n", mixinnet.HashMembers(grp.members), grp.threshold, grp.GenesisId())
	filter := make(map[string]bool)
	for {
		time.Sleep(grp.waitDuration)
		// drain all the utxos in the order of sequence
		logger.Verbosef("Group.Run(drainOutputsFromNetwork) created\n")
		grp.drainOutputsFromNetwork(ctx, filter, 500)
		grp.store.WriteProperty(ctx, groupBootSynced, "1")

		// handle the utxos queue by sequence
		logger.Verbosef("Group.Run(handleActionsQueue)\n")
		grp.handleActionsQueue(ctx)

		// because some utxos are unlocked for these signing transactions
		logger.Verbosef("Group.Run(unlockExpiredTransactions)\n")
		grp.unlockExpiredTransactions(ctx)

		// sign any possible transactions from BuildTransaction
		logger.Verbosef("Group.Run(signTransactions)\n")
		grp.signTransactions(ctx)

		// verify all transactions
		logger.Verbosef("Group.Run(publishTransactions)\n")
		grp.verifyTransactions(ctx)
	}
}

func (grp *Group) ListOutputsForAsset(ctx context.Context, groupId, assetId string, state, limit int) ([]*Output, error) {
	outputs, err := grp.store.ListOutputsForAsset(ctx, groupId, assetId, state, limit)
	if err != nil {
		return nil, err
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].CreatedAt.Before(outputs[j].CreatedAt) })
	return outputs, nil
}

func (grp *Group) ListOutputsForTransaction(ctx context.Context, traceId string) ([]*Output, error) {
	outputs, err := grp.store.ListOutputsForTransaction(ctx, traceId)
	if err != nil {
		return nil, err
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].CreatedAt.Before(outputs[j].CreatedAt) })
	return outputs, nil
}

func (grp *Group) signTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(ctx, TransactionStateInitial, 0)
	if err != nil {
		return err
	}

	for _, tx := range txs {
		raw, err := grp.signTransaction(ctx, tx)
		logger.Verbosef("Group.signTransaction(%v) => %s %v", *tx, hex.EncodeToString(raw), err)
		if err != nil {
			continue
		}
		ver, _ := common.UnmarshalVersionedTransaction(raw)
		tx.Raw = raw
		tx.Hash = ver.PayloadHash()
		tx.UpdatedAt = grp.clock.Now(ctx)
		tx.State = TransactionStateSigning

		if ver.AggregatedSignature != nil || len(ver.SignaturesMap) > 0 {
			tx.State = TransactionStateSigned
		} else {
			p := DecodeMixinExtra(string(ver.Extra))
			if p.T.String() != tx.TraceId {
				panic(hex.EncodeToString(raw))
			}
			if p.G != tx.GroupId {
				panic(hex.EncodeToString(raw))
			}
		}

		grp.writeTansactionOrPanic(ctx, tx)
	}

	return nil
}

func (grp *Group) unlockExpiredTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(ctx, TransactionStateSigning, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		outputs, err := grp.ListOutputsForTransaction(ctx, tx.TraceId)
		if err != nil {
			return err
		}
		if len(outputs) > 0 && outputs[0].SignedBy == tx.Hash.String() {
			continue
		}
		// FIXME do more check about compaction transaction
		if tx.Memo == CompactionTransactionMemo {
			err := grp.store.DeleteTransaction(ctx, tx)
			logger.Verbosef("Group.deleteTransaction(%v) => %v", *tx, err)
			if err != nil {
				return err
			}
			continue
		}
		tx.State = TransactionStateInitial
		tx.Hash = crypto.Hash{}
		tx.Raw = nil
		grp.writeTansactionOrPanic(ctx, tx)
		logger.Verbosef("Group.unlockTransaction(%v)", *tx)
	}
	return nil
}

func (grp *Group) verifyTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(ctx, TransactionStateSigned, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		snapshot, err := grp.snapshotTransaction(ctx, tx.Hash)
		if err != nil {
			return err
		} else if !snapshot {
			continue
		}
		tx.State = TransactionStateSnapshot
		grp.writeTansactionOrPanic(ctx, tx)
	}
	return nil
}

func (grp *Group) snapshotTransaction(ctx context.Context, tx crypto.Hash) (bool, error) {
	c := mixinnet.NewClient(mixinnet.Config{
		Safe: true,
	})
	_, err := c.GetTransaction(ctx, mixinnet.Hash(tx))
	logger.Verbosef("Group.GetTransaction(%s) => %v", tx.String(), err)
	if err != nil {
		return false, err
	}
	return true, nil
}

func generateGenesisId(conf *Configuration) string {
	sort.Slice(conf.Genesis.Members, func(i, j int) bool {
		return conf.Genesis.Members[i] < conf.Genesis.Members[j]
	})
	id := strings.Join(conf.Genesis.Members, "")
	id = fmt.Sprintf("%s:%d:%d", id, conf.Genesis.Threshold, conf.Genesis.Epoch)
	return crypto.Sha256Hash([]byte(id)).String()
}

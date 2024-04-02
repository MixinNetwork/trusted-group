package mtg

import (
	"bytes"
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
)

const (
	groupGenesisId  = "group-genesis-id"
	groupBootSynced = "group-boot-synced"
)

type Group struct {
	mixin        *mixin.Client
	store        Store
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

func BuildGroup(ctx context.Context, store Store, conf *Configuration) (*Group, error) {
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

	clock, err := NewClock(store)
	if err != nil {
		return nil, err
	}
	grp.clock = clock

	oid, err := store.ReadProperty([]byte(groupGenesisId))
	if err != nil {
		return nil, err
	}
	if len(oid) > 0 && string(oid) != grp.id {
		return nil, fmt.Errorf("malformed group genesis id %s %s", string(oid), grp.id)
	}
	err = store.WriteProperty([]byte(groupGenesisId), []byte(grp.id))
	if err != nil {
		return nil, err
	}

	err = store.WriteProperty([]byte(groupBootSynced), []byte{0})
	if err != nil {
		return nil, err
	}

	for _, id := range conf.Genesis.Members {
		err = grp.AddNode(id, conf.Genesis.Threshold, conf.Genesis.Epoch)
		if err != nil {
			return nil, err
		}
	}
	members, threshold, epoch, err := grp.ListActiveNodes()
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

func (grp *Group) Synced() (bool, error) {
	v, err := grp.store.ReadProperty([]byte(groupBootSynced))
	return bytes.Equal(v, []byte{1}), err
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
	logger.Printf("Group(%s, %d, %s).Run(v0.6.1)\n", strings.Join(grp.members, ","), grp.threshold, grp.GenesisId())
	filter := make(map[string]bool)
	for {
		time.Sleep(grp.waitDuration)
		// drain all the utxos in the order of sequence
		logger.Verbosef("Group.Run(drainOutputsFromNetwork) created\n")
		grp.drainOutputsFromNetwork(ctx, filter, 500)
		grp.store.WriteProperty([]byte(groupBootSynced), []byte{1})

		// handle the utxos queue by sequence
		logger.Verbosef("Group.Run(handleActionsQueue)\n")
		grp.handleActionsQueue(ctx)

		// because some utxos are unlocked for these signing transactions
		logger.Verbosef("Group.Run(unlockExpiredTransactions)\n")
		grp.unlockExpiredTransactions(ctx)

		// sign any possible transactions from BuildTransaction
		logger.Verbosef("Group.Run(signTransactions)\n")
		grp.signTransactions(ctx)
	}
}

func (grp *Group) ListOutputsForAsset(groupId, assetId, state string, limit int) ([]*Output, error) {
	outputs, err := grp.store.ListOutputsForAsset(groupId, state, assetId, limit)
	if err != nil {
		return nil, err
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].CreatedAt.Before(outputs[j].CreatedAt) })
	return outputs, nil
}

func (grp *Group) ListOutputsForTransaction(traceId string) ([]*Output, error) {
	outputs, err := grp.store.ListOutputsForTransaction(traceId)
	if err != nil {
		return nil, err
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].CreatedAt.Before(outputs[j].CreatedAt) })
	return outputs, nil
}

func (grp *Group) signTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(TransactionStateInitial, 0)
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
		tx.UpdatedAt = grp.clock.Now()
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

		grp.writeTansactionOrPanic(tx)
	}

	return nil
}

func (grp *Group) unlockExpiredTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(TransactionStateSigning, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		outputs, err := grp.ListOutputsForTransaction(tx.TraceId)
		if err != nil {
			return err
		}
		if len(outputs) > 0 && outputs[0].SignedBy == tx.Hash.String() {
			continue
		}
		// FIXME do more check about compaction transaction
		if tx.Memo == CompactionTransactionMemo {
			err := grp.store.DeleteTransaction(tx)
			logger.Verbosef("Group.deleteTransaction(%v) => %v", *tx, err)
			if err != nil {
				return err
			}
			continue
		}
		tx.State = TransactionStateInitial
		tx.Hash = crypto.Hash{}
		tx.Raw = nil
		grp.writeTansactionOrPanic(tx)
		logger.Verbosef("Group.unlockTransaction(%v)", *tx)
	}
	return nil
}

func (grp *Group) publishTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(TransactionStateSigned, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		snapshot, err := grp.snapshotTransaction(ctx, tx.Hash, tx.Raw)
		if err != nil {
			return err
		} else if !snapshot {
			continue
		}
		tx.State = TransactionStateSnapshot
		grp.writeTansactionOrPanic(tx)
	}
	return nil
}

func (grp *Group) snapshotTransaction(ctx context.Context, tx crypto.Hash, b []byte) (bool, error) {
	// FIXME remove this storage transaction hack
	if tx.String() == "5b4ce1833fffd87b837e67dfffc38d5bcce93266da74756763bcf873845071ae" {
		return true, nil
	}
	raw := hex.EncodeToString(b)
	h, err := grp.mixin.SendRawTransaction(ctx, raw)
	logger.Verbosef("Group.snapshotTransaction(%s) => %s, %v", raw, h, err)
	if err != nil {
		return false, err
	}
	s, err := grp.mixin.GetRawTransaction(ctx, *h)
	if err != nil {
		return false, err
	}
	return s.Snapshot != nil && s.Snapshot.HasValue(), nil
}

func generateGenesisId(conf *Configuration) string {
	sort.Slice(conf.Genesis.Members, func(i, j int) bool {
		return conf.Genesis.Members[i] < conf.Genesis.Members[j]
	})
	id := strings.Join(conf.Genesis.Members, "")
	id = fmt.Sprintf("%s:%d:%d", id, conf.Genesis.Threshold, conf.Genesis.Timestamp)
	return crypto.Sha256Hash([]byte(id)).String()
}

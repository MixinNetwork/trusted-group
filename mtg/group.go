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
	"github.com/fox-one/mixin-sdk-go"
)

const (
	groupGenesisId = "group-genesis-id"
)

type Group struct {
	mixin     *mixin.Client
	store     Store
	workers   []Worker
	grouper   func(*Output) string
	groupSize int

	clock     *Clock
	id        string
	members   []string
	epoch     time.Time
	threshold int
	pin       string
}

func BuildGroup(ctx context.Context, store Store, conf *Configuration) (*Group, error) {
	if cg := conf.Genesis; len(cg.Members) < cg.Threshold || cg.Threshold < 1 {
		return nil, fmt.Errorf("invalid group threshold %d %d", len(cg.Members), cg.Threshold)
	}
	if !strings.Contains(strings.Join(conf.Genesis.Members, ","), conf.App.ClientId) {
		return nil, fmt.Errorf("app %s not belongs to the group", conf.App.ClientId)
	}

	s := &mixin.Keystore{
		ClientID:   conf.App.ClientId,
		SessionID:  conf.App.SessionId,
		PrivateKey: conf.App.PrivateKey,
		PinToken:   conf.App.PinToken,
	}
	client, err := mixin.NewFromKeystore(s)
	if err != nil {
		return nil, err
	}
	err = client.VerifyPin(ctx, conf.App.PIN)
	if err != nil {
		return nil, err
	}

	grp := &Group{
		mixin:     client,
		store:     store,
		pin:       conf.App.PIN,
		id:        generateGenesisId(conf),
		groupSize: conf.GroupSize,
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

	for _, id := range conf.Genesis.Members {
		ts := time.Unix(0, conf.Genesis.Timestamp)
		err = grp.AddNode(id, conf.Genesis.Threshold, ts)
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

func (grp *Group) AddWorker(wkr Worker) {
	grp.workers = append(grp.workers, wkr)
}

func (grp *Group) Run(ctx context.Context) {
	logger.Printf("Group(%s, %d, %s).Run(v0.2.2)\n", mixin.HashMembers(grp.members), grp.threshold, grp.GenesisId())
	filter := make(map[string]bool)
	for {
		// drain all the utxos in the order of created time
		grp.drainOutputsFromNetwork(ctx, filter, 500, "created")
		grp.drainOutputsFromNetwork(ctx, filter, 500, "updated")

		// handle the utxos queue by created time
		grp.handleActionsQueue(ctx)

		// sing any possible transactions from BuildTransaction
		grp.signTransactions(ctx)

		// publish all signed transactions to the mainnet
		grp.publishTransactions(ctx)

		grp.signCollectibleTransactions(ctx)
		grp.publishCollectibleTransactions(ctx)
	}
}

func (grp *Group) ListOutputsForAsset(groupId, assetId, state string, limit int) ([]*Output, error) {
	return grp.store.ListOutputsForAsset(groupId, state, assetId, limit)
}

// FIXME sign one transaction per loop, slow
func (grp *Group) signTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(TransactionStateInitial, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	tx := txs[0]
	for _, ct := range txs {
		// because we rely on the updated time of outputs, then build
		// transaction can result in different order, so sign the first
		// signed transaction by others at first
		outs, err := grp.store.ListOutputsForTransaction(ct.TraceId)
		if err != nil {
			return err
		}
		if len(outs) > 0 {
			tx = ct
			break
		}
	}
	raw, err := grp.signTransaction(ctx, tx)
	logger.Verbosef("Group.signTransaction(%v) => %s %v", *tx, hex.EncodeToString(raw), err)
	if err != nil {
		return err
	}
	ver, _ := common.UnmarshalVersionedTransaction(raw)
	tx.Raw = raw
	tx.Hash = ver.PayloadHash()
	tx.UpdatedAt = grp.clock.Now()
	tx.State = TransactionStateSigning

	p := DecodeMixinExtra(string(ver.Extra))
	if p.T.String() != tx.TraceId {
		panic(hex.EncodeToString(raw))
	}
	if p.G != tx.GroupId {
		panic(hex.EncodeToString(raw))
	}

	return grp.store.WriteTransaction(tx)
}

func (grp *Group) publishTransactions(ctx context.Context) error {
	txs, err := grp.store.ListTransactions(TransactionStateSigned, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		snapshot, err := grp.snapshotTransaction(ctx, tx.Raw)
		if err != nil {
			return err
		} else if !snapshot {
			continue
		}
		tx.State = TransactionStateSnapshot
		err = grp.store.WriteTransaction(tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (grp *Group) signCollectibleTransactions(ctx context.Context) error {
	txs, err := grp.store.ListCollectibleTransactions(TransactionStateInitial, 1)
	if err != nil || len(txs) != 1 {
		return err
	}
	tx := txs[0]
	raw, err := grp.signCollectibleTransaction(ctx, tx)
	logger.Verbosef("Group.signCollectibleTransaction(%v) => %s %v", *tx, hex.EncodeToString(raw), err)
	if err != nil {
		return err
	}
	ver, _ := common.UnmarshalVersionedTransaction(raw)
	tx.Raw = raw
	tx.Hash = ver.PayloadHash()
	tx.UpdatedAt = grp.clock.Now()
	tx.State = TransactionStateSigning

	nfm, err := DecodeNFOMemo(ver.Extra)
	if err != nil {
		panic(hex.EncodeToString(raw))
	} else if nfm.WillMint() && nfoTraceId(ver.Extra) != tx.TraceId {
		panic(hex.EncodeToString(raw))
	}

	return grp.store.WriteCollectibleTransaction(tx.TraceId, tx)
}

func (grp *Group) publishCollectibleTransactions(ctx context.Context) error {
	txs, err := grp.store.ListCollectibleTransactions(TransactionStateSigned, 0)
	if err != nil || len(txs) == 0 {
		return err
	}
	for _, tx := range txs {
		snapshot, err := grp.snapshotTransaction(ctx, tx.Raw)
		if err != nil {
			return err
		} else if !snapshot {
			continue
		}
		tx.State = TransactionStateSnapshot
		err = grp.store.WriteCollectibleTransaction(tx.TraceId, tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (grp *Group) snapshotTransaction(ctx context.Context, b []byte) (bool, error) {
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
	return crypto.NewHash([]byte(id)).String()
}

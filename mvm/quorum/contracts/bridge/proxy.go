package main

import (
	"context"
	"math/big"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/quorum/contracts/bridge/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

// abigen --abi storage.json --pkg main --type StorageContract --out abi.go

type Proxy struct {
	*mixin.Client
	key      *mixin.Keystore
	storage  *abi.StorageContract
	bridge   *abi.BridgeContract
	registry *abi.RegistryContract
	signer   *bind.TransactOpts
}

func NewProxy(ctx context.Context, kst *mixin.Keystore, conn *ethclient.Client) *Proxy {
	client, err := mixin.NewFromKeystore(kst)
	if err != nil {
		panic(err)
	}
	ps, err := abi.NewStorageContract(common.HexToAddress(MVMStorageContract), conn)
	if err != nil {
		panic(err)
	}
	pb, err := abi.NewBridgeContract(common.HexToAddress(MVMBridgeContract), conn)
	if err != nil {
		panic(err)
	}
	pr, err := abi.NewRegistryContract(common.HexToAddress(MVMRegistryContract), conn)
	if err != nil {
		panic(err)
	}

	chainId := new(big.Int).SetInt64(GethChainId)
	priv, err := crypto.HexToECDSA(GethPrivateKey)
	if err != nil {
		panic(err)
	}
	signer, err := bind.NewKeyedTransactorWithChainID(priv, chainId)
	if err != nil {
		panic(err)
	}
	proxy := &Proxy{client, kst, ps, pb, pr, signer}
	_, err = proxy.UserMe(ctx)
	if err != nil {
		panic(err)
	}
	return proxy
}

func (p *Proxy) Run(ctx context.Context, store *Storage) {
	go func() {
		for {
			p.processSnapshots(ctx, store)
		}
	}()

	for {
		p.loopSnapshots(ctx, store)
	}
}

func (p *Proxy) loopSnapshots(ctx context.Context, store *Storage) {
	ckpt, err := store.readSnapshotsCheckpoint(ctx)
	if err != nil {
		panic(err)
	}
	snapshots, err := p.ReadNetworkSnapshots(ctx, "", ckpt, "ASC", 500)
	if err != nil {
		panic(err)
	}
	logger.Verbosef("Proxy.loopSnapshots(%s) => %d %v", ckpt, len(snapshots), err)

	for _, s := range snapshots {
		ckpt = s.CreatedAt
		if s.UserID == "" {
			continue
		}
		if s.Amount.Cmp(decimal.NewFromFloat(0.00000001)) < 0 {
			continue
		}
		logger.Verbosef("Proxy.loopSnapshots(%s) => %d %v => %v", ckpt, len(snapshots), err, *s)
		err = store.writeSnapshot(s)
		if err != nil {
			panic(err)
		}
	}

	err = store.writeSnapshotsCheckpoint(ctx, ckpt)
	if err != nil {
		panic(err)
	}
	if len(snapshots) < 500 {
		time.Sleep(time.Second * 5)
	}
}

func (p *Proxy) processSnapshots(ctx context.Context, store *Storage) {
	snapshots, err := store.listSnapshots(100)
	if err != nil {
		panic(err)
	}

	for _, s := range snapshots {
		user, err := store.readUser(s.UserID)
		if err != nil {
			panic(err)
		}
		if user == nil {
			continue
		}
		err = p.processSnapshotForUser(ctx, s, user)
		if err != nil {
			panic(err)
		}
	}

	err = store.deleteSnapshots(snapshots)
	if err != nil {
		panic(err)
	}
	if len(snapshots) < 100 {
		time.Sleep(3 * time.Second)
	}
}

func (p *Proxy) processSnapshotForUser(ctx context.Context, s *mixin.Snapshot, user *User) error {
	act, err := p.decodeAction(user, s)
	if err != nil {
		return err
	}
	if act != nil {
		return user.handle(s, act)
	}
	return user.pass(ctx, p, s)
}

package main

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fox-one/mixin-sdk-go"
)

// abigen --abi storage.json --pkg main --type StorageContract --out abi.go

type Proxy struct {
	*mixin.Client
	*mixin.Keystore
	*StorageContract
	signer *bind.TransactOpts
}

func NewProxy(kst *mixin.Keystore, conn *ethclient.Client) *Proxy {
	client, err := mixin.NewFromKeystore(kst)
	if err != nil {
		panic(err)
	}
	proc, err := NewStorageContract(common.HexToAddress(MVMStorageContract), conn)
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
	return &Proxy{client, kst, proc, signer}
}

func (p *Proxy) Run(store *Storage) {
	go p.processSnapshots(store)
	p.loopSnapshots(store)
}

func (p *Proxy) loopSnapshots(store *Storage) {
	ctx := context.Background()
	ckpt, err := store.readSnapshotsCheckpoint(ctx)
	if err != nil {
		panic(err)
	}
	snapshots, err := p.ReadNetworkSnapshots(ctx, "", ckpt, "ASC", 500)
	if err != nil {
		panic(err)
	}

	for _, s := range snapshots {
		ckpt = s.CreatedAt
		if s.UserID == "" {
			continue
		}
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

func (p *Proxy) processSnapshots(store *Storage) {
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
		err = p.processSnapshotForUser(s, user)
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

func (p *Proxy) processSnapshotForUser(s *mixin.Snapshot, user *User) error {
	act, err := p.decodeAction(user.PublicKey, s.Memo)
	if err != nil {
		return err
	}
	if act != nil {
		return user.handle(s, act)
	}
	return user.pass(s)
}

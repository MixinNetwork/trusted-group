package main

import (
	"context"
	"crypto/ed25519"
	"math/big"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/domains/ethereum"
	"github.com/fox-one/mixin-sdk-go"
)

type User struct {
	*mixin.User
	*mixin.Keystore
	PIN string `json:"-"`
}

// TODO should verify the signature from MetaMask of the addr
func (p *Proxy) createUser(ctx context.Context, store *Storage, addr string) (*User, error) {
	err := ethereum.VerifyAddress(addr)
	if err != nil {
		return nil, err
	}
	seed := crypto.NewHash([]byte(ProxyUserSecret + addr))
	signer := ed25519.NewKeyFromSeed(seed[:])
	u, ks, err := p.CreateUser(ctx, signer, addr)
	if err != nil {
		return nil, err
	}
	user := &User{u, ks, ""}

	seed = crypto.NewHash(seed[:])
	pin := new(big.Int).SetBytes(seed[:]).String()
	for len(pin) < 6 {
		pin = pin + pin
	}
	user.PIN = pin[:6]

	uc, err := mixin.NewFromKeystore(ks)
	if err != nil {
		return nil, err
	}
	err = uc.ModifyPin(ctx, "", user.PIN)
	if err != nil {
		return nil, err
	}

	err = store.writeUser(user)
	return user, err
}

func (u *User) handle(s *mixin.Snapshot, act *Action) error {
	panic(0)
}

func (u *User) pass(p *Proxy, s *mixin.Snapshot) error {
	return p.bindAndPass(s.SnapshotID, u.FullName, s.AssetID, s.Amount)
}

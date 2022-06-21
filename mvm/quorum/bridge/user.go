package main

import (
	"context"
	"crypto/ed25519"
	"math/big"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/domains/ethereum"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/fox-one/mixin-sdk-go"
)

type User struct {
	*mixin.User
	Key      *mixin.Keystore `json:"key"`
	PIN      string          `json:"-"`
	Contract string          `json:"contract"`
}

// TODO should verify the signature from MetaMask of the addr
func (p *Proxy) createUser(ctx context.Context, store *Storage, addr, sig string) (*User, error) {
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
	user := &User{u, ks, "", ""}

	seed = crypto.NewHash(seed[:])
	pin := new(big.Int).SetBytes(seed[:]).String()
	for len(pin) < 6 {
		pin = pin + pin
	}
	user.PIN = pin[:6]

	if !user.HasPin {
		uc, err := mixin.NewFromKeystore(ks)
		if err != nil {
			return nil, err
		}
		err = uc.ModifyPin(ctx, "", user.PIN)
		if err != nil {
			return nil, err
		}
	}

	err = store.writeUser(user)
	if err != nil {
		return nil, err
	}
	return p.readUser(store, user.UserID)
}

func (p *Proxy) readUser(store *Storage, id string) (*User, error) {
	user, err := store.readUser(id)
	if err != nil || user == nil {
		return nil, err
	}
	if user.Contract != "" {
		return user, nil
	}
	ua, err := user.getContract(p)
	if err != nil {
		return nil, err
	}
	user.Contract = ua.String()
	err = store.writeUser(user)
	return user, err
}

func (u *User) handle(ctx context.Context, s *mixin.Snapshot, act *Action) error {
	if act.Destination != "" {
		return u.withdraw(ctx, s, act)
	}

	traceId := mixin.UniqueConversationID(s.SnapshotID, "HANDLE||TRANSFER")
	input := &mixin.TransferInput{
		AssetID: s.AssetID,
		Amount:  s.Amount,
		TraceID: traceId,
		Memo:    act.Extra,
	}
	if len(act.Receivers) == 1 {
		input.OpponentID = act.Receivers[0]
	} else {
		input.OpponentMultisig.Receivers = act.Receivers
		input.OpponentMultisig.Threshold = uint8(act.Threshold)
	}
	return u.send(ctx, input)
}

func (u *User) pass(ctx context.Context, p *Proxy, s *mixin.Snapshot) error {
	logger.Verbosef("User.pass(%v)", *s)
	return u.bindAndPass(ctx, p, s.SnapshotID, u.FullName, s.AssetID, s.Amount)
}

func (u *User) send(ctx context.Context, in *mixin.TransferInput) error {
	uc, err := mixin.NewFromKeystore(u.Key)
	if err != nil {
		return err
	}
	_, err = uc.Transaction(ctx, in, u.PIN)
	return err
}

// TODO this wont' work as no fee
func (u *User) withdraw(ctx context.Context, s *mixin.Snapshot, act *Action) error {
	uc, err := mixin.NewFromKeystore(u.Key)
	if err != nil {
		return err
	}
	ain := mixin.CreateAddressInput{
		AssetID:     s.AssetID,
		Destination: act.Destination,
		Tag:         act.Tag,
		Label:       s.SnapshotID,
	}
	addr, err := uc.CreateAddress(ctx, ain, u.PIN)
	if err != nil {
		return err
	}

	traceId := mixin.UniqueConversationID(s.SnapshotID, "HANDLE||WITHDRAWAL")
	win := mixin.WithdrawInput{
		AddressID: addr.AddressID,
		Amount:    s.Amount,
		TraceID:   traceId,
		Memo:      act.Extra,
	}
	_, err = uc.Withdraw(ctx, win, u.PIN)
	return err
}

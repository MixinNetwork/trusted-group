package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
)

const (
	WithdrawalTimeout = 5 * time.Minute
)

type Withdrawal struct {
	TraceId     string
	Destination string
	Tag         string
	CreatedAt   time.Time
	UserId      string
	Asset       *mixin.Snapshot
	Fee         *mixin.Snapshot
}

func (u *User) submit(ctx context.Context, store *Storage, s *mixin.Snapshot, act *Action) error {
	if len(act.Extra) != 38 {
		return fmt.Errorf("invalid withdrawal trace data %s", act.Extra)
	}
	parts := strings.Split(act.Extra, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid withdrawal trace data %s", act.Extra)
	}
	if uuid.FromStringOrNil(parts[0]).String() != parts[0] {
		return fmt.Errorf("invalid withdrawal trace data %s", act.Extra)
	}
	if parts[1] != "A" && parts[1] != "B" {
		return fmt.Errorf("invalid withdrawal trace data %s", act.Extra)
	}

	asset, err := store.readAsset(s.AssetID)
	if err != nil {
		panic(err)
	}
	if parts[1] == "B" && s.AssetID != asset.ChainID {
		return fmt.Errorf("invalid withdrawal fee %v", *act)
	}

	traceId := mixin.UniqueConversationID(u.UserID, parts[0])
	w := &Withdrawal{
		TraceId:     traceId,
		Destination: act.Destination,
		Tag:         act.Tag,
		CreatedAt:   time.Now(),
		UserId:      s.UserID,
	}
	if parts[1] == "A" {
		w.Asset = s
	} else {
		w.Fee = s
	}

	old, err := store.readWithdrawalById(traceId)
	if err != nil {
		panic(err)
	}
	if old == nil {
		return store.writeWithdrawal(w)
	}
	if old.CreatedAt.Add(WithdrawalTimeout).Before(w.CreatedAt) {
		return fmt.Errorf("withdrawl pair time out %v", *s)
	}
	w.CreatedAt = old.CreatedAt

	if old.Asset != nil && old.Fee != nil {
		return fmt.Errorf("invalid withdrawal pair %v", *act)
	}
	if w.Fee == nil {
		w.Fee = old.Fee
	}
	if w.Asset == nil {
		w.Asset = old.Asset
	}
	if w.Asset == nil || w.Fee == nil {
		return fmt.Errorf("invalid withdrawal pair %v", *act)
	}
	return store.writeWithdrawal(w)
}

func (u *User) withdraw(ctx context.Context, s *mixin.Snapshot, destination, tag string) error {
	uc, err := mixin.NewFromKeystore(u.Key)
	if err != nil {
		return err
	}
	ain := mixin.CreateAddressInput{
		AssetID:     s.AssetID,
		Destination: destination,
		Tag:         tag,
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
		Memo:      s.SnapshotID,
	}
	_, err = uc.Withdraw(ctx, win, u.PIN)
	if err != nil {
		return err
	}
	return uc.DeleteAddress(ctx, ain.AssetID, u.PIN)
}

func (p *Proxy) processWithdrawals(ctx context.Context, store *Storage) {
	withdrawals, err := store.listWithdrawals(1000)
	if err != nil {
		panic(err)
	}

	for _, w := range withdrawals {
		user, err := store.readUserById(w.UserId)
		if err != nil {
			panic(err)
		}
		err = p.processWithdrawalForUser(ctx, store, w, user)
		if err != nil {
			panic(err)
		}
	}

	err = store.deleteWitdrawals(withdrawals)
	if err != nil {
		panic(err)
	}
	if len(withdrawals) < 100 {
		time.Sleep(1 * time.Second)
	}
}

func (p *Proxy) processWithdrawalForUser(ctx context.Context, store *Storage, w *Withdrawal, u *User) error {
	if w.CreatedAt.Add(WithdrawalTimeout).Before(time.Now()) {
		return u.expireWithdrawal(ctx, p, w)
	}
	if w.Asset == nil || w.Fee == nil {
		return nil
	}
	if w.Asset.UserID != u.UserID || w.Fee.UserID != u.UserID {
		panic(u.UserID)
	}
	err := u.withdraw(ctx, w.Asset, w.Destination, w.Tag)
	if err == nil {
		return nil
	}
	err = u.pass(ctx, p, w.Asset)
	if err != nil {
		return err
	}
	return u.pass(ctx, p, w.Fee)
}

func (u *User) expireWithdrawal(ctx context.Context, p *Proxy, w *Withdrawal) error {
	if w.Asset != nil {
		err := u.pass(ctx, p, w.Asset)
		if err != nil {
			return err
		}
	}
	if w.Fee != nil {
		err := u.pass(ctx, p, w.Fee)
		if err != nil {
			return err
		}
	}
	return nil
}

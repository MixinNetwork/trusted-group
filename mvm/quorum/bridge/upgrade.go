package main

import (
	"context"
	"math/big"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/trusted-group/mtg"
	"github.com/dgraph-io/badger/v3"
	"github.com/fox-one/mixin-sdk-go"
)

func upgradeUserPIN(ctx context.Context, store *Storage) {
	users, err := store.listAllUsers()
	if err != nil {
		panic(err)
	}
	for _, u := range users {
		if u.Version == CurrentUserVersion {
			continue
		}

		seed := crypto.NewHash([]byte(ProxyUserSecret + u.UserID))
		seed = crypto.NewHash(append(seed[:], ProxyUserSecret...))
		pin := new(big.Int).SetBytes(seed[:]).String()
		for len(pin) < 6 {
			pin = pin + pin
		}
		pin = pin[:6]

		uc, err := mixin.NewFromKeystore(u.Key)
		if err != nil {
			panic(err)
		}
		err = uc.ModifyPin(ctx, u.PIN, pin)
		if err != nil {
			panic(err)
		}

		u.PIN = pin
		u.HasPin = true
		u.Version = CurrentUserVersion
		err = store.writeUser(u)
		if err != nil {
			panic(err)
		}
	}
}

func (s *Storage) listAllUsers() ([]*User, error) {
	txn := s.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.Prefix = []byte(storePrefixUser)
	it := txn.NewIterator(opts)
	defer it.Close()

	users := make([]*User, 0)
	it.Seek(opts.Prefix)
	for ; it.Valid(); it.Next() {
		item := it.Item()
		v, err := item.ValueCopy(nil)
		if err != nil {
			return users, err
		}

		var user User
		err = mtg.MsgpackUnmarshal(v, &user)
		users = append(users, &user)
	}

	return users, nil
}

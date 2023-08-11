package store

import (
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
	"github.com/dgraph-io/badger/v4"
)

const (
	prefixCollectibleTokenMeta = "MVM:COLLECTIBLE:TOKEN:META:"
	prefixCollectibleOrAsset   = "MVM:OR:COLLECTIBLE:ASSET:"
)

func (bs *BadgerStore) ReadAssetOrCollectible(id string) (string, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	key := []byte(prefixCollectibleOrAsset + id)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return "", nil
	} else if err != nil {
		return "", err
	}

	val, err := item.ValueCopy(nil)
	return string(val), err
}

func (bs *BadgerStore) WriteAssetOrCollectible(id, cat string) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		key := []byte(prefixCollectibleOrAsset + id)
		val := []byte(cat)
		return txn.Set(key, val)
	})
}

func (bs *BadgerStore) ReadCollectibleToken(id string) (*machine.CollectibleToken, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	return bs.readCollectibleTokenMeta(txn, id)
}

func (bs *BadgerStore) WriteCollectibleToken(a *machine.CollectibleToken) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		old, err := bs.readCollectibleTokenMeta(txn, a.Id)
		if err != nil || old != nil {
			return err
		}
		key := buildCollectibleTokenMetaKey(a.Id)
		val := encoding.JSONMarshalPanic(a)
		return txn.Set(key, val)
	})
}

func (bs *BadgerStore) readCollectibleTokenMeta(txn *badger.Txn, id string) (*machine.CollectibleToken, error) {
	key := buildCollectibleTokenMetaKey(id)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	val, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	var a machine.CollectibleToken
	err = encoding.JSONUnmarshal(val, &a)
	return &a, err
}

func buildCollectibleTokenMetaKey(id string) []byte {
	key := prefixCollectibleTokenMeta + id
	return []byte(key)
}

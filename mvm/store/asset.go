package store

import (
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixAssetMeta = "MVM:ASSET:META:"
)

func (bs *BadgerStore) ReadAsset(id string) (*machine.Asset, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	return bs.readAssetMeta(txn, id)
}

func (bs *BadgerStore) WriteAsset(a *machine.Asset) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		old, err := bs.readAssetMeta(txn, a.Id)
		if err != nil || old != nil {
			return err
		}
		key := buildAssetMetaKey(a.Id)
		val := encoding.JSONMarshalPanic(a)
		return txn.Set(key, val)
	})
}

func (bs *BadgerStore) readAssetMeta(txn *badger.Txn, id string) (*machine.Asset, error) {
	key := buildAssetMetaKey(id)
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
	var a machine.Asset
	err = encoding.JSONUnmarshal(val, &a)
	return &a, err
}

func buildAssetMetaKey(id string) []byte {
	key := prefixAssetMeta + id
	return []byte(key)
}

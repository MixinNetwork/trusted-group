package store

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixAccountSnapshotInc = "MVM:ACCOUNT:SNAPSHOT:INC:"
	prefixAccountSnapshotDec = "MVM:ACCOUNT:SNAPSHOT:DEC:"
	prefixAccountBalance     = "MVM:ACCOUNT:BALANCE:"
)

func (bs *BadgerStore) CheckAccountSnapshot(as *machine.AccountSnapshot) (bool, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()
	if as.Credit {
		panic(as.Amount)
	}

	ask := buildAccountSnapshotKey(as)
	_, err := txn.Get(ask)
	if err == nil {
		return true, nil
	} else if err != badger.ErrKeyNotFound {
		return false, err
	}

	balance, err := bs.readAccountBalance(txn, as.Process, as.Asset)
	if err != nil {
		return false, err
	}
	return balance.Cmp(as.Amount) >= 0, nil
}

func (bs *BadgerStore) WriteAccountSnapshot(as *machine.AccountSnapshot) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		ask := buildAccountSnapshotKey(as)
		_, err := txn.Get(ask)
		if err == nil {
			return nil
		} else if err != badger.ErrKeyNotFound {
			return err
		}

		bal, err := bs.readAccountBalance(txn, as.Process, as.Asset)
		if err != nil {
			return err
		}
		if !as.Credit && bal.Cmp(as.Amount) < 0 {
			panic(bal)
		}
		if as.Credit {
			bal = bal.Add(as.Amount)
		} else {
			bal = bal.Sub(as.Amount)
		}
		key := buildAccountBalanceKey(as.Process, as.Asset)
		err = txn.Set(key, []byte(bal.String()))
		if err != nil {
			return err
		}

		return txn.Set(ask, encoding.JSONMarshalPanic(as))
	})
}

func (bs *BadgerStore) readAccountBalance(txn *badger.Txn, pid, asset string) (common.Integer, error) {
	key := buildAccountBalanceKey(pid, asset)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return common.Zero, nil
	} else if err != nil {
		return common.Zero, err
	}
	val, err := item.ValueCopy(nil)
	if err != nil {
		return common.Zero, err
	}
	return common.NewIntegerFromString(string(val)), nil
}

func buildAccountSnapshotKey(as *machine.AccountSnapshot) []byte {
	prefix := prefixAccountSnapshotDec
	if as.Credit {
		prefix = prefixAccountSnapshotInc
	}
	ask := []byte(prefix + as.Process)
	return append(ask, uint64Bytes(as.Nonce)...)
}

func buildAccountBalanceKey(pid, asset string) []byte {
	key := append([]byte(prefixAccountBalance), pid...)
	return append(key, asset...)
}

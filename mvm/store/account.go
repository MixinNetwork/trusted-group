package store

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixAccountBalance = "MVM:ACCOUNT:BALANCE:"
)

func (bs *BadgerStore) ReadAccount(pid string, asset string) (*machine.Account, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	balance, err := bs.readAccountBalance(txn, pid, asset)
	if err != nil {
		return nil, err
	}
	return &machine.Account{
		Process: pid,
		Asset:   asset,
		Balance: balance,
	}, nil
}

func (bs *BadgerStore) WriteAccountChange(pid string, asset string, amount common.Integer, credit bool) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		bal, err := bs.readAccountBalance(txn, pid, asset)
		if err != nil {
			return err
		}
		if !credit && bal.Cmp(amount) < 0 {
			panic(bal)
		}
		if credit {
			bal = bal.Add(amount)
		} else {
			bal = bal.Sub(amount)
		}
		key := buildAccountKey(pid, asset)
		return txn.Set(key, []byte(bal.String()))
	})
}

func (bs *BadgerStore) readAccountBalance(txn *badger.Txn, pid, asset string) (common.Integer, error) {
	key := buildAccountKey(pid, asset)
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

func buildAccountKey(pid, asset string) []byte {
	key := append([]byte(prefixAccountBalance), pid...)
	return append(key, asset...)
}

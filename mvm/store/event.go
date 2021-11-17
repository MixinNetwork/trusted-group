package store

import (
	"encoding/binary"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixPendingEventQueue      = "MVM:EVENT:PENDING:QUEUE:"
	prefixPendingEventSignatures = "MVM:EVENT:PENDING:SIGNATURES:"
)

func (bs *BadgerStore) WritePendingGroupEventAndNonce(event *encoding.Event) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		proc, err := bs.readProcess(txn, event.Process)
		if err != nil {
			return err
		}
		if proc.Nonce != event.Nonce {
			panic(event)
		}
		proc.Nonce = proc.Nonce + 1
		err = bs.writeProcess(txn, proc)
		if err != nil {
			return err
		}
		key := buildPendingEventTimedKey(event)
		val := common.MsgpackMarshalPanic(event)
		return txn.Set(key, val)
	})
}

func (bs *BadgerStore) ListPendingGroupEvents(limit int) ([]*encoding.Event, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixPendingEventQueue)
	it := txn.NewIterator(opts)
	defer it.Close()

	var evts []*encoding.Event
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		val, err := it.Item().ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var evt encoding.Event
		err = common.MsgpackUnmarshal(val, &evt)
		if err != nil {
			return nil, err
		}
		evts = append(evts, &evt)
		if len(evts) == limit {
			break
		}
	}
	return evts, nil
}

func (bs *BadgerStore) ReadPendingGroupEventSignatures(pid string, nonce uint64) ([][]byte, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	key := buildPendingEventSignaturesKey(pid, nonce)
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
	sigs := make([][]byte, len(val)/96)
	for i := 0; i < len(sigs); i++ {
		sigs[i] = val[i*96 : i*96+1]
	}
	return sigs, nil
}

func (bs *BadgerStore) WritePendingGroupEventSignatures(pid string, nonce uint64, partials [][]byte) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		var val []byte
		for _, p := range partials {
			if len(p) != 96 {
				panic(p)
			}
			val = append(val, p...)
		}
		key := buildPendingEventSignaturesKey(pid, nonce)
		return txn.Set(key, val)
	})
}

func (bs *BadgerStore) WriteSignedGroupEvent(event *encoding.Event) error {
	panic(0)
}

func (bs *BadgerStore) ListSignedGroupEvents(pid string, limit int) ([]*encoding.Event, error) {
	panic(0)
}

func (bs *BadgerStore) ExpireGroupEventsWithCost(events []*encoding.Event, cost common.Integer) error {
	panic(0)
}

func buildPendingEventSignaturesKey(pid string, nonce uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, nonce)
	key := append([]byte(prefixPendingEventSignatures), pid...)
	return append(key, buf...)
}

func buildPendingEventTimedKey(evt *encoding.Event) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, evt.Timestamp)
	key := append([]byte(prefixPendingEventQueue), buf...)
	key = append(key, evt.Process...)
	binary.BigEndian.PutUint64(buf, evt.Nonce)
	return append(key, buf...)
}

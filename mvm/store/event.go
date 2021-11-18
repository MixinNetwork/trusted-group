package store

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixPendingEventQueue      = "MVM:EVENT:PENDING:QUEUE:"
	prefixPendingEventSignatures = "MVM:EVENT:PENDING:SIGNATURES:"
	prefixPendingEventIdentifier = "MVM:EVENT:PENDING:IDENTIFIER:"
	prefixSignedEventQueue       = "MVM:EVENT:SIGNED:QUEUE:"
)

func (bs *BadgerStore) CheckPendingGroupEventIdentifier(id string) (bool, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	ts, err := bs.readPendingGroupEventIdentifier(txn, id)
	return ts > 0, err
}

func (bs *BadgerStore) WritePendingGroupEventAndNonce(event *encoding.Event, id string) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		if event.Timestamp <= 0 {
			panic(event.Timestamp)
		}
		ts, err := bs.readPendingGroupEventIdentifier(txn, id)
		if err != nil {
			return err
		} else if ts > 0 && ts != event.Timestamp {
			panic(id)
		}
		err = bs.writePendingGroupEventIdentifier(txn, id, event.Timestamp)
		if err != nil {
			return err
		}

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
		val := encoding.JSONMarshalPanic(event)
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
		err = encoding.JSONUnmarshal(val, &evt)
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
	sigs := make([][]byte, len(val)/66)
	for i := 0; i < len(sigs); i++ {
		sigs[i] = val[i*66 : (i+1)*66]
	}
	return sigs, nil
}

func (bs *BadgerStore) WritePendingGroupEventSignatures(pid string, nonce uint64, partials [][]byte) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		var val []byte
		for _, p := range partials {
			if len(p) != 66 {
				panic(hex.EncodeToString(p))
			}
			val = append(val, p...)
		}
		key := buildPendingEventSignaturesKey(pid, nonce)
		return txn.Set(key, val)
	})
}

func (bs *BadgerStore) WriteSignedGroupEventAndExpirePending(event *encoding.Event) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		pending := buildPendingEventTimedKey(event)
		err := txn.Delete(pending)
		if err != nil {
			return err
		}
		key := buildSignedEventTimedKey(event)
		val := encoding.JSONMarshalPanic(event)
		return txn.Set(key, val)
	})
}

func (bs *BadgerStore) ListSignedGroupEvents(pid string, limit int) ([]*encoding.Event, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = append([]byte(prefixSignedEventQueue), pid...)
	it := txn.NewIterator(opts)
	defer it.Close()

	var evts []*encoding.Event
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		val, err := it.Item().ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var evt encoding.Event
		err = encoding.JSONUnmarshal(val, &evt)
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

func (bs *BadgerStore) ExpireGroupEventsWithCost(events []*encoding.Event, cost common.Integer) error {
	if len(events) == 0 {
		return nil
	}
	pid := events[0].Process
	return bs.Badger().Update(func(txn *badger.Txn) error {
		for _, evt := range events {
			if evt.Process != pid {
				panic(evt.Process)
			}
			key := buildSignedEventTimedKey(evt)
			err := txn.Delete(key)
			if err != nil {
				return err
			}
		}
		if cost.Sign() == 0 {
			return nil
		}
		p, err := bs.readProcess(txn, pid)
		if err != nil {
			return err
		}
		p.Credit = p.Credit.Sub(cost)
		return bs.writeProcess(txn, p)
	})
}

func (bs *BadgerStore) writePendingGroupEventIdentifier(txn *badger.Txn, id string, ts uint64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, ts)
	key := append([]byte(prefixPendingEventIdentifier), id...)
	return txn.Set(key, buf)
}

func (bs *BadgerStore) readPendingGroupEventIdentifier(txn *badger.Txn, id string) (uint64, error) {
	key := append([]byte(prefixPendingEventIdentifier), id...)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	val, err := item.ValueCopy(nil)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(val), nil
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

func buildSignedEventTimedKey(evt *encoding.Event) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, evt.Nonce)
	key := append([]byte(prefixSignedEventQueue), evt.Process...)
	return append(key, buf...)
}

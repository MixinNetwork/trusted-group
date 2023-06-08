package quorum

import (
	"encoding/binary"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixQuorumContractNotifier      = "QUORUM:CONTRACT:NOTIFIER:"
	prefixQuorumContractLogOffset     = "QUORUM:CONTRACT:LOG:OFFSET:ALL"
	prefixQuorumContractEventQueue    = "QUORUM:CONTRACT:EVENT:QUEUE:"
	prefixQuorumGroupEventQueue       = "QUORUM:GROUP:EVENT:QUEUE:"
	prefixQuorumGroupEventTransaction = "QUORUM:GROUP:EVENT:TRANSACTION:"
)

func (e *Engine) storeWriteContractNotifier(address, notifier string) error {
	key := []byte(prefixQuorumContractNotifier + address)
	return e.db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err == nil {
			panic(address)
		} else if err != badger.ErrKeyNotFound {
			return err
		}
		return txn.Set(key, []byte(notifier))
	})
}

func (e *Engine) storeReadContractNotifier(address string) string {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	key := []byte(prefixQuorumContractNotifier + address)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return ""
	} else if err != nil {
		panic(err)
	}

	val, err := item.ValueCopy(nil)
	if err != nil {
		panic(err)
	}
	return string(val)
}

func (e *Engine) storeListContractAddresses() ([]string, error) {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixQuorumContractNotifier)
	it := txn.NewIterator(opts)
	defer it.Close()

	var addresses []string
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		key := string(it.Item().Key())
		addr := key[len(prefixQuorumContractNotifier):]
		addresses = append(addresses, addr)
	}
	return addresses, nil
}

func (e *Engine) storeReadContractLogsOffset() uint64 {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	key := []byte(prefixQuorumContractLogOffset)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return 0
	} else if err != nil {
		panic(err)
	}

	val, err := item.ValueCopy(nil)
	if err != nil {
		panic(err)
	}
	return binary.BigEndian.Uint64(val)
}

func (e *Engine) storeWriteContractLogsOffset(offset uint64) error {
	key := []byte(prefixQuorumContractLogOffset)
	return e.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, uint64Bytes(offset))
	})
}

func (e *Engine) storeReadLastContractEventNonce(address string) uint64 {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.Prefix = []byte(prefixQuorumContractEventQueue + address)
	opts.PrefetchValues = false
	opts.Reverse = true

	it := txn.NewIterator(opts)
	defer it.Close()

	it.Seek(append(opts.Prefix, uint64Bytes(^uint64(0))...))
	if !it.Valid() {
		return 0
	}
	val, err := it.Item().ValueCopy(nil)
	if err != nil {
		panic(err)
	}
	var evt encoding.Event
	err = encoding.JSONUnmarshal(val, &evt)
	if err != nil {
		panic(err)
	}
	return evt.Nonce
}

func (e *Engine) storeWriteContractEvent(address string, evt *encoding.Event) error {
	key := []byte(prefixQuorumContractEventQueue + address)
	key = append(key, uint64Bytes(evt.Nonce)...)
	val := encoding.JSONMarshalPanic(evt)
	return e.db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err == nil {
			return nil
		} else if err != badger.ErrKeyNotFound {
			return err
		}
		return txn.Set(key, val)
	})
}

func (e *Engine) storeListContractEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixQuorumContractEventQueue + address)
	it := txn.NewIterator(opts)
	defer it.Close()

	var events []*encoding.Event
	it.Seek(append(opts.Prefix, uint64Bytes(offset)...))
	for ; it.Valid(); it.Next() {
		val, err := it.Item().ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var evt encoding.Event
		err = encoding.JSONUnmarshal(val, &evt)
		if err != nil {
			panic(err)
		}
		events = append(events, &evt)
		if len(events) >= limit {
			break
		}
	}
	return events, nil
}

func (e *Engine) storeWriteGroupEvents(address string, events []*encoding.Event) error {
	return e.db.Update(func(txn *badger.Txn) error {
		for _, evt := range events {
			key := []byte(prefixQuorumGroupEventQueue + address)
			key = append(key, uint64Bytes(evt.Nonce)...)
			val := encoding.JSONMarshalPanic(evt)
			_, err := txn.Get(key)
			if err == nil {
				continue
			} else if err != badger.ErrKeyNotFound {
				return err
			}
			err = txn.Set(key, val)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (e *Engine) storeListGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixQuorumGroupEventQueue + address)
	it := txn.NewIterator(opts)
	defer it.Close()

	var events []*encoding.Event
	it.Seek(append(opts.Prefix, uint64Bytes(offset)...))
	for ; it.Valid(); it.Next() {
		val, err := it.Item().ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var evt encoding.Event
		err = encoding.JSONUnmarshal(val, &evt)
		if err != nil {
			panic(err)
		}
		events = append(events, &evt)
		if len(events) >= limit {
			break
		}
	}
	return events, nil
}

func (e *Engine) storeWriteGroupEventTransaction(address string, nonce uint64, txHash string) error {
	return e.db.Update(func(txn *badger.Txn) error {
		key := []byte(prefixQuorumGroupEventTransaction + address)
		key = append(key, uint64Bytes(nonce)...)
		return txn.Set(key, []byte(txHash))
	})
}

func (e *Engine) storeReadGroupEventTransaction(address string, nonce uint64) (string, error) {
	txn := e.db.NewTransaction(false)
	defer txn.Discard()

	key := []byte(prefixQuorumGroupEventTransaction + address)
	key = append(key, uint64Bytes(nonce)...)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return "", err
	} else if err != nil {
		return "", err
	}
	val, err := item.ValueCopy(nil)
	return string(val), err
}

func (e *Engine) FlushDataByOffset(address string, offset uint64) error {
	events, err := e.storeListContractEvents(address, offset, 100)
	if err != nil {
		return err
	}
	key := []byte(prefixQuorumContractEventQueue + address)
	for _, evt := range events {
		err = e.db.Update(func(txn *badger.Txn) error {
			key = append(key, uint64Bytes(evt.Nonce)...)
			return txn.Delete(key)
		})
		logger.Verbosef("FlushDataByOffset(%s, %d) => %v", address, evt.Nonce, err)
		if err != nil {
			return err
		}
	}

	logger.Verbosef("storeReadContractLogsOffset() => %d", e.storeReadContractLogsOffset())
	return e.storeWriteContractLogsOffset(offset)
}

func uint64Bytes(i uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	return buf
}

func openBadger(dir string) *badger.DB {
	opts := badger.DefaultOptions(dir)
	db, err := badger.Open(opts)
	if err != nil {
		panic(err)
	}
	return db
}

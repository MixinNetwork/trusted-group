package store

import (
	"encoding/binary"

	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixProcessPayload          = "MVM:PROCESS:PAYLOAD:"
	prefixEngineGroupEventsOffset = "MVM:ENGINE:GROUP:EVENTS:OFFSET:"
)

func (bs *BadgerStore) ReadEngineGroupEventsOffset(pid string) (uint64, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	key := append([]byte(prefixEngineGroupEventsOffset), pid...)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	val, err := item.ValueCopy(nil)
	return binary.BigEndian.Uint64(val), err
}

func (bs *BadgerStore) WriteEngineGroupEventsOffset(pid string, offset uint64) error {
	buf := uint64Bytes(offset)
	key := append([]byte(prefixEngineGroupEventsOffset), pid...)
	return bs.Badger().Update(func(txn *badger.Txn) error {
		return txn.Set(key, buf)
	})
}

func (bs *BadgerStore) ListProcesses() ([]*machine.Process, error) {
	txn := bs.Badger().NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = false
	opts.Prefix = []byte(prefixProcessPayload)
	it := txn.NewIterator(opts)
	defer it.Close()

	var procs []*machine.Process
	for it.Seek(opts.Prefix); it.Valid(); it.Next() {
		val, err := it.Item().ValueCopy(nil)
		if err != nil {
			return nil, err
		}
		var proc machine.Process
		err = encoding.JSONUnmarshal(val, &proc)
		if err != nil {
			return nil, err
		}
		procs = append(procs, &proc)
	}
	return procs, nil
}

func (bs *BadgerStore) WriteProcess(p *machine.Process) error {
	return bs.Badger().Update(func(txn *badger.Txn) error {
		old, err := bs.readProcess(txn, p.Identifier)
		if err != nil || old != nil {
			return err
		}
		return bs.writeProcess(txn, p)
	})
}

func (bs *BadgerStore) writeProcess(txn *badger.Txn, p *machine.Process) error {
	key := []byte(prefixProcessPayload + p.Identifier)
	val := encoding.JSONMarshalPanic(p)
	return txn.Set(key, val)
}

func (bs *BadgerStore) readProcess(txn *badger.Txn, pid string) (*machine.Process, error) {
	key := []byte(prefixProcessPayload + pid)
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
	var proc machine.Process
	err = encoding.JSONUnmarshal(val, &proc)
	return &proc, err
}

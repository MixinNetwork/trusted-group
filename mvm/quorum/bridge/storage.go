package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/dgraph-io/badger/v3"
	"github.com/fox-one/mixin-sdk-go"
)

const (
	storePrefixUser                        = "USER:"
	storePrefixAsset                       = "ASSET:"
	storePrefixAddress                     = "ADDRESS:"
	storePrefixSnapshotList                = "SNAPSHOT:LIST:"
	storePrefixSnapshotCheckpoint          = "SNAPSHOT:CHECKPOINT"
	storePrefixWithdrawalPair              = "WITHDRAWAL:PAIR:"
	storePrefixCollectibleOutputCheckpoint = "COLLECTIBLE:OUTPUT:CHECKPOINT"
	storePrefixCollectibleOutputList       = "COLLECTIBLE:OUTPUT:LIST:"
	storePrefixCollectibleRawTransaction   = "COLLECTIBLE:RAW:TRANSACTION:"
)

type Storage struct {
	*badger.DB
}

func OpenStorage(path string) (*Storage, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}
	return &Storage{db}, nil
}

func (s *Storage) close() error {
	return s.Close()
}

func (s *Storage) writeAsset(a *mixin.Asset) error {
	return s.Update(func(txn *badger.Txn) error {
		key := []byte(storePrefixAsset + a.AssetID)
		val := common.MsgpackMarshalPanic(a)
		return txn.Set(key, val)
	})
}

func (s *Storage) readAsset(id string) (*mixin.Asset, error) {
	txn := s.NewTransaction(false)
	defer txn.Discard()

	key := []byte(storePrefixAsset + id)
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
	var a mixin.Asset
	err = common.MsgpackUnmarshal(val, &a)
	return &a, err
}

func (s *Storage) readSnapshotsCheckpoint(ctx context.Context) (time.Time, error) {
	key := []byte(storePrefixSnapshotCheckpoint)
	return s.readCheckpoint(ctx, key)
}

func (s *Storage) writeSnapshotsCheckpoint(ctx context.Context, ckpt time.Time) error {
	key := []byte(storePrefixSnapshotCheckpoint)
	return s.writeCheckpoint(ctx, key, ckpt)
}

func (s *Storage) readCheckpoint(ctx context.Context, key []byte) (time.Time, error) {
	txn := s.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return time.Now(), nil
	} else if err != nil {
		return time.Time{}, err
	}
	val, err := item.ValueCopy(nil)
	if err != nil {
		return time.Time{}, err
	}
	ckpt := binary.BigEndian.Uint64(val)
	return time.Unix(0, int64(ckpt)), nil
}

func (s *Storage) writeCheckpoint(ctx context.Context, key []byte, ckpt time.Time) error {
	return s.Update(func(txn *badger.Txn) error {
		val := timeToBytes(ckpt)
		return txn.Set(key, val)
	})
}

func (s *Storage) readUserByAddress(addr string) (*User, error) {
	txn := s.NewTransaction(false)
	defer txn.Discard()

	key := []byte(storePrefixAddress + addr)
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
	return s.readUser(txn, string(val))
}

func (s *Storage) readUserById(id string) (*User, error) {
	txn := s.NewTransaction(false)
	defer txn.Discard()

	return s.readUser(txn, id)
}

func (s *Storage) readUser(txn *badger.Txn, id string) (*User, error) {
	key := []byte(storePrefixUser + id)
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
	var user User
	err = common.MsgpackUnmarshal(val, &user)
	return &user, err
}

func (s *Storage) writeUser(u *User) error {
	return s.Update(func(txn *badger.Txn) error {
		key := []byte(storePrefixUser + u.UserID)
		val := common.MsgpackMarshalPanic(u)
		err := txn.Set(key, val)
		if err != nil {
			return err
		}
		key = []byte(storePrefixAddress + u.FullName)
		val = []byte(u.UserID)
		return txn.Set(key, val)
	})
}

func (s *Storage) writeSnapshot(snap *mixin.Snapshot) error {
	return s.Update(func(txn *badger.Txn) error {
		key := snapshotKey(snap)
		val := common.CompressMsgpackMarshalPanic(snap)
		return txn.Set(key, val)
	})
}

func (s *Storage) listSnapshots(limit int) ([]*mixin.Snapshot, error) {
	snapshots := make([]*mixin.Snapshot, 0)
	txn := s.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.Prefix = []byte(storePrefixSnapshotList)
	it := txn.NewIterator(opts)
	defer it.Close()

	it.Seek(opts.Prefix)
	for ; it.Valid() && len(snapshots) < limit; it.Next() {
		item := it.Item()
		v, err := item.ValueCopy(nil)
		if err != nil {
			return snapshots, err
		}
		var snap mixin.Snapshot
		err = common.DecompressMsgpackUnmarshal(v, &snap)
		if err != nil {
			return snapshots, err
		}
		snapshots = append(snapshots, &snap)
	}

	return snapshots, nil
}

func (s *Storage) deleteSnapshots(snaps []*mixin.Snapshot) error {
	return s.Update(func(txn *badger.Txn) error {
		for _, snap := range snaps {
			key := snapshotKey(snap)
			err := txn.Delete(key)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) listWithdrawals(limit int) ([]*Withdrawal, error) {
	withdrawals := make([]*Withdrawal, 0)
	txn := s.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.Prefix = []byte(storePrefixWithdrawalPair)
	it := txn.NewIterator(opts)
	defer it.Close()

	it.Seek(opts.Prefix)
	for ; it.Valid() && len(withdrawals) < limit; it.Next() {
		item := it.Item()
		v, err := item.ValueCopy(nil)
		if err != nil {
			return withdrawals, err
		}
		var w Withdrawal
		err = common.DecompressMsgpackUnmarshal(v, &w)
		if err != nil {
			return withdrawals, err
		}
		withdrawals = append(withdrawals, &w)
	}

	return withdrawals, nil
}

func (s *Storage) deleteWitdrawals(withdrawals []*Withdrawal) error {
	return s.Update(func(txn *badger.Txn) error {
		for _, w := range withdrawals {
			key := []byte(storePrefixWithdrawalPair + w.TraceId)
			err := txn.Delete(key)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) writeWithdrawal(w *Withdrawal) error {
	return s.Update(func(txn *badger.Txn) error {
		old, err := s.readWithdrawal(txn, w.TraceId)
		if err != nil {
			return err
		}
		if old != nil && w.Asset == nil {
			panic(old.Asset)
		}
		if old != nil && w.Fee == nil {
			panic(old.Fee)
		}
		if old != nil && old.CreatedAt != w.CreatedAt {
			panic(old.CreatedAt)
		}
		if old != nil && old.UserId != w.UserId {
			panic(old.UserId)
		}
		key := []byte(storePrefixWithdrawalPair + w.TraceId)
		val := common.CompressMsgpackMarshalPanic(w)
		return txn.Set(key, val)
	})
}

func (s *Storage) readWithdrawalById(id string) (*Withdrawal, error) {
	txn := s.NewTransaction(false)
	defer txn.Discard()

	return s.readWithdrawal(txn, id)
}

func (s *Storage) readWithdrawal(txn *badger.Txn, id string) (*Withdrawal, error) {
	key := []byte(storePrefixWithdrawalPair + id)
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
	var w Withdrawal
	err = common.DecompressMsgpackUnmarshal(val, &w)
	return &w, err
}

func snapshotKey(s *mixin.Snapshot) []byte {
	key := []byte(storePrefixSnapshotList)
	buf := timeToBytes(s.CreatedAt)
	key = append(key, buf...)
	key = append(key, s.SnapshotID...)
	return key
}

func timeToBytes(t time.Time) []byte {
	buf := make([]byte, 8)
	now := uint64(t.UnixNano())
	binary.BigEndian.PutUint64(buf, now)
	return buf
}

func (s *Storage) readCollectiblesCheckpoint(ctx context.Context) (time.Time, error) {
	key := []byte(storePrefixCollectibleOutputCheckpoint)
	return s.readCheckpoint(ctx, key)
}

func (s *Storage) writeCollectiblesCheckpoint(ctx context.Context, ckpt time.Time) error {
	key := []byte(storePrefixCollectibleOutputCheckpoint)
	return s.writeCheckpoint(ctx, key, ckpt)
}

func (s *Storage) writeCollectibleOutput(out *CollectibleOutput) error {
	return s.Update(func(txn *badger.Txn) error {
		key := collectibleOutputKey(out)
		val := common.CompressMsgpackMarshalPanic(out)
		return txn.Set(key, val)
	})
}

func (s *Storage) listCollectibleOutputs(limit int) ([]*CollectibleOutput, error) {
	outputs := make([]*CollectibleOutput, 0)
	txn := s.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.Prefix = []byte(storePrefixCollectibleOutputList)
	it := txn.NewIterator(opts)
	defer it.Close()

	it.Seek(opts.Prefix)
	for ; it.Valid() && len(outputs) < limit; it.Next() {
		item := it.Item()
		v, err := item.ValueCopy(nil)
		if err != nil {
			return outputs, err
		}
		var out CollectibleOutput
		err = common.DecompressMsgpackUnmarshal(v, &out)
		if err != nil {
			return outputs, err
		}
		outputs = append(outputs, &out)
	}

	return outputs, nil
}

func (s *Storage) deleteCollectibleOutputs(outs []*CollectibleOutput) error {
	return s.Update(func(txn *badger.Txn) error {
		for _, out := range outs {
			key := collectibleOutputKey(out)
			err := txn.Delete(key)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) writeCollectibleRawTransaction(raw string) error {
	b, _ := hex.DecodeString(raw)
	ver, _ := common.UnmarshalVersionedTransaction(b)
	return s.Update(func(txn *badger.Txn) error {
		key := storePrefixCollectibleRawTransaction + ver.PayloadHash().String()
		return txn.Set([]byte(key), []byte(raw))
	})
}

func (s *Storage) listCollectibleRawTransactions(limit int) (map[string]string, error) {
	raws := make(map[string]string)
	txn := s.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.Prefix = []byte(storePrefixCollectibleRawTransaction)
	it := txn.NewIterator(opts)
	defer it.Close()

	it.Seek(opts.Prefix)
	for ; it.Valid() && len(raws) < limit; it.Next() {
		item := it.Item()
		v, err := item.ValueCopy(nil)
		if err != nil {
			return raws, err
		}
		key := item.Key()[len(storePrefixCollectibleRawTransaction):]
		raws[string(key)] = string(v)
	}

	return raws, nil
}

func (s *Storage) deleteCollectibleRawTransaction(raw string) error {
	b, _ := hex.DecodeString(raw)
	ver, _ := common.UnmarshalVersionedTransaction(b)
	return s.Update(func(txn *badger.Txn) error {
		key := storePrefixCollectibleRawTransaction + ver.PayloadHash().String()
		return txn.Delete([]byte(key))
	})
}

func collectibleOutputKey(o *CollectibleOutput) []byte {
	key := []byte(storePrefixCollectibleOutputList)
	buf := timeToBytes(o.CreatedAt)
	key = append(key, buf...)
	key = append(key, o.OutputID...)
	return key
}

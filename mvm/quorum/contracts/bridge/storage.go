package main

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/dgraph-io/badger/v3"
	"github.com/fox-one/mixin-sdk-go"
)

const (
	storePrefixUser               = "USER:"
	storePrefixSnapshotList       = "SNAPSHOT:LIST:"
	storePrefixSnapshotCheckpoint = "SNAPSHOT:CHECKPOINT"
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

func (s *Storage) readSnapshotsCheckpoint(ctx context.Context) (time.Time, error) {
	txn := s.NewTransaction(false)
	defer txn.Discard()

	key := []byte(storePrefixSnapshotCheckpoint)
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return time.Time{}, nil
	} else if err != nil {
		return time.Time{}, err
	}
	val, err := item.ValueCopy(nil)
	if err != nil {
		return time.Time{}, err
	}
	var t time.Time
	err = common.MsgpackUnmarshal(val, &t)
	return t, err
}

func (s *Storage) writeSnapshotsCheckpoint(ctx context.Context, ckpt time.Time) error {
	return s.Update(func(txn *badger.Txn) error {
		key := []byte(storePrefixSnapshotCheckpoint)
		val := common.MsgpackMarshalPanic(ckpt)
		return txn.Set(key, val)
	})
}

func (s *Storage) readUser(id string) (*User, error) {
	txn := s.NewTransaction(false)
	defer txn.Discard()

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

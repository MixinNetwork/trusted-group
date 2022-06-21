package main

import (
	"context"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/fox-one/mixin-sdk-go"
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
	panic(0)
}

func (s *Storage) writeSnapshotsCheckpoint(ctx context.Context, ckpt time.Time) error {
	panic(0)
}

func (s *Storage) readUser(id string) (*User, error) {
	panic(0)
}

func (s *Storage) writeUser() {
	panic(0)
}

func (s *Storage) writeSnapshot(snap *mixin.Snapshot) error {
	panic(0)
}

func (s *Storage) listSnapshots(limit int) ([]*mixin.Snapshot, error) {
	panic(0)
}

func (s *Storage) deleteSnapshots(snaps []*mixin.Snapshot) error {
	panic(0)
}

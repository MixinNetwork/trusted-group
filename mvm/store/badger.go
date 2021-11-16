package store

import (
	"context"

	"github.com/MixinNetwork/nfo/store"
)

type BadgerStore struct {
	store.BadgerStore
}

func OpenBadger(ctx context.Context, path string) (*BadgerStore, error) {
	bs, err := store.OpenBadger(ctx, path)
	if err != nil {
		return nil, err
	}
	return &BadgerStore{*bs}, err
}

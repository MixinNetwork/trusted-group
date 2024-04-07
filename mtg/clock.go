package mtg

import (
	"context"
	"sync"
	"time"
)

const clockStorePropertyKey = "MTG:GROUP:CLOCK:MONOTONIC"

type Clock struct {
	sync.RWMutex
	store *SQLite3Store
	now   time.Time
}

func NewClock(ctx context.Context, store *SQLite3Store) (*Clock, error) {
	ts, err := readPropertyAsTime(ctx, store)
	if err != nil {
		return nil, err
	}
	if now := time.Now(); ts.Before(now) {
		ts = now
	}
	clock := new(Clock)
	clock.store = store
	clock.now = ts
	return clock, nil
}

func (c *Clock) Now(ctx context.Context) time.Time {
	c.Lock()
	defer c.Unlock()

	for {
		now := time.Now()
		if now.After(c.now) {
			c.now = now
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	for {
		err := c.store.WriteProperty(ctx, clockStorePropertyKey, c.now.Format(time.RFC3339Nano))
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return c.now
}

func readPropertyAsTime(ctx context.Context, store *SQLite3Store) (time.Time, error) {
	bs, err := store.ReadProperty(ctx, clockStorePropertyKey)
	if err != nil || bs == "" {
		return time.Now(), err
	}
	return time.Parse(time.RFC3339Nano, bs)
}

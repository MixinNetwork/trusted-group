package store

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

func (bs *BadgerStore) WriteGroupEventAndNonce(pid string, event *encoding.Event) error {
	panic(0)
}

func (bs *BadgerStore) ListSignedGroupEvents(pid string, limit int) ([]*encoding.Event, error) {
	panic(0)
}

func (bs *BadgerStore) ExpireGroupEventsWithCost(events []*encoding.Event, cost common.Integer) error {
	panic(0)
}

func (bs *BadgerStore) ListPendingGroupEvents(limit int) ([]*encoding.Event, error) {
	panic(0)
}

func (bs *BadgerStore) WriteGroupEventState(pid string, nonce uint64) error {
	panic(0)
}

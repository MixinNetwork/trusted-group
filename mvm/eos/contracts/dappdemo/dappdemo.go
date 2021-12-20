package main

import (
	"github.com/uuosio/chain"
)

const (
	KEY_NONCE            = 1
	KEY_TX_REQUEST_INDEX = 2
)

var (
	MTG_XIN       = chain.NewName("mtgxinmtgxin")
	MTG_PUBLISHER = chain.NewName("mtgpublisher")
	//uuid: 49b00892-6954-4826-aaec-371ca165558a
	PROCESS_ID = chain.Uint128([16]byte{0x49, 0xb0, 0x08, 0x92, 0x69, 0x54, 0x48, 0x26, 0xaa, 0xec, 0x37, 0x1c, 0xa1, 0x65, 0x55, 0x8a})
)

//table txevents
type TxEvent struct {
	nonce     uint64 //primary : t.nonce
	process   chain.Uint128
	asset     chain.Uint128
	members   []chain.Uint128
	threshold int32
	amount    chain.Uint128
	extra     []byte
	timestamp uint64
	signature []byte
}

//table txrequests
type TxRequest struct {
	nonce     uint64 //primary : t.nonce
	contract  chain.Name
	process   chain.Uint128
	asset     chain.Uint128
	members   []chain.Uint128
	threshold int32
	amount    chain.Uint128
	extra     []byte
	timestamp uint64
}

//table counters
type Counter struct {
	id    uint64 //primary : t.id
	count uint64
}

func check(b bool, msg string) {
	chain.Check(b, msg)
}

//contract dappdemo
type Contract struct {
	self, firstReceiver, action chain.Name
}

func NewContract(receiver, firstReceiver, action chain.Name) *Contract {
	return &Contract{receiver, firstReceiver, action}
}

//action onevent
func (c *Contract) OnEvent(event *TxEvent) {
	chain.RequireAuth(MTG_PUBLISHER)
	check(event.process == PROCESS_ID, "Invalid process id")

	nonce := c.GetNonce()
	check(event.nonce >= nonce, "bad nonce!")

	payer := c.self
	db := NewTxEventDB(c.self, c.self)
	it := db.Find(event.nonce)
	check(!it.IsOk(), "event already exists!")
	db.Store(event, payer)
}

//action exec
func (c *Contract) Exec(executor chain.Name) {
	chain.RequireAuth(executor)

	nonce := c.GetNonce()
	db := NewTxEventDB(c.self, c.self)
	it, event := db.Get(nonce)
	check(it.IsOk(), "event not found!")
	db.Remove(it)

	txRequestCount := 1
	for i := 0; i < txRequestCount; i++ {
		id := c.GetNextTxRequestNonce()
		notify := TxRequest{
			nonce:     id,
			contract:  c.self,
			process:   PROCESS_ID,
			asset:     event.asset,
			members:   event.members,
			threshold: event.threshold,
			amount:    event.amount,
			extra:     event.extra,
		}

		check(event.amount.Cmp(chain.NewUint128(chain.MAX_AMOUNT, 0)) < 0, "Invalid amount")

		amount := event.amount.Uint64() / uint64(txRequestCount)
		chain.Println("+++++++set amount:", amount)
		notify.amount.SetUint64(amount)

		chain.NewAction(
			chain.PermissionLevel{c.self, chain.ActiveName},
			MTG_XIN,
			chain.NewName("txrequest"),
			&notify,
		).Send()
	}
	c.IncNonce()
}

//action onerror
func (c *Contract) OnError(event *TxEvent, code uint64, reason string) {
	chain.RequireAuth(MTG_PUBLISHER)

	c.CheckAndIncNonce(event.nonce)

	id := c.GetNextTxRequestNonce()
	notify := TxRequest{
		nonce:     id,
		contract:  c.self,
		process:   PROCESS_ID,
		asset:     event.asset,
		members:   event.members,
		threshold: event.threshold,
		amount:    event.amount,
		extra:     event.extra,
	}

	chain.NewAction(
		chain.PermissionLevel{c.self, chain.ActiveName},
		MTG_XIN,
		chain.NewName("txrequest"),
		&notify,
	).Send()
}

func (c *Contract) GetNextIndex(key uint64, initialValue uint64) uint64 {
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(key); it.IsOk() {
		item.count += 1
		db.Update(it, item, chain.Name{N: 0})
		return item.count
	} else {
		item := Counter{id: key, count: initialValue}
		db.Store(&item, c.self)
		return item.count
	}
}

func (c *Contract) IncNonce() {
	key := uint64(KEY_NONCE)
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(key); it.IsOk() {
		item.count += 1
		db.Update(it, item, chain.SamePayer)
	} else {
		item := Counter{id: key, count: 1}
		db.Store(&item, c.self)
	}
}

func (c *Contract) GetNonce() uint64 {
	key := uint64(KEY_NONCE)
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(key); it.IsOk() {
		return item.count
	} else {
		return 1
	}
}

func (c *Contract) CheckAndIncNonce(oldNonce uint64) {
	key := uint64(KEY_NONCE)
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(key); it.IsOk() {
		chain.Println("++++CheckAndIncNonce:", item.count, oldNonce)
		check(item.count == oldNonce, "Invalid nonce")
		item.count = oldNonce + 1
		db.Update(it, item, chain.SamePayer)
	} else {
		item := Counter{id: key, count: oldNonce + 1}
		db.Store(&item, c.self)
	}
}

func (c *Contract) GetNextTxRequestNonce() uint64 {
	return c.GetNextIndex(KEY_TX_REQUEST_INDEX, 1)
}

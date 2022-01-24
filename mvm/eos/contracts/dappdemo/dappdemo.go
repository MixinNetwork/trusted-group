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
)

//table txevents
type TxEvent struct {
	nonce      uint64 //primary : t.nonce
	process    chain.Uint128
	asset      chain.Uint128
	members    []chain.Uint128
	threshold  int32
	amount     chain.Uint128
	extra      []byte
	timestamp  uint64
	signatures []chain.Signature
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

//contract dappdemo
type Contract struct {
	self          chain.Name
	firstReceiver chain.Name
	action        chain.Name
}

func NewContract(receiver, firstReceiver, action chain.Name) *Contract {
	c := &Contract{receiver, firstReceiver, action}
	return c
}

//action onevent ignore
func (c *Contract) OnEvent(event *TxEvent) {
	event = &TxEvent{}
	data := chain.ReadActionData()
	event.Unpack(data)
	dataSize := len(data) - 1 - len(event.signatures)*66

	VerifySignatures(data[:dataSize], event.signatures)

	VerifyProcess(c.self, event.process)

	assert(event.process == c.process, "Invalid process id")

	nonce := c.GetNonce()
	assert(event.nonce >= nonce, "bad nonce!")

	payer := c.self
	db := NewTxEventDB(c.self, c.self)
	it := db.Find(event.nonce)
	assert(!it.IsOk(), "event already exists!")
	db.Store(event, payer)
}

//action exec
func (c *Contract) Exec(executor chain.Name) {
	chain.RequireAuth(executor)

	nonce := c.GetNonce()
	db := NewTxEventDB(c.self, c.self)
	it, event := db.Get(nonce)
	assert(it.IsOk(), "event not found!")
	db.Remove(it)

	txRequestCount := 1
	for i := 0; i < txRequestCount; i++ {
		id := c.GetNextTxRequestNonce()
		notify := TxRequest{
			nonce:     id,
			contract:  c.self,
			process:   c.process,
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
		//nonce starts from 1, event with nonce 0 is for addprocess which sends to mtg.xin contract
		item := Counter{id: key, count: 2}
		db.Store(&item, c.self)
	}
}

func (c *Contract) GetNonce() uint64 {
	key := uint64(KEY_NONCE)
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(key); it.IsOk() {
		return item.count
	} else {
		//nonce starts from 1, event with nonce 0 is for addprocess which sends to mtg.xin contract
		item := Counter{id: key, count: 1}
		db.Store(&item, c.self)
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

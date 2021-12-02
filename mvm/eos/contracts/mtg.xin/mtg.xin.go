package main

import (
	"github.com/uuosio/chain"
)

const (
	KEY_TX_REQUEST_SEQ = 1
)

//contract mtg.xin
type Contract struct {
	self, firstReceiver, action chain.Name
}

func NewContract(receiver, firstReceiver, action chain.Name) *Contract {
	return &Contract{receiver, firstReceiver, action}
}

//action addprocess
func (c *Contract) AddProcess(contract chain.Name, process chain.Uint128) {
	check(chain.IsAccount(contract), "contract account does not exists!")
	chain.RequireAuth(c.self)
	db := NewProcessDB(c.self, c.self)
	it := db.Find(contract.N)
	check(!it.IsOk(), "process already exists!")
	item := &Process{
		contract: contract,
		process:  process,
	}
	db.Store(item, c.self)
}

//action txrequest
func (c *Contract) TxRequest(nonce uint64,
	contract chain.Name,
	process chain.Uint128,
	asset chain.Uint128,
	members []chain.Uint128,
	threshold int32,
	amount chain.Uint128,
	extra []byte) {

	chain.RequireAuth(contract)
	db := NewProcessDB(c.self, c.self)
	it, item := db.Get(contract.N)
	check(it.IsOk(), "process not found!")
	check(item.process == process, "invalid process!")

	seq := c.GetNextSeq()
	log := TxLog{
		id:        seq,
		nonce:     nonce,
		contract:  contract,
		process:   process,
		asset:     asset,
		members:   members,
		threshold: threshold,
		amount:    amount,
		extra:     extra,
		timestamp: chain.CurrentTime().Elapsed * 1000,
	}

	chain.NewAction(
		chain.PermissionLevel{c.self, chain.ActiveName},
		c.self,
		chain.NewName("ontxlog"),
		&log,
	).Send()
	//TODO: emit transfer event so block explorer can show it
}

//node call push event error, refund
//action refundreq
func (c *Contract) RefundRequest(nonce uint64,
	contract chain.Name,
	process chain.Uint128,
	asset chain.Uint128,
	members []chain.Uint128,
	threshold int32,
	amount chain.Uint128,
	extra []byte) {

	chain.RequireAuth(contract)
	db := NewProcessDB(c.self, c.self)
	it, item := db.Get(contract.N)
	check(it.IsOk(), "process not found!")
	check(item.process == process, "invalid process!")

	seq := c.GetNextSeq()
	log := TxLog{
		id:        seq,
		nonce:     nonce,
		contract:  contract,
		process:   process,
		asset:     asset,
		members:   members,
		threshold: threshold,
		amount:    amount,
		extra:     extra,
		timestamp: chain.CurrentTime().Elapsed * 1000,
	}

	chain.NewAction(
		chain.PermissionLevel{c.self, chain.ActiveName},
		c.self,
		chain.NewName("ontxlog"),
		&log,
	).Send()
	//TODO: emit transfer event so block explorer can show it
}

//action ontxlog ignore
func (c *Contract) OnTxLog(log *TxLog) {
	chain.RequireAuth(c.self)
}

func (c *Contract) GetNextIndex(key uint64) uint64 {
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(key); it.IsOk() {
		index := item.Count
		item.Count += 1
		db.Update(it, item, chain.SamePayer)
		return index
	} else {
		item := Counter{Id: key, Count: 1}
		db.Store(&item, c.self)
		return 0
	}
}

func (c *Contract) GetNextSeq() uint64 {
	return c.GetNextIndex(KEY_TX_REQUEST_SEQ)
}

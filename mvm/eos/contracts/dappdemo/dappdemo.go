package main

import (
	"github.com/uuosio/chain"
)

//contract dappdemo
type Contract struct {
	self, firstReceiver, action chain.Name
}

func NewContract(receiver, firstReceiver, action chain.Name) *Contract {
	return &Contract{receiver, firstReceiver, action}
}

//needs MTG multisig permission
//action onevent
func (c *Contract) OnEvent(event *TxEvent) {
	chain.RequireAuth(MTG_CONTRACT)
	c.CheckAndIncNonce(event.nonce)
	// nonce := c.GetNextNonce()
	// check(nonce == event.nonce, "Invalid nonce")
	payer := c.self
	check(event.process == PROCESS_ID, "Invalid process id")
	chain.Println("+++OnEvent")
	if false {
		db := NewTxEventDB(c.self, c.self)
		it := db.Find(event.nonce)
		check(!it.IsOk(), "event already exists!")
		db.Store(event, payer)
	}

	txRequestCount := 3
	for i := 0; i < txRequestCount; i++ {
		id := c.GetNextTxRequestIndex()
		notify := TxRequest{
			nonce:     id,
			contract:  c.self,
			process:   event.process,
			asset:     event.asset,
			members:   event.members,
			threshold: event.threshold,
			amount:    event.amount,
			extra:     event.extra,
		}
		amount := event.amount.Uint64() / uint64(txRequestCount)
		chain.Println("+++++++set amount:", amount)
		notify.amount.SetUint64(amount)

		//two methods of query event log
		//1. query event log by action history
		//2. query event log from on-chain database(needs to clean finished requests)

		//current used method: 1

		//send event to a specified account for log query by action trace history
		chain.NewAction(
			chain.PermissionLevel{c.self, chain.ActiveName},
			MTG_CONTRACT,
			chain.NewName("txrequest"),
			&notify,
		).Send()
	}

	//remove as mush as finished requests
	// lastFinishedRequest := c.GetLastFinishedRequestIndex()
	// if lastFinishedRequest != 0 {
	// 	c.ClearFinishedRequests(lastFinishedRequest)
	// }
}

//action clearreqs
func (c *Contract) ClearFinishedRequests(lastFinishedRequest uint64) {
	chain.RequireAuth(c.self)
	db := NewTxRequestDB(c.self, c.self)
	count := 0

	for {
		it := db.Lowerbound(uint64(0))
		if !it.IsOk() {
			break
		}
		data, err := db.GetByIterator(it)
		if err != nil {
			break
		}
		if data.nonce <= lastFinishedRequest {
			db.Remove(it)
		}
		count += 1
		if count >= MAX_REMOVE_RECORD_COUNT {
			c.SetLastFinishedRequestIndex(lastFinishedRequest)
		}
	}
}

//action clearnonce
func (c *Contract) ClearNonce() {
	key := uint64(KEY_NONCE)
	db := NewCounterDB(c.self, c.self)
	if it := db.Find(key); it.IsOk() {
		chain.Println("++++it:", it.I)
		db.Remove(it)
	}
}

//action sayhello
func (c *Contract) SayHello(name string) {
	chain.Println("Hello, ", name)
}

func (c *Contract) GetNextIndex(key uint64, initialValue uint64) uint64 {
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(key); it.IsOk() {
		item.Count += 1
		db.Update(it, item, chain.Name{N: 0})
		return item.Count
	} else {
		item := Counter{Id: key, Count: initialValue}
		db.Store(&item, c.self)
		return item.Count
	}
}

func (c *Contract) SetCounterValue(key uint64, value uint64) {
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(key); it.IsOk() {
		item.Count = value
		db.Update(it, item, chain.SamePayer)
	} else {
		item := Counter{Id: key, Count: value}
		db.Store(&item, c.self)
	}
}

func (c *Contract) SetLastFinishedRequestIndex(index uint64) {
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(KEY_FINISHED_REQUEST); it.IsOk() {
		item.Count = index
		db.Update(it, item, chain.SamePayer)
	} else {
		item := Counter{Id: KEY_FINISHED_REQUEST, Count: index}
		db.Store(&item, c.self)
	}
}

func (c *Contract) GetLastFinishedRequestIndex() uint64 {
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(KEY_FINISHED_REQUEST); it.IsOk() {
		return item.Count
	}
	return 0
}

func (c *Contract) RemoveLastFinishedRequestIndex() {
	db := NewCounterDB(c.self, c.self)
	it := db.Find(KEY_FINISHED_REQUEST)
	if it.IsOk() {
		db.Remove(it)
	}
}

func (c *Contract) GetNextNonce() uint64 {
	return c.GetNextIndex(KEY_NONCE, 0)
}

func (c *Contract) CheckAndIncNonce(oldNonce uint64) {
	key := uint64(KEY_NONCE)
	db := NewCounterDB(c.self, c.self)
	if it, item := db.Get(key); it.IsOk() {
		chain.Println("++++CheckAndIncNonce:", item.Count, oldNonce)
		//		check(item.Count == oldNonce, "Invalid nonce")
		item.Count = oldNonce + 1
		db.Update(it, item, chain.SamePayer)
	} else {
		item := Counter{Id: key, Count: oldNonce + 1}
		db.Store(&item, c.self)
	}
}

func (c *Contract) GetNextTxRequestIndex() uint64 {
	return c.GetNextIndex(KEY_TX_REQUEST_INDEX, 1)
}

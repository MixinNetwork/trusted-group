package main

import (
	"github.com/uuosio/chain"
)

const (
	KEY_NONCE        = 1
	KEY_TX_OUT_INDEX = 2
	KEY_TX_IN_INDEX  = 3
)

//asset id:
//c6d0c728-2624-429b-8e0d-d9d19b6592fa (bitcoin)
//31d2ea9c-95eb-3355-b65b-ba096853bc18 (Pando USD)
//4d8c508b-91c5-375b-92b0-ee702ed2dac5 (USDT)
//c94ac88f-4671-3976-b60a-09064f1811e8 (MIXIN)
//965e5c6e-434c-3fa9-b780-c50f43cd955c (CNB)
//43d61dcd-e413-450d-80b8-101d5e903357 (ETH)
//6cfe566e-4aad-470b-8c9a-2fd35b49c68d (EOS)

var (
	MTG_XIN       = chain.NewName("mtgxinmtgxin")
	MTG_PUBLISHER = chain.NewName("mtgpublisher")
	//uuid: e0148fc6-0e10-470e-8127-166e0829c839
	PROCESS_ID = chain.Uint128([16]byte{0xe0, 0x14, 0x8f, 0xc6, 0x0e, 0x10, 0x47, 0x0e, 0x81, 0x27, 0x16, 0x6e, 0x08, 0x29, 0xc8, 0x39})

	ASSET_ID_EOS = chain.Uint128([16]byte{0x6c, 0xfe, 0x56, 0x6e, 0x4a, 0xad, 0x47, 0x0b, 0x8c, 0x9a, 0x2f, 0xd3, 0x5b, 0x49, 0xc6, 0x8d})
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

//contract mixincross
type Contract struct {
	self, firstReceiver, action chain.Name
}

func NewContract(receiver, firstReceiver, action chain.Name) *Contract {
	return &Contract{receiver, firstReceiver, action}
}

//action onevent ignore
func (c *Contract) OnEvent(event *TxEvent) {
	event = &TxEvent{}
	data := chain.ReadActionData()
	event.Unpack(data)
	dataSize := len(data) - 1 - len(event.signatures)*66

	VerifySignatures(data[:dataSize], event.signatures)

	check(event.process == PROCESS_ID, "Invalid process id")

	nonce := c.GetNonce()
	check(event.nonce >= nonce, "bad nonce!")

	payer := c.self
	db := NewTxEventDB(c.self, c.self)
	it := db.Find(event.nonce)
	check(!it.IsOk(), "event already exists!")
	db.Store(event, payer)
	chain.Println("Done!")
}

//action exec
func (c *Contract) Exec(executor chain.Name) {
	chain.RequireAuth(executor)

	nonce := c.GetNonce()
	db := NewTxEventDB(c.self, c.self)
	it, event := db.Get(nonce)
	check(it.IsOk(), "event not found!")
	db.Remove(it)

	if event.amount.Cmp(chain.NewUint128(chain.MAX_AMOUNT, 0)) > 0 {
		c.Refund(event, "refund")
		return
	}

	c.HandleCrossTransfer(event)
	c.IncNonce()
}

func (c *Contract) HandleCrossTransfer(event *TxEvent) {
	toAccount := string(event.extra)
	to := chain.NewName(toAccount)
	if !chain.IsAccount(to) {
		c.Refund(event, "to account not exists, refund")
		return
	}

	if event.asset == ASSET_ID_EOS {
		sym := chain.NewSymbol("EOS", 4)
		quantity := chain.NewAsset(int64(event.amount.Uint64())/10000, sym)
		totalBalance := GetBalance(c.self, chain.TokenContractName, chain.NewSymbol("EOS", 4))
		if totalBalance.Amount < quantity.Amount {
			c.Refund(event, "balance not enough, refund")
			return
		}
		c.TransferTo(to, quantity, string(event.extra))
	} else {
		c.Refund(event, "unsupported asset, refund")
	}
}

//action transferin
func (c *Contract) TransferTo(to chain.Name, quantity *chain.Asset, memo string) {
	id := c.GetNextTxInIndex()
	a := chain.NewAction(
		chain.PermissionLevel{c.self, chain.ActiveName},
		chain.TokenContractName,
		chain.NewName("transfer"),
		c.self,   //from
		to,       // to,
		quantity, //quantity
		memo,
	)
	tx := chain.NewTransaction(0)
	tx.Actions = []*chain.Action{a}
	payer := c.self
	tx.Send(id, true, payer)
}

func (c *Contract) TransferOut(member *chain.Uint128, amount chain.Asset, memo string) {
	//TODO: make sure balance in MTG is sufficient.
	check(amount.Symbol == chain.NewSymbol("EOS", 4), "unsupported asset")
	_amount := chain.NewUint128(uint64(amount.Amount), 0)
	_amount.Mul(_amount, chain.NewUint128(10000, 0))
	id := c.GetNextTxRequestNonce()
	notify := TxRequest{
		nonce:     id,
		contract:  c.self,
		process:   PROCESS_ID,
		asset:     ASSET_ID_EOS,
		members:   []chain.Uint128{*member},
		threshold: 1,
		amount:    *_amount,
		extra:     []byte(memo),
	}

	chain.NewAction(
		chain.PermissionLevel{c.self, chain.ActiveName},
		MTG_XIN,
		chain.NewName("txrequest"),
		&notify,
	).Send()
}

//notify transfer
func (c *Contract) Transfer(from chain.Name, to chain.Name, quantity chain.Asset, memo string) {
	if to != c.self {
		return
	}

	check(c.firstReceiver == chain.TokenContractName, "bad token contract!")
	check(quantity.Symbol == chain.NewSymbol("EOS", 4), "unsupported asset")
	//TODO: check free amount of MTG
	cliendId, ok := GetClientId(memo)
	if !ok {
		return
	}
	c.TransferOut(cliendId, quantity, "xtransfer")
}

func (c *Contract) Refund(event *TxEvent, memo string) {
	id := c.GetNextTxRequestNonce()
	notify := TxRequest{
		nonce:     id,
		contract:  c.self,
		process:   PROCESS_ID,
		asset:     event.asset,
		members:   event.members,
		threshold: event.threshold,
		amount:    event.amount,
		extra:     []byte(memo),
	}

	chain.NewAction(
		chain.PermissionLevel{c.self, chain.ActiveName},
		MTG_XIN,
		chain.NewName("txrequest"),
		&notify,
	).Send()
}

//action test
func (c *Contract) Test() {
	a := chain.NewAction(
		chain.PermissionLevel{c.self, chain.ActiveName},
		chain.NewName("eosio.token"),
		chain.NewName("transfer"),
		c.self,
		chain.NewName("notexita"),
		chain.NewAsset(10000, chain.NewSymbol("EOS", 4)),
		"hello,world",
	)

	tx := chain.NewTransaction(1)
	tx.Actions = []*chain.Action{a}
	payer := c.self
	tx.Send(1, false, payer)
	chain.Println("transaction sent")
}

//void onerror( ignore<uint128_t> sender_id, ignore<std::vector<char>> sent_trx );

//action onerror ignore
func (c *Contract) OnError(sender_id *chain.Uint128, sent_trx []byte) {
	chain.Println("+++++++on error:", *sender_id)

	tx := chain.Transaction{}
	tx.Unpack(sent_trx)
	if len(tx.Actions) == 0 {
		return
	}
	action := tx.Actions[0]
	if action.Account == chain.NewName("eosio.token") {
	}
	chain.Println("+++++++on error:", *sender_id, action.Account)
}

//action clear
func (c *Contract) clear() {
	chain.RequireAuth(c.self)
	{
		db := NewCounterDB(c.self, c.self)
		for {
			it := db.Lowerbound(0)
			if !it.IsOk() {
				break
			}
			db.Remove(it)
		}
	}
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
	return c.GetNextIndex(KEY_TX_OUT_INDEX, 1)
}

func (c *Contract) GetNextTxInIndex() uint64 {
	return c.GetNextIndex(KEY_TX_IN_INDEX, 1)
}

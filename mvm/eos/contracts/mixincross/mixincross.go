package main

import (
	"github.com/uuosio/chain"
	"github.com/uuosio/chain/database"
)

const (
	KEY_NONCE        = 1
	KEY_TX_OUT_INDEX = 2
	KEY_TX_IN_INDEX  = 3
	KEY_ASSET_INDEX  = 4

	MTG_WORK_EXPIRATION_SECONDS = 60
	MAX_SUPPLY                  = 100000000000000
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
	MIXIN_WTOKENS = chain.NewName("mixinwtokens")
	//uuid: e0148fc6-0e10-470e-8127-166e0829c839
	PROCESS_ID = chain.Uint128([16]byte{0xe0, 0x14, 0x8f, 0xc6, 0x0e, 0x10, 0x47, 0x0e, 0x81, 0x27, 0x16, 0x6e, 0x08, 0x29, 0xc8, 0x39})

	ASSET_ID_EOS = chain.Uint128([16]byte{0x6c, 0xfe, 0x56, 0x6e, 0x4a, 0xad, 0x47, 0x0b, 0x8c, 0x9a, 0x2f, 0xd3, 0x5b, 0x49, 0xc6, 0x8d})

	ASSET_ID_BTC  = chain.Uint128([16]byte{0xc6, 0xd0, 0xc7, 0x28, 0x26, 0x24, 0x42, 0x9b, 0x8e, 0x0d, 0xd9, 0xd1, 0x9b, 0x65, 0x92, 0xfa})
	ASSET_ID_PUSD = chain.Uint128([16]byte{0x31, 0xd2, 0xea, 0x9c, 0x95, 0xeb, 0x33, 0x55, 0xb6, 0x5b, 0xba, 0x09, 0x68, 0x53, 0xbc, 0x18})
	ASSET_ID_USDT = chain.Uint128([16]byte{0x4d, 0x8c, 0x50, 0x8b, 0x91, 0xc5, 0x37, 0x5b, 0x92, 0xb0, 0xee, 0x70, 0x2e, 0xd2, 0xda, 0xc5})
	ASSET_ID_XIN  = chain.Uint128([16]byte{0xc9, 0x4a, 0xc8, 0x8f, 0x46, 0x71, 0x39, 0x76, 0xb6, 0x0a, 0x09, 0x06, 0x4f, 0x18, 0x11, 0xe8})
	ASSET_ID_ETH  = chain.Uint128([16]byte{0x43, 0xd6, 0x1d, 0xcd, 0xe4, 0x13, 0x45, 0x0d, 0x80, 0xb8, 0x10, 0x1d, 0x5e, 0x90, 0x33, 0x57})
	ASSET_ID_CNB  = chain.Uint128([16]byte{0x96, 0x5e, 0x5c, 0x6e, 0x43, 0x4c, 0x3f, 0xa9, 0xb7, 0x80, 0xc5, 0x0f, 0x43, 0xcd, 0x95, 0x5c})
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

//table eosbalances singleton
type EOSBalance struct {
	amount chain.Asset
}

//table works
type MTGWork struct {
	id         uint64 //primary : t.id
	expiration uint32 //IDX64 : ByExpiration : uint64(t.expiration) : t.expiration = uint32(%v)
	from       chain.Uint128
	to         chain.Name
	quantity   chain.Asset
	memo       string
}

//contract mixincross
type Contract struct {
	self          chain.Name
	firstReceiver chain.Name
	action        chain.Name
	event         *TxEvent
}

func NewContract(receiver, firstReceiver, action chain.Name) *Contract {
	c := &Contract{receiver, firstReceiver, action, nil}
	// sys.Init(c)
	return c
}

func (c *Contract) OnRevert(msg string) {
	if c.event != nil {
		c.Refund(c.event, msg)
		c.event = nil
	}
}

//action onevent ignore
func (c *Contract) OnEvent(event *TxEvent) {
	event = &TxEvent{}
	data := chain.ReadActionData()
	event.Unpack(data)
	dataSize := len(data) - 1 - len(event.signatures)*66

	VerifySignatures(data[:dataSize], event.signatures)

	assert(event.process == PROCESS_ID, "Invalid process id")

	nonce := c.GetNonce()
	assert(event.nonce >= nonce, "bad nonce!")

	if event.amount.Cmp(chain.NewUint128(chain.MAX_AMOUNT, 0)) > 0 {
		c.Refund(event, "amount too large, refund")
		return
	}

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

	c.event = event
	c.HandleCrossTransfer(event)
	c.IncNonce()
}

//action dowork
func (c *Contract) DoWork(executor chain.Name, id uint64) {
	db := NewMTGWorkDB(c.self, c.self)
	//check expired work first
	idxDB := db.GetIdxDBByExpiration()
	itExpiration, _ := idxDB.Lowerbound(uint64(0))
	var it database.Iterator
	var transfer *MTGWork

	if itExpiration.IsOk() {
		it, transfer = db.Get(itExpiration.Primary)
		assert(it.IsOk(), "MTGWork not found!")
		if transfer.expiration < chain.CurrentTimeSeconds() {
			clientId := transfer.from
			assetId, ok := GetAssetId(transfer.quantity.Symbol)
			assert(ok, "unsupported asset id")
			amount := chain.NewUint128(uint64(transfer.quantity.Amount), 0)
			if transfer.quantity.Symbol == chain.NewSymbol("EOS", 4) {
				amount = amount.Mul(amount, chain.NewUint128(10000, 0))
			}
			c.HandleRefund(clientId, assetId, *amount, "expired, refund")
			db.Remove(it)
			chain.Exit()
		}
	}
	if transfer.id != id {
		it := db.Lowerbound(id)
		assert(it.IsOk(), "MTGWork not found!")
		transfer, _ = db.GetByIterator(it)
	}
	c.HandleTransferIn(transfer)
	db.Remove(it)
}

func (c *Contract) HandleTransferIn(transfer *MTGWork) {
	if transfer.quantity.Symbol == chain.NewSymbol("EOS", 4) {
		chain.NewAction(
			chain.PermissionLevel{c.self, chain.ActiveName},
			chain.TokenContractName,
			chain.NewName("transfer"),
			c.self,            //from
			transfer.to,       // to,
			transfer.quantity, //quantity
			transfer.memo,
		).Send()
	} else {
		symbol := transfer.quantity.Symbol
		sym_code := symbol.Code()
		db := NewCurrencyStatsDB(MIXIN_WTOKENS, chain.Name{sym_code})
		itr := db.Find(sym_code)
		if !itr.IsOk() {
			maxSupply := chain.NewAsset(MAX_SUPPLY, symbol)
			chain.NewAction(
				chain.PermissionLevel{MIXIN_WTOKENS, chain.ActiveName},
				MIXIN_WTOKENS,
				chain.NewName("create"),
				c.self,
				maxSupply,
				"create",
			).Send()
		}
		chain.NewAction(
			chain.PermissionLevel{c.self, chain.ActiveName},
			MIXIN_WTOKENS,
			chain.NewName("issue"),
			c.self,
			transfer.quantity,
			"issue",
		).Send()

		chain.NewAction(
			chain.PermissionLevel{c.self, chain.ActiveName},
			MIXIN_WTOKENS,
			chain.NewName("transfer"),
			c.self,
			transfer.to,
			transfer.quantity,
			"transfer",
		).Send()
	}
}

//action revert
func (c *Contract) Revert(errMsg string) {
}

func (c *Contract) HandleCrossTransfer(event *TxEvent) {
	toAccount := string(event.extra)
	to := chain.NewName(toAccount)
	check(chain.IsAccount(to), "account does not exists, refund")
	check(len(event.members) == 1, "multisig event not supported currently")
	from := event.members[0]

	if event.asset == ASSET_ID_EOS {
		sym := chain.NewSymbol("EOS", 4)
		quantity := chain.NewAsset(int64(event.amount.Uint64())/10000, sym)
		totalBalance := GetBalance(c.self, chain.TokenContractName, chain.NewSymbol("EOS", 4))
		check(totalBalance.Amount >= quantity.Amount, "balance not enough, refund")
		c.TransferTo(from, to, quantity, string(event.extra), event.timestamp)
		c.AddEOSBalance(quantity)
	} else {
		symbol := GetSymbol(event.asset)
		asset := chain.NewAsset(int64(event.amount.Uint64()), symbol)
		c.TransferTo(from, to, asset, string(event.extra), event.timestamp)
	}
}

func (c *Contract) TransferTo(from chain.Uint128, to chain.Name, quantity *chain.Asset, memo string, timestamp uint64) {
	id := c.GetNextTxInIndex()
	db := NewMTGWorkDB(c.self, c.self)
	x := &MTGWork{id, uint32(timestamp/1e9) + MTG_WORK_EXPIRATION_SECONDS, from, to, *quantity, memo}
	db.Store(x, c.self)
}

func (c *Contract) TransferOut(member *chain.Uint128, amount chain.Asset, memo string) {
	assetId, ok := GetAssetId(amount.Symbol)
	assert(ok, "unsupported asset id")
	//TODO: make sure balance in MTG is sufficient.
	_amount := chain.NewUint128(uint64(amount.Amount), 0)
	if amount.Symbol == chain.NewSymbol("EOS", 4) {
		_amount.Mul(_amount, chain.NewUint128(10000, 0))
		c.SubEOSBalance(&amount)
	}

	if c.firstReceiver == MIXIN_WTOKENS {
		chain.NewAction(
			chain.PermissionLevel{c.self, chain.ActiveName},
			MIXIN_WTOKENS,
			chain.NewName("retire"),
			&amount,
			"retire",
		).Send()
	}

	id := c.GetNextTxRequestNonce()
	notify := TxRequest{
		nonce:     id,
		contract:  c.self,
		process:   PROCESS_ID,
		asset:     assetId,
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

	if c.firstReceiver != chain.TokenContractName && c.firstReceiver != MIXIN_WTOKENS {
		return
	}

	cliendId, ok := GetClientId(memo)
	if !ok {
		return
	}
	//TODO: check free amount of MTG
	c.TransferOut(cliendId, quantity, "MTGWork")
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

func (c *Contract) HandleRefund(clientId chain.Uint128, assetId chain.Uint128, amount chain.Uint128, memo string) {
	id := c.GetNextTxRequestNonce()
	notify := TxRequest{
		nonce:     id,
		contract:  c.self,
		process:   PROCESS_ID,
		asset:     assetId,
		members:   []chain.Uint128{clientId},
		threshold: 1,
		amount:    amount,
		extra:     []byte(memo),
	}

	chain.NewAction(
		chain.PermissionLevel{c.self, chain.ActiveName},
		MTG_XIN,
		chain.NewName("txrequest"),
		&notify,
	).Send()
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

func (c *Contract) AddEOSBalance(amount *chain.Asset) {
	payer := c.self
	db := NewEOSBalanceDB(c.self, c.self)
	data := db.Get()
	if data == nil {
		db.Set(&EOSBalance{amount: *amount}, payer)
	} else {
		data.amount.Add(amount)
		db.Set(data, payer)
	}
}

func (c *Contract) SubEOSBalance(amount *chain.Asset) {
	db := NewEOSBalanceDB(c.self, c.self)
	data := db.Get()
	check(data != nil, "Balance not enough")
	data.amount.Sub(amount)
	check(data.amount.Amount > 0, "Balance not enough")
	db.Set(data, c.self)
}

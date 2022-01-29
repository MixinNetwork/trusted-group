package main

import "github.com/uuosio/chain"

//action initialize
func (c *Contract) Initialize() {
	chain.RequireAuth(c.self)
	db := NewAccountCacheDB(c.self, c.self)
	item := db.Get()
	check(item == nil, "Account cache already initialized")

	item = &AccountCache{}
	item.id = 0
	item.account = GetAccountNameFromId(item.id)

	CreateNewAccount(c.self, ACCOUNT_OWNER, item.account)
	db.Set(item, c.self)
}

//action addasset
func (c *Contract) AddMixinAsset(asset_id chain.Uint128, symbol chain.Symbol) {
	chain.RequireAuth(c.self)
	db := NewMixinAssetDB(c.self, c.self)
	it := db.Find(symbol.Code())
	check(!it.IsOk(), "Asset already exists")
	db.Store(&MixinAsset{symbol, asset_id}, c.self)
}

//action removeasset
func (c *Contract) RemoveMixinAsset(symbol chain.Symbol) {
	chain.RequireAuth(c.self)
	db := NewMixinAssetDB(c.self, c.self)
	it := db.Find(symbol.Code())
	check(it.IsOk(), "Asset does not exists")
	db.Remove(it)
}

//action onevent ignore
func (c *Contract) OnEvent(event *TxEvent) {
	event = &TxEvent{}
	data := chain.ReadActionData()
	event.Unpack(data)
	dataSize := len(data) - 1 - len(event.signatures)*66

	VerifySignatures(data[:dataSize], event.signatures)

	assert(event.process == c.process, "invalid process!")

	c.CheckNonce(event.nonce)

	c.HandleNormalEvent(event)
}

//action onerrorevent ignore
func (c *Contract) OnErrorEvent(event *TxEvent, reason *string) {
	errorEvent := &ErrorTxEvent{}
	data := chain.ReadActionData()

	dec := chain.NewDecoder(data)
	size := dec.Unpack(&errorEvent.event)

	event = &errorEvent.event
	dataSize := size - 1 - len(event.signatures)*66

	VerifySignatures(data[:dataSize], event.signatures)

	assert(event.process == c.process, "Invalid process id")

	errorEvent.reason = dec.UnpackString()

	nonce := c.GetNonce()
	assert(event.nonce >= nonce, "bad nonce!")

	c.StoreNonce(event.nonce)

	db := NewErrorTxEventDB(c.self, c.self)
	it := db.Find(event.nonce)
	assert(!it.IsOk(), "event already exists!")
	db.Store(errorEvent, c.self)
}

//action exec
func (c *Contract) Exec(executor chain.Name) {
	chain.RequireAuth(executor)
	//	db := NewTxEventDB(c.self, c.self)
	db := NewErrorTxEventDB(c.self, c.self)
	it := db.Lowerbound(uint64(0))
	assert(it.IsOk(), "error event not found!")
	errorEvent := db.GetByIterator(it)
	c.HandleErrorEvent(&errorEvent.event)
	db.Remove(it)
}

//action dowork
func (c *Contract) DoWork(executor chain.Name, id uint64) {
	assert(false, "Not implemented")
}

//action setfee
func (c *Contract) SetTransferFee(fee *chain.Asset) {
	chain.RequireAuth(c.self)
	assert(fee.Amount > 0, "fee must be greater than 0")
	{
		db := NewMixinAssetDB(c.self, c.self)
		it := db.Find(fee.Symbol.Code())
		assert(it.IsOk(), "asset not found!")
	}
	db := NewTransferFeeDB(c.self, c.self)
	it, transfeFee := db.Get(fee.Symbol.Code())
	if it.IsOk() {
		transfeFee.fee = *fee
		db.Update(it, transfeFee, chain.SamePayer)
	} else {
		db.Store(&TransferFee{*fee}, c.self)
	}
}

//action setaccfee
func (c *Contract) SetCreateAccountFee(fee *chain.Asset) {
	chain.RequireAuth(c.self)
	assert(fee.Amount > 0, "fee must be greater than 0")
	db := NewCreateAccountFeeDB(c.self, c.self)
	db.Set(&CreateAccountFee{*fee}, c.self)
}

//notify transfer
func (c *Contract) Transfer(from chain.Name, to chain.Name, quantity chain.Asset, memo string) {
	if to != c.self {
		return
	}

	if c.firstReceiver != MIXIN_WTOKENS {
		return
	}

	cliendId, ok := GetClientId(memo)
	if !ok {
		return
	}
	c.TransferOut(cliendId, quantity, "MTGWork")
}

//action ontransfer
func (c *Contract) OnTransfer(from chain.Name, to chain.Name, quantity chain.Asset, memo string) {
	//this is a deposit transfer
	if from == c.self {
		return
	}

	db := NewMixinAccountDB(c.self, c.self)
	it, record := db.Get(to.N)
	//to account is not a mixinaccount, no tx request
	if !it.IsOk() {
		return
	}

	amount := chain.NewUint128(uint64(quantity.Amount), 0)

	assetDB := NewMixinAssetDB(c.self, c.self)
	it, asset := assetDB.Get(quantity.Symbol.Code())
	check(it.IsOk(), "invalid mixin asset")
	assetId := asset.asset_id
	id := c.GetNextTxRequestNonce()
	notify := TxRequest{
		nonce:     id,
		contract:  c.self,
		process:   c.process,
		asset:     assetId,
		members:   []chain.Uint128{record.client_id},
		threshold: 1,
		amount:    *amount,
		extra:     []byte(memo),
	}
	chain.NewAction(
		&chain.PermissionLevel{c.self, chain.ActiveName},
		MTG_XIN,
		chain.NewName("txrequest"),
		&notify,
	).Send()
}

//action error
func (c *Contract) Error(err string) {
}

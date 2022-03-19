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
func (c *Contract) OnEvent(event *TxEvent, origin_extra []byte) {
	event = &TxEvent{}
	data := chain.ReadActionData()
	dec := chain.NewDecoder(data)
	dec.UnpackI(event)
	dataSize := dec.Pos() - 1 - len(event.signatures)*66
	origin_extra = dec.UnpackBytes()
	if len(origin_extra) == 0 {
		origin_extra = nil
	}
	VerifySignatures(data[:dataSize], event.signatures)

	assert(event.process == c.process, "invalid process!")

	c.CheckNonce(event.nonce)

	c.HandleEvent(event, origin_extra)
}

func (c *Contract) CreateAccount(event *TxEvent) (chain.Name, bool) {
	symbol, ok := c.GetSymbol(event.asset)
	if !ok {
		return chain.Name{}, false
	}
	quantity := chain.NewAsset(int64(event.amount.Uint64()), symbol)

	clientId := event.members[0]
	account, ok := c.GetAccount(event.members[0])
	if ok {
		return account, true
	}
	fee := c.GetCreateAccountFee()

	if quantity.Symbol != chain.NewSymbol("MEOS", 8) {
		c.ShowError("invalid asset for creating account")
		return chain.Name{}, false
	}

	if quantity.Amount < fee.Amount {
		c.ShowError("not enough fee for creating account")
		return chain.Name{}, false
	}
	quantity.Amount -= fee.Amount

	return c.CreateNewAccount(clientId), true
}

func (c *Contract) StorePendingEvent(account chain.Name, event *TxEvent) bool {
	//large extra, handled in execpending action
	if len(event.extra) <= 0 {
		return false
	}

	if event.extra[0] == EVENT_PENDING {
		if len(event.extra) < 1+32 {
			c.ShowError("invalid extra")
			return true
		}
		db := NewPendingEventDB(c.self, c.self)
		hash := chain.Uint256{}
		copy(hash[:], event.extra[1:33])
		pendingEvent := PendingEvent{event: *event, account: account, hash: hash}
		db.Store(&pendingEvent, c.self)
		return true
	}
	return false
}

//action onerrorevent ignore
func (c *Contract) OnErrorEvent(event *TxEvent, reason *string, origin_extra []byte) {
	errorEvent := &ErrorTxEvent{}
	data := chain.ReadActionData()

	dec := chain.NewDecoder(data)
	size := dec.Unpack(&errorEvent.event)

	event = &errorEvent.event
	dataSize := size - 1 - len(event.signatures)*66

	VerifySignatures(data[:dataSize], event.signatures)

	assert(event.process == c.process, "Invalid process id")

	errorEvent.reason = dec.UnpackString()

	errorEvent.originExtra = dec.UnpackBytes()

	nonce := c.GetNonce()
	assert(event.nonce >= nonce, "bad nonce!")

	c.StoreNonce(event.nonce)

	if event.amount.Cmp(chain.NewUint128(chain.MAX_AMOUNT, 0)) > 0 {
		c.ShowError("amount too large")
		return
	}

	if c.HandleExpiration(event) {
		return
	}

	db := NewErrorTxEventDB(c.self, c.self)
	it := db.Find(event.nonce)
	assert(!it.IsOk(), "event already exists!")
	db.Store(errorEvent, c.self)
}

//action exec
func (c *Contract) Exec(executor chain.Name) {
	chain.RequireAuth(executor)
	{
		db := NewPendingEventDB(c.self, c.self)
		it := db.Lowerbound(uint64(0))
		if it.IsOk() {
			item := db.GetByIterator(it)
			if c.HandleExpiration(&item.event) {
				db.Remove(it)
				return
			}
		}
	}

	{
		db := NewErrorTxEventDB(c.self, c.self)
		it := db.Lowerbound(uint64(0))
		assert(it.IsOk(), "error event not found!")
		errorEvent := db.GetByIterator(it)
		db.Remove(it)

		c.HandleEvent(&errorEvent.event, errorEvent.originExtra)
	}
}

func (c *Contract) HandleEvent(event *TxEvent, origin_extra []byte) {
	if event.amount.Cmp(chain.NewUint128(chain.MAX_AMOUNT, 0)) > 0 {
		c.ShowError("amount too large")
		return
	}

	if len(event.members) != 1 {
		c.ShowError("multisig event not supported currently")
		return
	}

	if len(event.members) != 1 {
		return
	}

	if c.HandleExpiration(event) {
		return
	}

	clientId := event.members[0]
	account, ok := c.GetAccount(clientId)
	if !ok {
		c.CreateAccount(event)
		return
	}

	if len(origin_extra) == 0 {
		if c.StorePendingEvent(account, event) {
			return
		}
	}

	c.HandleEventWithExtra(account, event, origin_extra)
}

//action execpending
func (c *Contract) ExecPendingEventByExtra(executor chain.Name, nonce uint64, origin_extra []byte) {
	chain.RequireAuth(executor)
	//	db := NewTxEventDB(c.self, c.self)
	db := NewPendingEventDB(c.self, c.self)
	it, item := db.Get(nonce)
	check(it.IsOk(), "pending event not found")
	db.Remove(it)
	check(len(origin_extra) > 0, "origin_extra not not be empty")
	c.HandleEvent(&item.event, origin_extra)
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
	chain.RequireAuth(MIXIN_WTOKENS)
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

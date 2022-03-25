package main

import (
	"encoding/binary"

	"github.com/uuosio/chain"
)

const (
	KEY_NONCE         = 1
	KEY_TX_OUT_INDEX  = 2
	KEY_TX_IN_INDEX   = 3
	KEY_ASSET_INDEX   = 4
	KEY_ACCOUNT_INDEX = 5
	KEY_ACCOUNT_CACHE = 6

	MTG_WORK_EXPIRATION_SECONDS = 3 * 60
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

//contract mixinproxy
type Contract struct {
	self          chain.Name
	firstReceiver chain.Name
	action        chain.Name
	process       chain.Uint128
}

func NewContract(receiver, firstReceiver, action chain.Name) *Contract {
	db := NewProcessDB(MTG_XIN, MTG_XIN)
	it, record := db.Get(receiver.N)
	assert(it.IsOk(), "process not found!")

	c := &Contract{receiver, firstReceiver, action, record.process}
	return c
}

//action initialize
func (c *Contract) Initialize() {
	chain.RequireAuth(c.self)
	db := NewAccountCacheDB(c.self, c.self)
	item := db.Get()
	check(item == nil, "Account cache already initialized")

	item = &AccountCache{}
	item.id = 0
	item.account = GetAccountNameFromId(item.id)

	CreateNewAccount(ACCOUNT_CREATER, ACCOUNT_OWNER, item.account)
	db.Set(item, c.self)
}

//action addasset
func (c *Contract) AddMixinAsset(asset_id chain.Uint128, symbol chain.Symbol) {
	chain.RequireAuth(c.self)
	db := NewMixinAssetDB(c.self, c.self)
	it := db.Find(symbol.Code())
	check(!it.IsOk(), "Asset has already been added")
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

	if !c.checkFee(event) {
		return
	}

	c.HandleEvent(event, origin_extra)
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

	if !c.checkFee(event) {
		return
	}

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

		if c.HandleExpiration(&errorEvent.event) {
			return
		}

		c.HandleEvent(&errorEvent.event, errorEvent.originExtra)
	}
}

//action execpending
func (c *Contract) ExecPendingEventByExtra(executor chain.Name, nonce uint64, origin_extra []byte) {
	chain.RequireAuth(executor)
	//	db := NewTxEventDB(c.self, c.self)
	db := NewPendingEventDB(c.self, c.self)
	it, item := db.Get(nonce)
	check(it.IsOk(), "pending event not found")
	db.Remove(it)
	check(len(origin_extra) > 0, "origin_extra should not be empty")
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

	if len(memo) >= 128 {
		memo = memo[:128]
	}

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

func (c *Contract) HandleEvent(event *TxEvent, origin_extra []byte) {
	if event.amount.Cmp(chain.NewUint128(chain.MAX_AMOUNT, 0)) > 0 {
		c.ShowError("amount too large")
		return
	}

	if len(event.members) != 1 {
		c.ShowError("multisig event not supported currently")
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

func (c *Contract) GetMixinAssetId(sym chain.Symbol) (chain.Uint128, bool) {
	db := NewMixinAssetDB(c.self, c.self)
	it, asset := db.Get(sym.Code())
	if !it.IsOk() {
		return chain.Uint128{}, false
	}
	return asset.asset_id, true
}

func (c *Contract) GetTransferFee(sym chain.Symbol) *chain.Asset {
	feeDB := NewTransferFeeDB(c.self, c.self)
	it, transferFee := feeDB.Get(sym.Code())
	if !it.IsOk() {
		return &chain.Asset{0, sym}
	}
	return &transferFee.fee
}

func (c *Contract) AddFee(fee *chain.Asset) {
	totalFeeDB := NewTotalFeeDB(c.self, c.self)
	it, record := totalFeeDB.Get(fee.Symbol.Code())
	if it.IsOk() {
		record.total.Amount += fee.Amount
		totalFeeDB.Update(it, record, chain.SamePayer)
	} else {
		totalFeeDB.Store(&TotalFee{*fee}, c.self)
	}
}

func (c *Contract) CheckNonce(eventNonce uint64) {
	nonce := c.GetNonce()
	assert(eventNonce >= nonce, "bad nonce!")

	db := NewSubmittedEventDB(c.self, c.self)
	it := db.Find(eventNonce)
	assert(!it.IsOk(), "event already exists!")
	db.Store(&SubmittedEvent{eventNonce}, c.self)

	///increase nonce and remove it from SubmittedEvent db if it's sequential
	for {
		it := db.Find(nonce)
		if !it.IsOk() {
			break
		}
		db.Remove(it)
		c.IncNonce()
		nonce += 1
	}

	//remove stale nonce
	for {
		it := db.Lowerbound(uint64(0))
		if !it.IsOk() {
			break
		}
		record := db.GetByIterator(it)
		if record.nonce > nonce {
			break
		}
		db.Remove(it)
	}
}

func (c *Contract) StoreNonce(eventNonce uint64) {
	db := NewSubmittedEventDB(c.self, c.self)
	it := db.Find(eventNonce)
	assert(!it.IsOk(), "event already exists!")
	db.Store(&SubmittedEvent{eventNonce}, c.self)
}

func (c *Contract) parseAction(extra []byte) *chain.Action {
	if len(extra) < 8*2 {
		return nil
	}

	_account := binary.LittleEndian.Uint64(extra[0:8])
	account := chain.Name{_account}
	if !chain.IsAccount(account) {
		return nil
	}

	_action_name := binary.LittleEndian.Uint64(extra[8:16])
	action_name := chain.Name{_action_name}
	data := extra[16:]

	return &chain.Action{
		account,
		action_name,
		nil,
		data,
	}
}

func (c *Contract) checkFee(event *TxEvent) bool {
	symbol, ok := c.GetSymbol(event.asset)
	if !ok {
		return false
	}

	quantity := chain.NewAsset(int64(event.amount.Uint64()), symbol)

	fee := c.GetTransferFee(symbol)

	if fee.Amount == 0 {
		return true
	}

	if quantity.Amount <= fee.Amount {
		c.AddFee(quantity)
		c.ShowError("transfer amount is less than fee")
		return false
	} else {
		c.AddFee(fee)
	}

	quantity.Amount -= fee.Amount

	//deduct fee from event, in case of refundment
	event.amount.Sub(&event.amount, chain.NewUint128(uint64(fee.Amount), 0))
	return true
}

func (c *Contract) HandleExpiration(event *TxEvent) bool {
	expiration := uint32(event.timestamp/1e9) + MTG_WORK_EXPIRATION_SECONDS
	if expiration > chain.CurrentTimeSeconds() {
		return false
	}

	c.Refund(event, "expired, refund")
	return true
}

func (c *Contract) HandleEventWithExtra(fromAccount chain.Name, event *TxEvent, originExtra []byte) {
	symbol, ok := c.GetSymbol(event.asset)
	if !ok {
		return
	}
	quantity := chain.NewAsset(int64(event.amount.Uint64()), symbol)

	var action *chain.Action
	if len(originExtra) == 0 {
		if len(event.extra) == 0 {
			//transfer to self
			action = nil
		} else {
			check(event.extra[0] == EVENT_NORMAL, "bad extra type")
			extra := event.extra[1:]
			action = c.parseAction(extra)
			if action == nil {
				c.Refund(event, "invalid action data, refund!")
			}
		}
	} else {
		check(event.extra[0] == EVENT_PENDING, "not an extended extra type")
		originExtraHash := event.extra[1:33]
		check(len(originExtraHash) >= 32, "bad extra")
		checksum := chain.Checksum256{}
		copy(checksum[:], originExtraHash)
		chain.AssertSha256(originExtra, checksum) //check extra hash
		op := DecodeOperation(originExtra)
		check(op.Extra[0] == 0, "invalid extra type")
		action = c.parseAction(op.Extra[1:])
		if action == nil {
			c.Refund(event, "invalid action data, refund!")
		}
	}

	ok = c.IssueAsset(fromAccount, quantity, event.timestamp)
	if !ok {
		return
	}

	if action != nil {
		c.SendAction(fromAccount, action)
	}
}

func (c *Contract) SendAction(fromAccount chain.Name, action *chain.Action) {
	action.Authorization = []*chain.PermissionLevel{
		&chain.PermissionLevel{
			Actor:      fromAccount,
			Permission: chain.ActiveName,
		},
	}
	action.Send()
}

func (c *Contract) GetAccount(userId chain.Uint128) (chain.Name, bool) {
	dbAccounts := NewMixinAccountDB(c.self, c.self)
	idxDB := dbAccounts.GetIdxDBByClientId()
	it2 := idxDB.Find(userId)
	if !it2.IsOk() {
		return chain.Name{}, false
	}

	it, record := dbAccounts.Get(it2.Primary)
	if !it.IsOk() {
		return chain.Name{}, false
	}

	return record.eos_account, true
}

func (c *Contract) CreateNewAccount(from chain.Uint128) chain.Name {
	var fromAccount chain.Name
	dbAccounts := NewMixinAccountDB(c.self, c.self)
	idxDB := dbAccounts.GetIdxDBByClientId()
	it2 := idxDB.Find(from)
	assert(!it2.IsOk(), "account already exists!!")
	//		accountId := c.GetNextAccountId()
	fromAccount = c.GetNextAvailableAccount()
	record := MixinAccount{eos_account: fromAccount, client_id: from}
	dbAccounts.Store(&record, c.self)
	return fromAccount
}

func (c *Contract) IssueAsset(fromAccount chain.Name, quantity *chain.Asset, timestamp uint64) bool {
	symbol := quantity.Symbol
	sym_code := symbol.Code()
	db := NewCurrencyStatsDB(MIXIN_WTOKENS, chain.Name{sym_code})
	itr := db.Find(sym_code)
	if !itr.IsOk() {
		maxSupply := chain.NewAsset(MAX_SUPPLY, symbol)
		chain.NewAction(
			&chain.PermissionLevel{MIXIN_WTOKENS, chain.ActiveName},
			MIXIN_WTOKENS,
			chain.NewName("create"),
			c.self,
			maxSupply,
			"create",
		).Send()
	}

	chain.NewAction(
		&chain.PermissionLevel{c.self, chain.ActiveName},
		MIXIN_WTOKENS,
		chain.NewName("issue"),
		c.self,
		quantity,
		"issue",
	).Send()

	chain.NewAction(
		&chain.PermissionLevel{c.self, chain.ActiveName},
		MIXIN_WTOKENS,
		chain.NewName("transfer"),
		c.self,
		fromAccount,
		quantity,
		"transfer",
	).Send()
	return true
}

func (c *Contract) TransferOut(member *chain.Uint128, amount chain.Asset, memo string) {
	assetId, ok := c.GetMixinAssetId(amount.Symbol)
	assert(ok, "unsupported asset id")
	//TODO: make sure balance in MTG is sufficient.
	_amount := chain.NewUint128(uint64(amount.Amount), 0)
	if c.firstReceiver == MIXIN_WTOKENS {
		chain.NewAction(
			&chain.PermissionLevel{c.self, chain.ActiveName},
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
		process:   c.process,
		asset:     assetId,
		members:   []chain.Uint128{*member},
		threshold: 1,
		amount:    *_amount,
		extra:     []byte(memo),
	}

	chain.NewAction(
		&chain.PermissionLevel{c.self, chain.ActiveName},
		MTG_XIN,
		chain.NewName("txrequest"),
		&notify,
	).Send()
}

func (c *Contract) Refund(event *TxEvent, memo string) {
	id := c.GetNextTxRequestNonce()
	notify := TxRequest{
		nonce:     id,
		contract:  c.self,
		process:   c.process,
		asset:     event.asset,
		members:   event.members,
		threshold: event.threshold,
		amount:    event.amount,
		extra:     []byte(memo),
	}

	chain.NewAction(
		&chain.PermissionLevel{c.self, chain.ActiveName},
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
		process:   c.process,
		asset:     assetId,
		members:   []chain.Uint128{clientId},
		threshold: 1,
		amount:    amount,
		extra:     []byte(memo),
	}

	chain.NewAction(
		&chain.PermissionLevel{c.self, chain.ActiveName},
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
	//	assert(!c.nonceIncreased, "nonce already increased")
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
	//	c.nonceIncreased = true
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

func (c *Contract) GetNextAccountId() uint64 {
	return c.GetNextIndex(KEY_ACCOUNT_INDEX, 1)
}

func (c *Contract) GetNextAvailableAccount() chain.Name {
	db := NewAccountCacheDB(c.self, c.self)
	item := db.Get()
	check(item != nil, "No availabe account")

	account := item.account

	item.id += 1
	item.account = GetAccountNameFromId(item.id)

	CreateNewAccount(ACCOUNT_CREATER, ACCOUNT_OWNER, item.account)

	db.Set(item, c.self)
	return account
}

func (c *Contract) GetSymbol(assetId chain.Uint128) (chain.Symbol, bool) {
	assetDB := NewMixinAssetDB(c.self, c.self)
	idxDB := assetDB.GetIdxDBByAssetId()
	itAssetId := idxDB.Find(assetId)
	if !itAssetId.IsOk() {
		return chain.Symbol{}, false
	}

	it, asset := assetDB.Get(itAssetId.Primary)
	if !it.IsOk() {
		return chain.Symbol{}, false
	}
	return asset.symbol, true
}

func (c *Contract) ShowError(err string) {
	chain.NewAction(
		&chain.PermissionLevel{c.self, chain.ActiveName},
		c.self,
		chain.NewName("error"),
		err,
	).Send()
}

func (c *Contract) GetCreateAccountFee() *chain.Asset {
	db := NewCreateAccountFeeDB(c.self, c.self)
	accountFee := db.Get()
	if accountFee == nil {
		return chain.NewAsset(0, chain.NewSymbol("MEOS", 8))
	}
	return &accountFee.fee
}

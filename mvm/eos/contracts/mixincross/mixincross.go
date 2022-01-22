package main

import (
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

var (
	MTG_XIN       = chain.NewName("mtgxinmtgxin")
	MTG_PUBLISHER = chain.NewName("mtgpublisher")
	MIXIN_WTOKENS = chain.NewName("mixinwtokens")
	ACCOUNT_OWNER = chain.NewName("mtgxinmtgxin")
	//uuid: e0148fc6-0e10-470e-8127-166e0829c839
	PROCESS_ID   = chain.Uint128([16]byte{0xe0, 0x14, 0x8f, 0xc6, 0x0e, 0x10, 0x47, 0x0e, 0x81, 0x27, 0x16, 0x6e, 0x08, 0x29, 0xc8, 0x39})
	ASSET_ID_EOS = chain.Uint128([16]byte{0x6c, 0xfe, 0x56, 0x6e, 0x4a, 0xad, 0x47, 0x0b, 0x8c, 0x9a, 0x2f, 0xd3, 0x5b, 0x49, 0xc6, 0x8d})
)

//contract mixinaccount
type Contract struct {
	self           chain.Name
	firstReceiver  chain.Name
	action         chain.Name
	event          *TxEvent
	nonceIncreased bool
}

func NewContract(receiver, firstReceiver, action chain.Name) *Contract {
	c := &Contract{receiver, firstReceiver, action, nil, false}
	// sys.Init(c)
	return c
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

func (c *Contract) HandleErrorEvent(event *TxEvent) {
	c.HandleEventNoNonceChecking(event)
}

func (c *Contract) HandleNormalEvent(event *TxEvent) {
	c.HandleEventNoNonceChecking(event)
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
		if record.nonce >= nonce {
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

func (c *Contract) HandleEventNoNonceChecking(event *TxEvent) {
	assetDB := NewMixinAssetDB(c.self, c.self)
	idxDB := assetDB.GetIdxDBByAssetId()
	itAssetId := idxDB.Find(event.asset)
	if !itAssetId.IsOk() {
		//c.Refund(event, "unsupported asset id, refund!")
		return
	}

	if event.asset == ASSET_ID_EOS {
		if event.amount.Cmp(chain.NewUint128(10000, 0)) < 0 {
			c.ShowError("EOS amount must be greater than 10000")
			return
		}
	} else {
		if event.amount.Cmp(chain.NewUint128(chain.MAX_AMOUNT, 0)) > 0 {
			c.ShowError("amount too large")
			return
		}
	}

	var to chain.Name
	toAccount := string(event.extra)
	if toAccount == "" {
		to = chain.Name{}
	} else {
		to = chain.NewName(toAccount)
		if !chain.IsAccount(to) {
			c.Refund(event, "account does not exists, refund")
			return
		}
	}

	var quantity *chain.Asset
	symbol := c.GetSymbol(event.asset)
	if symbol == chain.NewSymbol("EOS", 4) {
		quantity = chain.NewAsset(int64(event.amount.Uint64())/10000, symbol)
	} else {
		quantity = chain.NewAsset(int64(event.amount.Uint64()), symbol)
	}

	fee := c.GetTransferFee(quantity.Symbol)
	feeAmount := int64(fee.Amount)
	if symbol == chain.NewSymbol("EOS", 4) {
		feeAmount *= 1e4
	}
	if quantity.Amount <= fee.Amount {
		c.AddFee(quantity)
		c.ShowError("transfer amount is less than fee")
		return
	} else {
		c.AddFee(fee)
	}

	quantity.Amount -= fee.Amount
	//deduct fee from event, in case of refund
	event.amount.Sub(&event.amount, chain.NewUint128(uint64(feeAmount), 0))

	if len(event.members) != 1 {
		c.ShowError("multisig event not supported currently")
		return
	}
	from := event.members[0]
	if symbol == chain.NewSymbol("EOS", 4) {
		totalBalance := GetBalance(c.self, chain.TokenContractName, symbol)
		if totalBalance.Amount < quantity.Amount {
			c.Refund(event, "insufficient balance, refund")
			return
		}
		c.TransferTo(from, to, quantity, string(event.extra), event.timestamp)
		c.AddEOSBalance(quantity)
	} else {
		c.TransferTo(from, to, quantity, string(event.extra), event.timestamp)
	}
}

func (c *Contract) TransferTo(from chain.Uint128, to chain.Name, quantity *chain.Asset, memo string, timestamp uint64) {
	expiration := uint32(timestamp/1e9) + MTG_WORK_EXPIRATION_SECONDS
	//handle expired work
	if expiration < chain.CurrentTimeSeconds() {
		clientId := from
		asset_id, ok := c.GetMixinAssetId(quantity.Symbol)
		assert(ok, "asset not found!")
		amount := chain.NewUint128(uint64(quantity.Amount), 0)
		if quantity.Symbol == chain.NewSymbol("EOS", 4) {
			amount.Mul(amount, chain.NewUint128(10000, 0))
		}
		c.HandleRefund(clientId, asset_id, *amount, "expired, refund")
		return
	}

	var fromAccount chain.Name
	dbAccounts := NewMixinAccountDB(c.self, c.self)
	idxDB := dbAccounts.GetIdxDBByClientId()
	it2 := idxDB.Find(from)

	if !it2.IsOk() {
		fee := c.GetCreateAccountFee()
		if fee.Amount != 0 {
			if quantity.Symbol != chain.NewSymbol("EOS", 4) {
				c.ShowError("invalid asset for creating account")
				return
			}
			if quantity.Amount < fee.Amount {
				c.ShowError("not enough fee for creating account")
				return
			}
			quantity.Amount -= fee.Amount
		}
		//		accountId := c.GetNextAccountId()
		fromAccount = c.GetNexAvailableAccount()
		record := MixinAccount{eos_account: fromAccount, client_id: from}
		dbAccounts.Store(&record, c.self)
	} else {
		it, record := dbAccounts.Get(it2.Primary)
		assert(it.IsOk(), "account not found!")
		fromAccount = record.eos_account
	}

	if quantity.Symbol == chain.NewSymbol("EOS", 4) {
		chain.NewAction(
			&chain.PermissionLevel{c.self, chain.ActiveName},
			chain.TokenContractName,
			chain.NewName("transfer"),
			c.self,      //from
			fromAccount, // to,
			quantity,    //quantity
			memo,
		).Send()
		if (to != chain.Name{}) && fromAccount != to {
			chain.NewAction(
				&chain.PermissionLevel{fromAccount, chain.ActiveName},
				chain.TokenContractName,
				chain.NewName("transfer"),
				fromAccount, //from
				to,          // to,
				quantity,    //quantity
				memo,
			).Send()
		}
	} else {
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

		if (to != chain.Name{}) && to != fromAccount {
			chain.NewAction(
				&chain.PermissionLevel{fromAccount, chain.ActiveName},
				MIXIN_WTOKENS,
				chain.NewName("transfer"),
				fromAccount,
				to,
				quantity,
				"transfer",
			).Send()
		}
	}
}

func (c *Contract) TransferOut(member *chain.Uint128, amount chain.Asset, memo string) {
	assetId, ok := c.GetMixinAssetId(amount.Symbol)
	assert(ok, "unsupported asset id")
	//TODO: make sure balance in MTG is sufficient.
	_amount := chain.NewUint128(uint64(amount.Amount), 0)
	if amount.Symbol == chain.NewSymbol("EOS", 4) {
		_amount.Mul(_amount, chain.NewUint128(10000, 0))
		c.SubEOSBalance(&amount)
	}

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
		process:   PROCESS_ID,
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
		process:   PROCESS_ID,
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
		process:   PROCESS_ID,
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

func (c *Contract) GetNextAccountId() uint64 {
	return c.GetNextIndex(KEY_ACCOUNT_INDEX, 1)
}

func (c *Contract) GetNexAvailableAccount() chain.Name {
	db := NewAccountCacheDB(c.self, c.self)
	item := db.Get()
	check(item != nil, "No availabe account")

	account := item.account

	item.id += 1
	item.account = GetAccountNameFromId(item.id)

	CreateNewAccount(c.self, ACCOUNT_OWNER, item.account)

	db.Set(item, c.self)
	return account
}

func (c *Contract) GetSymbol(assetId chain.Uint128) chain.Symbol {
	assetDB := NewMixinAssetDB(c.self, c.self)
	idxDB := assetDB.GetIdxDBByAssetId()
	itAssetId := idxDB.Find(assetId)
	assert(itAssetId.IsOk(), "asset id not found")
	it, asset := assetDB.Get(itAssetId.Primary)
	assert(it.IsOk(), "asset not found")
	return asset.symbol
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
		return chain.NewAsset(0, chain.NewSymbol("EOS", 4))
	}
	return &accountFee.fee
}

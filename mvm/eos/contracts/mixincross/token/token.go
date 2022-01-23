package main

import (
	"github.com/uuosio/chain"
)

var (
	MASTER_ACCOUNT = chain.NewName("mixincrossss")
)

func check(b bool, msg string) {
	chain.Check(b, msg)
}

//table accounts
type account struct {
	balance chain.Asset //primary: t.balance.Symbol.Code()
}

//table stat
type currency_stats struct {
	supply     chain.Asset //primary: t.supply.Symbol.Code()
	max_supply chain.Asset
	issuer     chain.Name
}

//table bindaccounts
type MixinAccount struct {
	eos_account chain.Name    //primary : t.eos_account.N
	client_id   chain.Uint128 //IDX128: ByClientId : t.client_id : t.client_id
}

//table mixinassets
type MixinAsset struct {
	symbol   chain.Symbol  //primary : t.symbol.Code()
	asset_id chain.Uint128 //IDX128: ByAssetId : t.asset_id : t.asset_id
}

//contract token
type Token struct {
	receiver      chain.Name
	firstReceiver chain.Name
	action        chain.Name
}

func NewAccountDB(code chain.Name, scope chain.Name) *accountDB {
	return NewaccountDB(code, scope)
}

func NewCurrencyStatsDB(code chain.Name, scope chain.Name) *currency_statsDB {
	return Newcurrency_statsDB(code, scope)
}

func NewContract(receiver, firstReceiver, action chain.Name) *Token {
	return &Token{receiver, firstReceiver, action}
}

//action create
func (token *Token) Create(issuer chain.Name, maximum_supply chain.Asset) {
	chain.RequireAuth(token.receiver)
	check(maximum_supply.Symbol.IsValid(), "invalid symbol name")
	check(maximum_supply.IsValid(), "invalid supply")
	check(maximum_supply.Amount > 0, "max-supply must be positive")

	sym_code := maximum_supply.Symbol.Code()
	db := NewCurrencyStatsDB(token.receiver, chain.Name{sym_code})
	itr := db.Find(sym_code)
	check(!itr.IsOk(), "token with symbol already exists")

	stats := &currency_stats{}
	stats.supply.Symbol = maximum_supply.Symbol
	stats.max_supply = maximum_supply
	stats.issuer = issuer
	db.Store(stats, token.receiver)
}

//action issue
func (token *Token) Issue(to chain.Name, quantity chain.Asset, memo string) {
	check(quantity.Symbol.IsValid(), "invalid symbol name")
	check(len(memo) <= 256, "memo has more than 256 bytes")

	sym_code := quantity.Symbol.Code()
	db := NewCurrencyStatsDB(token.receiver, chain.Name{sym_code})
	it, item := db.Get(sym_code)
	check(it.IsOk(), "token with symbol does not exist, create token before issue")
	check(to == item.issuer, "tokens can only be issued to issuer account")

	chain.RequireAuth(item.issuer)
	check(quantity.IsValid(), "invalid quantity")
	check(quantity.Amount > 0, "must issue positive quantity")

	check(quantity.Symbol == item.supply.Symbol, "symbol precision mismatch")
	check(quantity.Amount <= item.max_supply.Amount-item.supply.Amount, "quantity exceeds available supply")

	item.supply.Add(&quantity)
	db.Update(it, item, item.issuer)

	token.addBalance(to, quantity, to)
}

//action retire
func (token *Token) Retire(quantity chain.Asset, memo string) {
	check(quantity.Symbol.IsValid(), "invalid symbol name")
	check(len(memo) <= 256, "memo has more than 256 bytes")
	stats := NewCurrencyStatsDB(token.receiver, chain.Name{quantity.Symbol.Code()})
	it, item := stats.Get(quantity.Symbol.Code())
	check(it.IsOk(), "token with symbol does not exist")
	chain.RequireAuth(item.issuer)
	check(quantity.IsValid(), "invalid quantity")
	check(quantity.Amount > 0, "must retire positive quantity")
	check(quantity.Symbol == item.supply.Symbol, "symbol precision mismatch")

	item.supply.Sub(&quantity)
	stats.Update(it, item, chain.SamePayer)
	token.subBalance(item.issuer, quantity)
}

//action retireex
func (token *Token) RetireEx(account chain.Name, quantity chain.Asset, memo string) {
	chain.RequireAuth(token.receiver)

	check(quantity.Symbol.IsValid(), "invalid symbol name")
	check(len(memo) <= 256, "memo has more than 256 bytes")

	stats := NewCurrencyStatsDB(token.receiver, chain.Name{quantity.Symbol.Code()})
	it, item := stats.Get(quantity.Symbol.Code())
	check(it.IsOk(), "token with symbol does not exist")
	// chain.RequireAuth(item.issuer)

	check(quantity.IsValid(), "invalid quantity")
	check(quantity.Amount > 0, "must retire positive quantity")
	check(quantity.Symbol == item.supply.Symbol, "symbol precision mismatch")

	item.supply.Sub(&quantity)
	stats.Update(it, item, chain.SamePayer)
	// token.subBalanceEx(account, quantity, chain.Name{0})
}

//action transfer
func (token *Token) Transfer(from chain.Name, to chain.Name, quantity chain.Asset, memo string) {
	check(from != to, "cannot transfer to self")
	chain.RequireAuth(from)
	check(chain.IsAccount(to), "to account does not exist")

	stats := NewCurrencyStatsDB(token.receiver, chain.Name{quantity.Symbol.Code()})
	it, st := stats.Get(quantity.Symbol.Code())
	check(it.IsOk(), "token with symbol does not exist")

	//	chain.RequireRecipient(from)
	chain.RequireRecipient(to)

	check(quantity.IsValid(), "invalid quantity")
	check(quantity.Amount > 0, "must transfer positive quantity")
	check(quantity.Symbol == st.supply.Symbol, "symbol precision mismatch")
	check(len(memo) <= 256, "memo has more than 256 bytes")

	payer := chain.Name{0}
	if chain.HasAuth(to) {
		payer = to
	} else {
		payer = from
	}

	//this is a deposit transfer
	if from == MASTER_ACCOUNT {
		token.subBalance(from, quantity)
		token.addBalance(to, quantity, payer)
		return
	}

	db := NewMixinAccountDB(MASTER_ACCOUNT, MASTER_ACCOUNT)
	it = db.Find(to.N)
	//to account is not a mixinaccount, no tx request
	if !it.IsOk() {
		token.subBalance(from, quantity)
		token.addBalance(to, quantity, payer)
		return
	}

	token.subBalance(from, quantity)

	chain.NewAction(
		chain.NewPermissionLevel(token.receiver, chain.ActiveName),
		token.receiver,
		chain.NewName("retireex"),
		from,
		quantity,
		"retire for transfer out",
	).Send()

	//mixin asset transfer request
	token.TxRequest(from, to, quantity, memo)
}

func (token *Token) TxRequest(from chain.Name, to chain.Name, quantity chain.Asset, memo string) {
	chain.NewAction(
		&chain.PermissionLevel{token.receiver, chain.ActiveName},
		MASTER_ACCOUNT,
		chain.NewName("ontransfer"),
		from,
		to,
		quantity,
		memo,
	).Send()
}

//action open
func (token *Token) Open(owner chain.Name, symbol chain.Symbol, ram_payer chain.Name) {
	chain.RequireAuth(ram_payer)
	check(chain.IsAccount(owner), "owner account does not exist")
	stats := NewCurrencyStatsDB(token.receiver, chain.Name{symbol.Code()})
	it, st := stats.Get(symbol.Code())
	check(it.IsOk(), "symbol does not exist")
	check(st.supply.Symbol == symbol, "symbol precision mismatch")

	accountDB := NewAccountDB(token.receiver, owner)
	it = accountDB.Find(symbol.Code())
	if !it.IsOk() {
		account := &account{}
		account.balance = chain.Asset{0, symbol}
		accountDB.Store(account, ram_payer)
	}
}

//action close
func (token *Token) Close(owner chain.Name, symbol chain.Symbol) {
	chain.RequireAuth(owner)
	accountDB := NewAccountDB(token.receiver, owner)
	it, item := accountDB.Get(symbol.Code())
	check(it.IsOk(), "Balance row already deleted or never existed. Action won't have any effect.")
	check(item.balance.Amount == 0, "Cannot close because the balance is not zero.")
	accountDB.Remove(it)
}

func (token *Token) subBalance(owner chain.Name, value chain.Asset) {
	accountDB := NewAccountDB(token.receiver, owner)
	it, from := accountDB.Get(value.Symbol.Code())
	check(it.IsOk(), "no balance object found")
	check(from.balance.Amount >= value.Amount, "overdrawn balance")
	from.balance.Sub(&value)
	accountDB.Update(it, from, owner)
}

func (token *Token) addBalance(owner chain.Name, value chain.Asset, ramPayer chain.Name) {
	accountDB := NewAccountDB(token.receiver, owner)
	it, to := accountDB.Get(value.Symbol.Code())
	if !it.IsOk() {
		account := &account{balance: value}
		accountDB.Store(account, ramPayer)
	} else {
		to.balance.Add(&value)
		accountDB.Update(it, to, chain.Name{0})
	}
}

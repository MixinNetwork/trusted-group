package main

import (
	"github.com/uuosio/chain"
)

//table accounts ignore
type account struct {
	balance chain.Asset //primary: t.balance.Symbol.Code()
}

//table stat ignore
type currency_stats struct {
	supply     chain.Asset //primary: t.supply.Symbol.Code()
	max_supply chain.Asset
	issuer     chain.Name
}

func NewAccountDB(code chain.Name, scope chain.Name) *accountDB {
	return NewaccountDB(code, scope)
}

func NewCurrencyStatsDB(code chain.Name, scope chain.Name) *currency_statsDB {
	return Newcurrency_statsDB(code, scope)
}

func GetBalance(owner chain.Name, tokenAccount chain.Name, sym chain.Symbol) chain.Asset {
	accountDB := NewAccountDB(tokenAccount, owner)
	it, to := accountDB.Get(sym.Code())
	if it.IsOk() {
		return to.balance
	} else {
		return chain.Asset{0, sym}
	}
}

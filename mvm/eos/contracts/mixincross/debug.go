//go:build debug
// +build debug

package main

import (
	"github.com/uuosio/chain"
	"github.com/uuosio/chain/database"
)

const (
	DEBUG = true
)

func ClearDB(db database.MultiIndexInterface) {
	for {
		it := db.Lowerbound(0)
		if !it.IsOk() {
			break
		}
		db.Remove(it)
	}
}

func ClearSingletonDB(db *database.SingletonDB) {
	db.Remove()
}

//action updateauth
func (c *Contract) UpdateAuth(account chain.Name) {
	chain.RequireAuth(c.self)
	auth := Authority{
		uint32(1),     //threshold
		[]KeyWeight{}, //keys
		[]PermissionLevelWeight{
			PermissionLevelWeight{
				PermissionLevel{
					c.self,
					chain.ActiveName,
				},
				uint16(1),
			},
			PermissionLevelWeight{
				PermissionLevel{
					c.self,
					chain.NewName("multisig"),
				},
				uint16(1),
			},
		}, //accounts
		[]WaitWeight{},
	}
	chain.NewAction(
		&chain.PermissionLevel{account, chain.ActiveName},
		chain.NewName("eosio"),
		chain.NewName("updateauth"),
		account,          //account
		chain.ActiveName, //permission
		chain.OwnerName,  //parent
		&auth,
	).Send()
}

//action clear
func (c *Contract) clear() {
	chain.RequireAuth(c.self)
	ClearSingletonDB(NewAccountCacheDB(c.self, c.self).db)
	ClearDB(NewMixinAccountDB(c.self, c.self).MultiIndexInterface)
	// return

	ClearSingletonDB(NewEOSBalanceDB(c.self, c.self).db)
	ClearDB(NewCounterDB(c.self, c.self).MultiIndexInterface)

	// ClearDB(NewMixinAssetDB(c.self, c.self).MultiIndexInterface)

	ClearDB(NewTxEventDB(c.self, c.self).MultiIndexInterface)
	ClearDB(NewMTGWorkDB(c.self, c.self).MultiIndexInterface)
}

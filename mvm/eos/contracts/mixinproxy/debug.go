//go:build debug
// +build debug

package main

import (
	"github.com/uuosio/chain"
	"github.com/uuosio/chain/database"
)

const (
	DEBUG            = true
	KEY_COUNTER_TEST = 7
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
	ClearDB(NewCounterDB(c.self, c.self).MultiIndexInterface)

	// ClearDB(NewMixinAssetDB(c.self, c.self).MultiIndexInterface)

	ClearDB(NewTxEventDB(c.self, c.self).MultiIndexInterface)
	// ClearDB(NewMTGWorkDB(c.self, c.self).MultiIndexInterface)
}

//action test
func (c *Contract) test() {
	c.GetNextIndex(KEY_COUNTER_TEST, 1)
}

//action testname
func (c *Contract) testName() {
	// aaaaaaaaamvm
	// name := GetAccountNameFromId((uint64)i)
	name := GetAccountNameFromId(uint64(30))
	chain.Check(name == chain.NewName("aaaaaaaa5mvm"), "bad value")

	name = GetAccountNameFromId(uint64(31))
	chain.Check(name == chain.NewName("aaaaaaabamvm"), "bad value")

	// for i := 0; i <= 100; i++ {
	// 	name := GetAccountNameFromId(uint64(i))
	// 	chain.Println("++++++++:", i, name)
	// }
}

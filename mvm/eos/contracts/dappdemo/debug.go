//go:build debug
// +build debug

package main

import "github.com/uuosio/chain"

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

package main

import (
	"github.com/uuosio/chain"
)

//table txevents
type TxEvent struct {
	nonce     uint64 //primary : t.nonce
	process   chain.Uint128
	asset     chain.Uint128
	members   []chain.Uint128
	threshold int32
	amount    chain.Uint128
	extra     []byte
	timestamp uint64
	signature []byte
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
	Id    uint64 //primary : t.Id
	Count uint64
}

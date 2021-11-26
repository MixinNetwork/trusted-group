package main

import (
	"github.com/uuosio/chain"
)

//table mytable
type MyData struct {
	primary uint64         //primary : t.primary
	a1      uint64         //IDX64 		: Bya1 : t.a1 : t.a1
	a2      chain.Uint128  //IDX128 		: Bya2 : t.a2 : t.a2
	a3      chain.Uint256  //IDX256 		: Bya3 : t.a3 : t.a3
	a4      float64        //IDXFloat64 	: Bya4 : t.a4 : t.a4
	a5      chain.Float128 //IDXFloat128 	: Bya5 : t.a5 : t.a5
}

//table processes
type Process struct {
	contract chain.Name    //primary : t.contract.N
	process  chain.Uint128 //IDX128 : ByProcess : t.process : t.process
}

//table logs
type TxLog struct {
	id        uint64 //primary : t.id
	nonce     uint64
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

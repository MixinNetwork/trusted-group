package main

import "github.com/uuosio/chain"

//packer
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

//packer
type KeyWeight struct {
	Key    chain.PublicKey
	Weight uint16
}

//packer
type PermissionLevel struct {
	Actor      chain.Name
	Permission chain.Name
}

//packer
type PermissionLevelWeight struct {
	Permission PermissionLevel
	Weight     uint16
}

//packer
type WaitWeight struct {
	WaitSec uint32
	Weight  uint16
}

//packer
type Authority struct {
	Threshold uint32
	Keys      []KeyWeight
	Accounts  []PermissionLevelWeight
	Waits     []WaitWeight
}

//packer
type NewAccount struct {
	Creator chain.Name
	Name    chain.Name
	Owner   Authority
	Active  Authority
}

package main

import "github.com/uuosio/chain"

const (
	KEY_NONCE            = 1
	KEY_TX_REQUEST_INDEX = 2
	KEY_FINISHED_REQUEST = 3
)

const (
	MAX_REMOVE_RECORD_COUNT = 30
)

var (
	MTG_CONTRACT = chain.NewName("mtgxinmtgxin")
	//uuid: 49b00892-6954-4826-aaec-371ca165558a
	PROCESS_ID = chain.Uint128([16]byte{0x49, 0xb0, 0x08, 0x92, 0x69, 0x54, 0x48, 0x26, 0xaa, 0xec, 0x37, 0x1c, 0xa1, 0x65, 0x55, 0x8a})
)

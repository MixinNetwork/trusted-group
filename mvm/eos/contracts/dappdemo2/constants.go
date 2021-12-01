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
	//uuid: 18a62033-8845-455f-bcde-0e205ef4da44
	PROCESS_ID = chain.Uint128([16]byte{0x18, 0xa6, 0x20, 0x33, 0x88, 0x45, 0x45, 0x5f, 0xbc, 0xde, 0x0e, 0x20, 0x5e, 0xf4, 0xda, 0x44})
)

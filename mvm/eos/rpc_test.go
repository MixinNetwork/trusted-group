package eos

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/learnforpractice/goeoslib/chain"
	"github.com/stretchr/testify/assert"
)

const (
	MTG_XIN_CONTRACT  = "mtgxinmtgxin"
	TX_REQUEST_ACTION = "txrequest"
)

func TestGetTable(t *testing.T) {
	api := chain.NewChainApi("http://127.0.0.1:9000")
	result, err := api.GetTableRows(
		false,            //json bool,
		MTG_XIN_CONTRACT, //code string,
		MTG_XIN_CONTRACT, //scope string,
		"txrequests",     //table string,
		"1",              //lowerbound string,
		"",               //upperbound string,
		10,               //limit int,
		"i64",            //keyType string,
		1,                //indexPosition int, 4 is the index database of To account
		false,            //reverse bool,
		false,            //showPayer bool,
	)

	t.Logf("+++++++++++hello\n")
	if err != nil {
		panic(err)
	}
	t.Logf("+++%v", result)
	rows, err := result.Get("rows")
	if err != nil {
		t.Logf("+++++%s", result.ToString())
		panic(err)
	}
	_rows, ok := rows.([]interface{})
	if !ok {
		panic(err)
	}
	for _, row := range _rows {
		r, ok := row.(string)
		if !ok {
			panic(err)
		}
		b, err := hex.DecodeString(r)
		if err != nil {
			continue
		}
		notify := &TxLog{}
		notify.Unpack(b)
		t.Logf("+++++++++notify.id: %d\n", notify.id)
		evt := convertTxLogToEvent(notify)
		// evt, err := encoding.DecodeEvent(b)
		t.Logf("%v", evt)
	}
}

func TestGetNonce(t *testing.T) {
	address := "helloworld11"
	api := chain.NewChainApi("http://127.0.0.1:9000")
	key := fmt.Sprintf("%d", KEY_NONCE)
	result, err := api.GetTableRows(
		false,      //json bool,
		address,    //code string,
		address,    //scope string,
		"counters", //table string,
		key,        //lowerbound string,
		"",         //upperbound string,
		10,         //limit int,
		"i64",      //keyType string,
		1,          //indexPosition int
		false,      //reverse bool,
		false,      //showPayer bool,
	)
	if err != nil {
		panic(err)
	}

	nonce, err := result.GetString("rows", 0)
	if err != nil {
		panic(err)
	}
	t.Log("++++nonce:", nonce)
	if len(nonce) != 32 {
		panic("bad nonce value")
	}

	b, err := hex.DecodeString(nonce)
	if err != nil {
		panic(err)
	}

	_nonce := binary.LittleEndian.Uint64(b[8:])
	t.Logf("nonce: %d", _nonce)
}

func TestGetCounter(t *testing.T) {
	address := "helloworld11"
	api := chain.NewChainApi("http://127.0.0.1:9000")

	key := fmt.Sprintf("%d", KEY_NONCE)
	result, err := api.GetTableRows(
		false,      //json bool,
		address,    //code string,
		address,    //scope string,
		"counters", //table string,
		key,        //lowerbound string,
		"",         //upperbound string,
		10,         //limit int,
		"i64",      //keyType string,
		1,          //indexPosition int
		false,      //reverse bool,
		false,      //showPayer bool,
	)
	if err != nil {
		panic(err)
	}

	nonce, err := result.GetString("rows", 0)
	if err != nil {
		panic(err)
	}

	if len(nonce) != 32 {
		panic("bad nonce value")
	}

	b, err := hex.DecodeString(nonce)
	if err != nil {
		panic(err)
	}
	t.Logf("++counter: %d", binary.LittleEndian.Uint64(b[8:]))
}

func TestGetActions(t *testing.T) {
	address := MTG_XIN_CONTRACT
	api := chain.NewChainApi("http://127.0.0.1:9000")
	r, err := api.GetActions(address, 0, 10)
	if err != nil {
		panic(err)
	}

	actions, err := r.GetArray("actions")
	if err != nil {
		logger.Verbosef(`Get("actions") => %v`, err)
		panic(err)
	}

	//	lastIndex := uint64(0)
	for _, action := range actions {
		obj, ok := chain.NewJsonObjectFromInterface(action)
		if !ok {
			panic(err)
		}

		seq, err := obj.GetUint64("account_action_seq")
		if err != nil {
			panic(err)
		}
		t.Logf("+++seq: %d", seq)

		receiver, err := obj.GetString("action_trace", "receiver")
		if err != nil {
			panic(err)
		}
		if receiver != MTG_XIN_CONTRACT {
			panic(err)
		}
		actionObj, err := obj.GetJsonObject("action_trace", "act")
		if err != nil {
			panic(err)
		}
		account, err := actionObj.GetString("account")
		if err != nil {
			panic(err)
		}
		if account != MTG_XIN_CONTRACT {
			panic(err)
		}

		action_name, err := actionObj.GetString("name")
		if err != nil {
			panic(err)
		}
		if action_name != TX_REQUEST_ACTION {
			panic(err)
		}

		// actor, err := actionObj.GetString("authorization", 0, "actor")
		// if err != nil {
		// 	panic(err)
		// }
		// if actor != address {
		// 	panic(err)
		// }

		// permission, err := actionObj.GetString("authorization", 0, "permission")
		// if err != nil {
		// 	panic(err)
		// }
		// if permission != "active" {
		// 	panic(err)
		// }

		data, err := actionObj.GetString("data")
		if err != nil {
			panic(err)
		}
		t.Logf("%s", data)
		b, err := hex.DecodeString(data)
		if err != nil {
			continue
		}
		notify := &TxRequest{}
		notify.Unpack(b)
		t.Logf("%v", notify)
	}
}

func TestGetAccount(t *testing.T) {
	api := chain.NewChainApi("http://127.0.0.1:9000")
	r, err := api.GetAccount("notexists")
	assert := assert.New(t)
	assert.NotNil(err)
	t.Logf("%v", r)
}

package eos

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"

	"github.com/MixinNetwork/trusted-group/mvm/eos/chain"
	"github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/crypto"
	// "github.com/uuosio/go-uuoskit/uuoskit"
)

const (
	KEY_NONCE     = 1
	TX_LOG_ACTION = "ontxlog"
	ClockTick     = 3 * time.Second
)

type Configuration struct {
	Store         string `toml:"store"`
	RPC           string `toml:"rpc"`
	PrivateKey    string `toml:"key"`
	MixinContract string `toml:"mixin_contract"`
}

type Engine struct {
	db            *badger.DB
	rpc           *chain.ChainApi
	key           string
	mixinContract string
}

func Boot(conf *Configuration) (*Engine, error) {
	rpc := chain.NewChainApi(conf.RPC)
	db := openBadger(conf.Store)
	e := &Engine{db: db, rpc: rpc, key: conf.PrivateKey, mixinContract: conf.MixinContract}
	if e.key != "" {
		chain.GetWallet().Import("test", conf.PrivateKey)
	}
	go e.loopHandleContracts()
	return e, nil
}

func (e *Engine) Hash(b []byte) []byte {
	return crypto.Keccak256(b)
}

func (e *Engine) VerifyAddress(addr string, extra []byte) error {
	info, err := e.rpc.GetAccount(addr)
	if err != nil {
		return err
	}

	lastUpdate, err := info.GetTime("last_code_update")
	if err != nil {
		return err
	}

	if lastUpdate.Add(time.Duration(60 * 2)).Before(time.Now()) {
		return nil
	} else {
		return fmt.Errorf("too yong %v", lastUpdate)
	}
}

func (e *Engine) SetupNotifier(address string) error {
	notifier := e.key
	if notifier == "" {
		notifier = address
	}
	old := e.storeReadContractNotifier(address)
	if old == notifier {
		return nil
	} else if old != "" {
		panic(old)
	}
	return e.storeWriteContractNotifier(address, notifier)
}

func (e *Engine) AddProcess(id, address string) error {
	if !e.IsPublisher() {
		return nil
	}

	_id := chain.Uint128{}
	copy(_id[:], uuidToBytes(id))
	action := chain.NewAction(
		chain.PermissionLevel{Actor: chain.NewName(e.mixinContract), Permission: chain.NewName("active")},
		chain.NewName(e.mixinContract),
		chain.NewName("addprocess"),
		chain.NewName(address),
		&_id,
	)
	e.rpc.PushAction(action)
	return nil
}

func (e *Engine) EstimateCost(events []*encoding.Event) (common.Integer, error) {
	return common.NewInteger(0), nil
}

func (e *Engine) EnsureSendGroupEvents(address string, events []*encoding.Event) error {
	return e.storeWriteGroupEvents(address, events)
}

func (e *Engine) ReceiveGroupEvents(block uint64) ([]*encoding.Event, error) {
	events := make([]*encoding.Event, 0, 1)
	offset := e.storeReadContractLogsOffset(e.mixinContract)
	r, err := e.rpc.GetActions(e.mixinContract, int(offset), 10)
	if err != nil {
		return nil, err
	}

	actions, err := r.GetArray("actions")
	if err != nil {
		return nil, err
	}

	if len(actions) == 0 {
		return nil, errors.New("no new action record")
	}

	logger.Verbosef("ReceiveGroupEvents offset %d, actions size:%d", offset, len(actions))

	lastIndex := uint64(0)
	for _, action := range actions {
		obj, ok := chain.NewJsonObjectFromInterface(action)
		if !ok {
			continue
		}

		seq, err := obj.GetUint64("account_action_seq")
		if err != nil {
			continue
		}
		lastIndex = seq

		receiver, err := obj.GetString("action_trace", "receiver")
		if err != nil {
			continue
		}
		if receiver != e.mixinContract {
			continue
		}
		actionObj, err := obj.GetJsonObject("action_trace", "act")
		if err != nil {
			continue
		}
		account, err := actionObj.GetString("account")
		if err != nil {
			continue
		}
		if account != e.mixinContract {
			continue
		}

		action_name, err := actionObj.GetString("name")
		if err != nil {
			continue
		}
		if action_name != TX_LOG_ACTION {
			continue
		}

		actor, err := actionObj.GetString("authorization", 0, "actor")
		if err != nil {
			continue
		}

		if actor != e.mixinContract {
			continue
		}

		permission, err := actionObj.GetString("authorization", 0, "permission")
		if err != nil {
			continue
		}
		if permission != "active" {
			continue
		}

		data, err := actionObj.GetString("hex_data")
		if err != nil {
			data, err = actionObj.GetString("data")
			if err != nil {
				continue
			}
		}

		b, err := hex.DecodeString(data)
		if err != nil {
			continue
		}
		txLog := &TxLog{}
		size, err := txLog.Unpack(b)
		if err != nil {
			continue
		}

		if size != len(b) {
			logger.Verbosef("txLog.Unpack: binary size does not as expected: %d, got %d", len(b), size)
			continue
		}

		evt := convertTxRequestToEvent(notify)
		// evt, err := encoding.DecodeEvent(b)
		logger.Verbosef("loopGetLogs(%s) => DecodeEvent(%x) => %v, %v", e.mixinContract, b, evt, err)
		if err != nil {
			continue
		}
		events = append(events, evt)
		//FIXME: check if process id belongs to contract
	}

	//FIXME: consensus on last finished tx request index
	// if e.IsPublisher() && lastIndex != 0 {
	// 	e.clearFinishedTxRequest(e.mixinContract, lastIndex)
	// }
	e.storeWriteContractLogsOffset(e.mixinContract, lastIndex+1)
	return events, nil
}

func (e *Engine) IsPublisher() bool {
	return e.key != ""
}

func (e *Engine) GetAddressNonce(address string) (uint64, error) {
	key := fmt.Sprintf("%d", KEY_NONCE)
	result, err := e.rpc.GetTableRows(
		false,      //json bool,
		address,    //code string,
		address,    //scope string,
		"counters", //table string,
		key,        //lowerbound string,
		key,        //upperbound string,
		10,         //limit int,
		"i64",      //keyType string,
		1,          //indexPosition int
		false,      //reverse bool,
		false,      //showPayer bool,
	)
	if err != nil {
		return 0, err
	}

	// logger.Verbosef("+++++address: %s, result: %v", address, result)
	nonce, err := result.GetString("rows", 0)
	if err != nil {
		return 0, err
	}

	if len(nonce) != 32 {
		return 0, fmt.Errorf("bad nonce value")
	}

	b, err := hex.DecodeString(nonce)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b[8:]), nil
}

func (e *Engine) loopSendGroupEvents(address string) {
	logger.Verbosef("Engine.loopSendGroupEvents(%s)", address)
	for e.IsPublisher() {
		time.Sleep(ClockTick)
		nonce, err := e.GetAddressNonce(address)
		if err != nil {
			logger.Verbosef("+++GetAddressNonce(%v) => %v", address, err)
			nonce = 0
		}

		evts, err := e.storeListGroupEvents(address, nonce, 100)
		if err != nil {
			panic(err)
		}
		for _, evt := range evts {
			err := e.pushEvent(address, evt, true)
			logger.Verbosef("pushEvent(%v, %v) => (err: %v)", address, evt, err)
			//TODO: refund on error
			if err != nil {
			}
		}
	}
}

func (e *Engine) loopHandleContracts() {
	contracts := make(map[string]bool)
	for {
		time.Sleep(ClockTick)
		all, err := e.storeListContractAddresses()
		if err != nil {
			panic(err)
		}
		for _, c := range all {
			if contracts[c] {
				continue
			}
			contracts[c] = true
			//			go e.loopGetLogs(c)
			go e.loopSendGroupEvents(c)
		}
		if !e.IsPublisher() {
			continue
		}
	}
}

func (e *Engine) pushEvent(address string, evt *encoding.Event, good bool) error {
	var actionName chain.Name
	if good {
		actionName = chain.NewName("onevent")
	} else {
		actionName = chain.NewName("onbadevent")
	}
	action := chain.NewAction(
		chain.PermissionLevel{Actor: chain.NewName(e.mixinContract), Permission: chain.NewName("active")},
		chain.NewName(address),
		actionName,
	)

	process := uuidToBytes(evt.Process)
	asset := uuidToBytes(evt.Asset)

	txEvent := TxEvent{}

	txEvent.nonce = evt.Nonce

	copy(txEvent.process[:], process)
	copy(txEvent.asset[:], asset)
	txEvent.members = make([]chain.Uint128, len(evt.Members))
	for i, member := range evt.Members {
		copy(txEvent.members[i][:], uuidToBytes(member))
	}
	txEvent.threshold = int32(evt.Threshold)

	amount, err := evt.Amount.MarshalMsgpack()
	if err != nil {
		return err
	}
	amount = reverseBytes(amount)
	//FIXME: amount overflow
	copy(txEvent.amount[:], amount)

	txEvent.extra = evt.Extra
	txEvent.timestamp = evt.Timestamp
	txEvent.signature = evt.Signature

	action.Data = txEvent.Pack()
	r, err := e.rpc.PushAction(action)
	if err != nil {
		logger.Verbosef("++++++PushAction => err: %v", err)
		return err
	}
	console, err := r.GetString("processed", "action_traces", 0, "console")
	if err != nil {
		panic(err)
	}
	logger.Verbosef("++++++pushEvent:%s => %s", address, console)
	return nil
}

func (e *Engine) clearFinishedTxRequest(address string, lastIndex uint64) error {
	action := chain.NewAction(
		chain.PermissionLevel{Actor: chain.NewName(e.mixinContract), Permission: chain.NewName("active")},
		chain.NewName(address),
		chain.NewName("clearreqs"),
		lastIndex,
	)

	_, err := e.rpc.PushAction(action)
	if err != nil {
		return err
	}
	return nil
}

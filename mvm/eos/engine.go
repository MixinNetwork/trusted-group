package eos

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"

	"github.com/MixinNetwork/trusted-group/mvm/eos/chain"
	"github.com/MixinNetwork/trusted-group/mvm/eos/secp256k1"
	"github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/crypto"
	// "github.com/uuosio/go-uuoskit/uuoskit"
)

const (
	KEY_NONCE               = 1
	MIXIN_CONTRACT_SEQUENCE = 1
	TX_LOG_ACTION           = "ontxlog"
	ClockTick               = 3 * time.Second
	DEBUG                   = true
)

type Configuration struct {
	Store         string   `toml:"store"`
	RPC           string   `toml:"rpc"`
	PrivateKey    string   `toml:"key"`
	MixinContract string   `toml:"mixin_contract"`
	ChainId       string   `toml:"chain_id"`
	PublicKeys    []string `toml:"public_keys"`
	Publisher     bool     `toml:"publisher"`
}

type Engine struct {
	db            *badger.DB
	rpc           *chain.ChainApi
	mixinContract string
	chainId       *chain.Bytes32
	key           *secp256k1.PrivateKey
	publicKeys    []*secp256k1.PublicKey
	publisher     bool
	threshold     int
}

func Boot(conf *Configuration, threshold int) (*Engine, error) {
	if threshold <= 0 {
		panic(fmt.Errorf("invalid threshold value %d", threshold))
	}

	rpc := chain.NewChainApi(conf.RPC)
	db := openBadger(conf.Store)
	if conf.ChainId == "" {
		panic("chain_id not specified!")
	}
	_chainId, err := chain.NewBytes32FromHex(conf.ChainId)
	if err != nil {
		panic(fmt.Errorf("Invalid chain id: %s", conf.ChainId))
	}

	key, err := secp256k1.NewPrivateKeyFromBase58(conf.PrivateKey)
	if err != nil {
		panic(fmt.Errorf("Invalid private key: %s", conf.PrivateKey))
	}

	pubs := make([]*secp256k1.PublicKey, 0, len(conf.PublicKeys))
	for _, pub := range conf.PublicKeys {
		_pub, err := secp256k1.NewPublicKeyFromBase58(pub)
		if err != nil {
			panic(fmt.Errorf("Invalid public key: %s", pub))
		}
		pubs = append(pubs, _pub)
	}

	if conf.MixinContract == "" {
		panic("mixin contract not specified!")
	}
	logger.Verbosef("++++conf.Publisher: %v", conf.Publisher)
	e := &Engine{
		db:            db,
		rpc:           rpc,
		mixinContract: conf.MixinContract,
		chainId:       _chainId,
		key:           key,
		publicKeys:    pubs,
		publisher:     conf.Publisher,
		threshold:     threshold,
	}

	if e.key != nil {
		chain.GetWallet().Import("test", conf.PrivateKey)
	}
	go e.loopHandleContracts()
	go e.loopContractEvents()
	return e, nil
}

func (e *Engine) Hash(b []byte) []byte {
	return crypto.Keccak256(b)
}

func (e *Engine) SignTx(address string, event *encoding.Event) ([]byte, error) {
	logger.Verbosef("+++++++SignTx chain id: %s", e.chainId.HexString())
	tx, err := BuildEventTransaction(e.mixinContract, address, event)
	if err != nil {
		return nil, err
	}
	signature, err := tx.Sign(e.key, e.chainId)
	if err != nil {
		return nil, err
	}
	return signature.Data[:], nil
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
	notifier := e.key.String()
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

func (e *Engine) VerifyEvent(address string, event *encoding.Event) bool {
	logger.Verbosef("+++++++verifyEvent")
	tx, err := BuildEventTransaction(e.mixinContract, address, event)
	if err != nil {
		return false
	}

	signature := secp256k1.NewSignature(event.Signature)
	digest := tx.Id(e.chainId)
	pub, err := secp256k1.Recover(digest[:], signature)
	if err != nil {
		logger.Verbosef("VerifyEvent: secp256k1.Recover(%v, %v) => %v", digest[:], signature, err)
		return false
	}
	logger.Verbosef("+++++++++Recover public key: %s", pub.StringEOS())
	for _, pk := range e.publicKeys {
		if bytes.Compare(pk.Data[:], pub.Data[:]) == 0 {
			return true
		}
	}
	logger.Verbosef("VerifyEvent return false, %s not found!", pub.StringEOS())
	return false
}

func (e *Engine) VerifyMTGTx(pid string, out *mtg.Output, extra []byte) bool {
	if len(extra) < 24 {
		logger.Verbosef("VerifyMTGTx: invalid reference block")
		return false
	}
	return true
}

func (e *Engine) EstimateCost(events []*encoding.Event) (common.Integer, error) {
	return common.NewInteger(0), nil
}

func (e *Engine) EnsureSendGroupEvents(address string, events []*encoding.Event) error {
	return e.storeWriteGroupEvents(address, events)
}

func (e *Engine) loopContractEvents() {
	for {
		count, err := e.PullContractEvent()
		if err != nil {
			logger.Verbosef("PullContractEvent return error: %v", err)
		}
		if count == 0 {
			time.Sleep(ClockTick)
		}
	}
}

func (e *Engine) PullContractEvent() (int, error) {
	seq, err := e.GetTxRequestSequence()
	if err != nil {
		return 0, err
	}
	offset := e.storeReadContractLogsOffset(e.mixinContract)
	logger.Verbosef("+++++++PullContractEvent seq: %d, offset: %d", seq, offset)

	r, err := e.rpc.GetActions(e.mixinContract, int(offset), 10)
	if err != nil {
		return 0, err
	}

	actions, err := r.GetArray("actions")
	if err != nil {
		return 0, err
	}

	if len(actions) == 0 {
		return 0, nil
	}

	logger.Verbosef("ReceiveGroupEvents offset %d, actions size:%d", offset, len(actions))

	lastIndex := uint64(0)
	count := 0
	for _, action := range actions {
		obj, ok := chain.NewJsonObjectFromInterface(action)
		if !ok {
			continue
		}

		seq, err := obj.GetUint64("account_action_seq")
		if err != nil {
			panic(err)
		}
		lastIndex = seq

		actionObj, err := obj.GetJsonObject("action_trace", "act")
		if err != nil {
			panic(err)
		}

		if DEBUG {
			receiver, err := obj.GetString("action_trace", "receiver")
			if err != nil {
				panic(err)
			}

			if receiver != e.mixinContract {
				panic(fmt.Errorf("receiver not match: expected: %s, got: %s", e.mixinContract, receiver))
			}

			account, err := actionObj.GetString("account")
			if err != nil {
				panic(err)
			}
			if account != e.mixinContract {
				panic("Invalid main account")
			}

			action_name, err := actionObj.GetString("name")
			if err != nil {
				panic(err)
			}
			if action_name != TX_LOG_ACTION {
				panic(fmt.Errorf("Invalid action name, expected: %s, got: %s", TX_LOG_ACTION, action_name))
			}

			actor, err := actionObj.GetString("authorization", 0, "actor")
			if err != nil {
				panic(err)
			}

			if actor != e.mixinContract {
				panic(fmt.Errorf("Invalid permission actor, expected: %s, got: %s", e.mixinContract, actor))
			}

			permission, err := actionObj.GetString("authorization", 0, "permission")
			if err != nil {
				panic(err)
			}
			if permission != "active" {
				panic(fmt.Errorf("Invalid permission, expected: active, got: %s", permission))
			}
		}

		data, err := actionObj.GetString("hex_data")
		if err != nil {
			data, err = actionObj.GetString("data")
			if err != nil {
				panic(err)
			}
		}

		b, err := hex.DecodeString(data)
		if err != nil {
			panic(err)
		}
		txLog := &TxLog{}
		size, err := txLog.Unpack(b)
		if err != nil {
			panic(err)
		}

		if size != len(b) {
			panic(fmt.Errorf("txLog.Unpack: binary size mismatch: %d, got %d", size, len(b)))
		}

		evt := convertTxLogToEvent(txLog)
		// evt, err := encoding.DecodeEvent(b)
		logger.Verbosef("loopGetLogs(%s) => DecodeEvent(%x) => %v, %v", e.mixinContract, b, evt, err)
		if err != nil {
			panic(err)
		}
		err = e.storeWriteContractEvent(txLog.contract.String(), evt)
		if err != nil {
			panic(err)
		}
		count += 1
	}

	//FIXME: consensus on last finished tx request index
	// if e.IsPublisher() && lastIndex != 0 {
	// 	e.clearFinishedTxRequest(e.mixinContract, lastIndex)
	// }
	e.storeWriteContractLogsOffset(e.mixinContract, lastIndex+1)
	return count, nil
}

func (e *Engine) ReceiveGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	return e.storeListContractEvents(address, offset, limit)
}

func (e *Engine) IsPublisher() bool {
	return e.publisher
}

func (e *Engine) GetTxRequestSequence() (uint64, error) {
	key := fmt.Sprintf("%d", MIXIN_CONTRACT_SEQUENCE)
	result, err := e.rpc.GetTableRows(
		false,           //json bool,
		e.mixinContract, //code string,
		e.mixinContract, //scope string,
		"counters",      //table string,
		key,             //lowerbound string,
		key,             //upperbound string,
		10,              //limit int,
		"i64",           //keyType string,
		1,               //indexPosition int
		false,           //reverse bool,
		false,           //showPayer bool,
	)
	if err != nil {
		return 0, err
	}

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
	for e.IsPublisher() {
		time.Sleep(ClockTick)
		nonce, err := e.GetAddressNonce(address)
		if err != nil {
			logger.Verbosef("+++GetAddressNonce(%v) => %v", address, err)
			nonce = 0
		}
		logger.Verbosef("Engine.loopSendGroupEvents, address: %s nonce: %d", address, nonce)

		evts, err := e.storeListGroupEvents(address, nonce, 100)
		if err != nil {
			panic(err)
		}
		for _, evt := range evts {
			err := e.pushEvent(address, evt, true)
			logger.Verbosef("pushEvent => (err: %v)", err)
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

func convertEventToTxEvent(evt *encoding.Event) (*TxEvent, error) {
	process := uuidToBytes(evt.Process)
	asset := uuidToBytes(evt.Asset)

	txEvent := &TxEvent{}

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
		return nil, err
	}
	amount = reverseBytes(amount)
	//FIXME: amount overflow
	copy(txEvent.amount[:], amount)

	txEvent.extra = evt.Extra
	txEvent.timestamp = evt.Timestamp
	txEvent.signature = evt.Signature
	return txEvent, nil
}

func (e *Engine) pushEvent(address string, evt *encoding.Event, good bool) error {
	tx, err := BuildEventTransaction(e.mixinContract, address, evt)
	if err != nil {
		return err
	}

	if len(evt.Signature)/65 < e.threshold {
		return fmt.Errorf("not enough signatures")
	}

	signatures := make([]string, 0, e.threshold)
	for i := 0; i < e.threshold; i += 1 {
		sign := secp256k1.NewSignature(evt.Signature[i*65 : (i+1)*65])
		signatures = append(signatures, sign.String())
	}
	r, err := e.rpc.PushTransaction(tx, signatures, false)
	if err != nil {
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

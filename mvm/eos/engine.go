package eos

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/learnforpractice/goeoslib/chain"
	"github.com/learnforpractice/goeoslib/crypto/secp256k1"
)

const (
	KEY_NONCE               = 1
	MIXIN_CONTRACT_SEQUENCE = 1

	KEY_TX_IN_INDEX = 3

	TX_LOG_ACTION = "ontxlog"
	ClockTick     = 3 * time.Second
	DEBUG         = true
	MAX_ACTIONS   = 100
)

var (
	ErrorNotIrreversible = errors.New("ErrorNotIrreversible")
)

type Configuration struct {
	Store          string   `toml:"store"`
	RPCPush        string   `toml:"rpc-push"`
	RPCGetState    string   `toml:"rpc-get-state"`
	PrivateKey     string   `toml:"key"`
	MixinContract  string   `toml:"mixin-contract"`
	MTGPublisher   string   `toml:"mtg-publisher"`
	MTGExecutor    string   `toml:"mtg-executor"`
	MTGExecutorKey string   `toml:"mtg-executor-key"`
	ChainId        string   `toml:"chain-id"`
	PublicKeys     []string `toml:"public-keys"`
	Publisher      bool     `toml:"publisher"`
	StartBlockNum  uint64   `toml:"start-block-num"`
}

type Engine struct {
	db                   *badger.DB
	chainApiPush         *chain.ChainApi
	chainApiGetState     *chain.ChainApi
	mixinContract        string
	mtgPublisherContract string
	mtgExecutor          string
	mtgExecutorKey       *secp256k1.PrivateKey
	chainId              *chain.Bytes32
	key                  *secp256k1.PrivateKey
	publicKeys           []*secp256k1.PublicKey
	publisher            bool
	threshold            int
	lastChainInfo        *chain.ChainInfo
	mutex                *sync.Mutex
	lastIrrBlockTime     time.Time
	lastIrrBlockId       string
	eventStatus          map[uint64]time.Time
	startBlockNum        uint64
	actionRequestClient  *http.Client
}

type ExtendedAction struct {
	Data string `json:"data"`
}

func Boot(conf *Configuration, threshold int) (*Engine, error) {
	if threshold <= 0 {
		panic(fmt.Errorf("invalid threshold value %d", threshold))
	}

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
	pubKey := key.GetPublicKey()
	pubKeyVerified := false

	if len(conf.PublicKeys) == 0 {
		panic("public-keys not specified!")
	}
	pubs := make([]*secp256k1.PublicKey, 0, len(conf.PublicKeys))
	for _, pub := range conf.PublicKeys {
		_pub, err := secp256k1.NewPublicKeyFromBase58(pub)
		if err != nil {
			panic(fmt.Errorf("Invalid public key: %s", pub))
		}
		pubKeyVerified = pubKeyVerified || (*_pub == *pubKey)
		pubs = append(pubs, _pub)
	}

	if !pubKeyVerified {
		panic("invalid eos.key: public key not found in eos.public-keys")
	}

	if conf.MixinContract == "" {
		panic("mixin-contract not specified!")
	}

	if conf.Publisher {
		if conf.RPCPush == "" {
			panic("rpc-push not specified!")
		}
	}

	if conf.MTGPublisher == "" {
		panic("mtg-publisher not specified!")
	}

	if conf.RPCGetState == "" {
		panic("rpc-get-state not specified!")
	}

	var executorKey *secp256k1.PrivateKey
	if conf.MTGExecutor != "" {
		executorKey, err = secp256k1.NewPrivateKeyFromBase58(conf.MTGExecutorKey)
		if err != nil {
			panic(fmt.Errorf("Invalid mtg-executor-key: %s", conf.MTGExecutorKey))
		}
	}

	logger.Verbosef("++++conf.Publisher: %v", conf.Publisher)

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    15 * time.Second,
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}

	e := &Engine{
		db:                   db,
		chainApiPush:         chain.NewChainApi(conf.RPCPush),
		chainApiGetState:     chain.NewChainApi(conf.RPCGetState),
		mixinContract:        conf.MixinContract,
		mtgPublisherContract: conf.MTGPublisher,
		mtgExecutor:          conf.MTGExecutor,
		mtgExecutorKey:       executorKey,
		chainId:              _chainId,
		key:                  key,
		publicKeys:           pubs,
		publisher:            conf.Publisher,
		threshold:            threshold,
		mutex:                new(sync.Mutex),
		eventStatus:          make(map[uint64]time.Time),
		startBlockNum:        conf.StartBlockNum,
		actionRequestClient:  client,
	}

	if e.key != nil {
		chain.GetWallet().Import("mywallet", conf.PrivateKey)
	}

	e.syncNetwork()
	go e.loopCheckNetworkStatus()
	go e.loopHandleContracts()
	go e.loopContractEvents()
	return e, nil
}

func (e *Engine) Hash(b []byte) []byte {
	return crypto.Keccak256(b)
}

func (e *Engine) SignEvent(address string, event *encoding.Event) []byte {
	if event.Nonce == 0 { //sign addprocess
		addprocess := NewAddProcess(address, event.Process, nil)
		signature, err := addprocess.Sign(e.key)
		if err != nil {
			panic(err)
		}
		return signature.Data[:]
	} else {
		txEvent, err := convertEventToTxEvent(event)
		if err != nil {
			panic(err)
		}
		signature, err := txEvent.Sign(e.key)
		if err != nil {
			panic(err)
		}
		return signature.Data[:]
	}
}

func (e *Engine) VerifyAddress(addr string, extra []byte) error {
	if addr == e.mixinContract {
		return fmt.Errorf("Mixin contract account can not set as Process address!")
	}

	info, err := e.chainApiGetState.GetAccount(addr)
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
	if event.Nonce == 0 {
		addprocess := NewAddProcess(address, event.Process, nil)
		digest := addprocess.Digest()
		for i := 0; i < len(event.Signature)/65; i++ {
			signature := secp256k1.NewSignature(event.Signature[i*65 : (i+1)*65])
			if !e.VerifySignature(digest, signature) {
				return false
			}
		}
		return true
	} else {
		txEvent, err := convertEventToTxEvent(event)
		if err != nil {
			return false
		}
		digest := txEvent.Digest()
		for i := 0; i < len(event.Signature)/65; i++ {
			signature := secp256k1.NewSignature(event.Signature[i*65 : (i+1)*65])
			if !e.VerifySignature(digest, signature) {
				return false
			}
		}
		return true
	}

}

func (e *Engine) VerifySignature(digest *chain.Bytes32, signature *secp256k1.Signature) bool {
	pub, err := secp256k1.Recover(digest[:], signature)
	if err != nil {
		logger.Verbosef("VerifyEvent: secp256k1.Recover(%v, %v) => %v", digest[:], signature, err)
		return false
	}
	for _, pk := range e.publicKeys {
		if bytes.Compare(pk.Data[:], pub.Data[:]) == 0 {
			return true
		}
	}
	return false
}

func (e *Engine) checkNetworkStatus() {
	info, err := e.chainApiGetState.GetInfo()
	if err != nil {
		panic(err)
	}
	e.SetLatestChainInfo(info)

	// t, err := time.Parse("2006-01-02T15:04:05", info.HeadBlockTime)
	// if err != nil {
	// 	panic(err)
	// }

	// t2, err := time.Parse("2006-01-02T15:04:05", info.LastIrreversibleBlockTime)
	// if err != nil {
	// 	panic(err)
	// }

	// libTime := t.Sub(t2)

	// logger.Verbosef("irrerversible block info: %v %v, lib time: %v", info.LastIrreversibleBlockNum, info.LastIrreversibleBlockTime, libTime.String())
}

func (e *Engine) syncNetwork() {
	for {
		info, err := e.chainApiGetState.GetInfo()
		if err != nil {
			panic(err)
		}
		e.SetLatestChainInfo(info)

		t, err := time.Parse("2006-01-02T15:04:05", info.HeadBlockTime)
		if err != nil {
			panic(err)
		}

		if t.Before(time.Now().Add(-time.Second * 30)) {
			logger.Verbosef("Network is not synced, waiting...")
			time.Sleep(time.Second * 10)
			continue
		}
		break
	}
}

func (e *Engine) loopCheckNetworkStatus() {
	for {
		e.checkNetworkStatus()
		time.Sleep(time.Second * 3)
	}
}

func (e *Engine) SetLatestChainInfo(info *chain.ChainInfo) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.lastChainInfo = info
}

func (e *Engine) GetLatestChainInfo() chain.ChainInfo {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return *e.lastChainInfo
}

func (e *Engine) GetRefBlockId() string {
	info := e.GetLatestChainInfo()
	return info.LastIrreversibleBlockID
}

func (e *Engine) getLastIrreversibleBlockNumber() uint32 {
	info, err := e.chainApiGetState.GetInfo()
	if err != nil {
		panic(err)
	}
	return info.LastIrreversibleBlockNum
}

func (e *Engine) EstimateCost(events []*encoding.Event) (common.Integer, error) {
	return common.NewInteger(0), nil
}

func (e *Engine) EnsureSendGroupEvents(address string, events []*encoding.Event) error {
	return e.storeWriteGroupEvents(address, events)
}

func (e *Engine) loopContractEvents() {
	for {
		err := e.PullContractEvents()
		if err != nil {
			if err == ErrorNotIrreversible {
				time.Sleep(time.Second * 3)
			} else {
				logger.Verbosef("PullContractEvents return error: %v", err)
			}
		}
	}
}

func (e *Engine) ParseTxLogFromActionTrace(obj chain.JsonObject) *TxLog {
	data, err := obj.GetString("data")
	if err != nil {
		panic(err)
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
	return txLog
}

func (e *Engine) FetchActions(blockNum uint64) ([]chain.JsonObject, error) {
	actions := make([]chain.JsonObject, 0)
	block, err := e.chainApiGetState.GetBlockTrace(blockNum)
	if err != nil {
		return nil, err
	}

	value, err := block.GetString("status")
	if err != nil {
		return nil, err
	}

	if value != "irreversible" {
		return nil, ErrorNotIrreversible
	}

	txs, err := block.GetArray("transactions")
	for _, _tx := range txs {
		tx, ok := chain.NewJsonObjectFromInterface(_tx)
		if !ok {
			return nil, fmt.Errorf("bad tx object")
		}

		acts, err := tx.GetArray("actions")
		if err != nil {
			return nil, err
		}

		for _, _act := range acts {
			act, ok := chain.NewJsonObjectFromInterface(_act)
			if !ok {
				return nil, fmt.Errorf("bad action object")
			}

			receiver, err := act.GetString("receiver")
			if err != nil {
				return nil, err
			}

			if e.mixinContract != receiver {
				continue
			}

			account, err := act.GetString("account")
			if err != nil {
				return nil, err
			}

			if account != receiver {
				continue
			}

			action, err := act.GetString("action")
			if err != nil {
				return nil, err
			}
			if action != "ontxlog" {
				continue
			}
			actions = append(actions, act)
		}
	}
	return actions, nil
}

func (e *Engine) PullContractEvents() error {
	curBlockNum := e.storeReadCurrentBlockNum()
	if curBlockNum < e.startBlockNum {
		curBlockNum = e.startBlockNum
	}
	if curBlockNum%100 == 0 {
		logger.Verbosef("+++++++++++++current block num %d", curBlockNum)
	}
	actions, err := e.FetchActions(curBlockNum)
	if err != nil {
		return err
	}
	e.storeWriteCurrentBlockNum(curBlockNum + 1)
	if len(actions) == 0 {
		return nil
	}

	e.parseActions(actions)
	return nil
}

func (e *Engine) parseActions(actions []chain.JsonObject) {
	for _, action := range actions {
		txLog := e.ParseTxLogFromActionTrace(action)
		evt := convertTxLogToEvent(txLog)

		err := e.storeWriteContractEvent(txLog.contract.String(), evt)
		if err != nil {
			panic(err)
		}
	}
}

func (e *Engine) ReceiveGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	return e.storeListContractEvents(address, offset, limit)
}

func (e *Engine) IsPublisher() bool {
	return e.publisher
}

func (e *Engine) IsExecutor() bool {
	return e.mtgExecutor != "" && e.mtgExecutorKey != nil
}

func (e *Engine) GetTxRequestsCount() (uint64, error) {
	key := fmt.Sprintf("%d", MIXIN_CONTRACT_SEQUENCE)
	result, err := e.chainApiGetState.GetTableRows(
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
	result, err := e.chainApiGetState.GetTableRows(
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

func (e *Engine) loopExecGroupEvents(address string) {
	if !e.IsExecutor() {
		return
	}

	executor := chain.NewName(e.mtgExecutor)
	counter := uint64(0)
	for {
		tx := chain.NewTransaction(uint32(time.Now().Unix()) + TX_EXPIRATION)

		refBlockId := e.GetRefBlockId()
		tx.SetReferenceBlock(refBlockId)
		action := chain.NewAction(
			&chain.PermissionLevel{Actor: executor, Permission: chain.NewName("active")},
			chain.NewName(address),
			chain.NewName("exec"),
			executor,
			counter,
		)
		tx.Actions = append(tx.Actions, action)
		sign, err := tx.Sign(e.mtgExecutorKey, e.chainId)
		if err != nil {
			panic(err)
		}
		r, err := e.chainApiPush.PushTransaction(tx, []string{sign.String()}, false)
		logger.Verbosef("+++++loopExecGroupEvents(%s): PushTransaction err: %v", address, err)
		if err != nil {
			if r != nil {
				msg, err := r.GetString("error", "details", 0, "message")
				logger.Verbosef("PushTransaction ret: err: %s %v", msg, err)
			} else {
				logger.Verbosef("PushTransaction ret: err: %v", err)
			}
			time.Sleep(time.Second * 5)
		} else {
			console, err := r.GetString("processed", "action_traces", 0, "console")
			if err != nil {
				panic(err)
			}
			logger.Verbosef("++++++execEvent:%s => %s", address, console)
			counter += 1
		}
	}
}

func (e *Engine) getOriginDataByUrl(url string) ([]byte, error) {
	resp, err := e.actionRequestClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	a := ExtendedAction{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		return nil, err
	}

	rawAction, err := hex.DecodeString(a.Data)
	if err != nil {
		return nil, err
	}
	return rawAction, nil
}

func (e *Engine) execPendingEvent(address string, url string, hash []byte) {
	tx := chain.NewTransaction(uint32(time.Now().Unix()) + TX_EXPIRATION)
	originMemo, err := e.getOriginDataByUrl(url)
	if err != nil {
		originMemo = []byte{}
	}

	h := sha256.New()
	h.Write(originMemo)
	digest := h.Sum(nil)
	if bytes.Compare(digest, hash) != 0 {
		logger.Verbosef("+++++invalid original data hash")
		originMemo = []byte{}
	}

	executor := chain.NewName(e.mtgExecutor)
	refBlockId := e.GetRefBlockId()
	tx.SetReferenceBlock(refBlockId)
	action := chain.NewAction(
		&chain.PermissionLevel{Actor: executor, Permission: chain.NewName("active")},
		chain.NewName(address),
		chain.NewName("execpending"),
		executor,
		originMemo,
	)
	tx.Actions = append(tx.Actions, action)
	sign, err := tx.Sign(e.mtgExecutorKey, e.chainId)
	if err != nil {
		panic(err)
	}
	r, err := e.chainApiPush.PushTransaction(tx, []string{sign.String()}, false)
	logger.Verbosef("+++++loopExecPendingEvents(%s): PushTransaction err: %v", address, err)
	if err != nil {
		if r != nil {
			msg, err := r.GetString("error", "details", 0, "message")
			logger.Verbosef("PushTransaction ret: err: %s %v", msg, err)
		} else {
			logger.Verbosef("PushTransaction ret: err: %v", err)
		}
		time.Sleep(time.Second * 5)
	} else {
		console, err := r.GetString("processed", "action_traces", 0, "console")
		if err != nil {
			panic(err)
		}
		logger.Verbosef("++++++execPendingEvent:%s => %s", address, console)
	}
}

func (e *Engine) loopExecPendingEvents(address string) {
	if !e.IsExecutor() {
		return
	}
	executedEvent := make(map[uint64]time.Time)
	for {
		events, err := e.GetPendingEvents(address, 20)
		if err != nil {
			time.Sleep(time.Second * 3)
			continue
		}

		for _, event := range events {
			if executedEvent[event.nonce].Add(TX_EXPIRATION * time.Second).After(time.Now()) {
				continue
			}
			executedEvent[event.nonce] = time.Now()
			if event.extra[0] == 1 && len(event.extra) > 33 {
				hash := event.extra[1:33]
				url := string(event.extra[33:])
				go e.execPendingEvent(address, url, hash)
			} else {
				go e.execPendingEvent(address, "", nil)
			}
		}
	}
}

func (e *Engine) loopDoWorks(address string) {
	if !e.IsExecutor() {
		return
	}

	executor := chain.NewName(e.mtgExecutor)
	counter := uint64(0)
	for {
		time.Sleep(time.Second * 5)
		result, err := e.chainApiGetState.GetTableRows(
			false,   //json bool,
			address, //code string,
			address, //scope string,
			"works", //table string,
			"",      //lowerbound string,
			"",      //upperbound string,
			100,     //limit int,
			"i64",   //keyType string,
			1,       //indexPosition int
			false,   //reverse bool,
			false,   //showPayer bool,
		)
		if err != nil {
			continue
		}

		logger.Verbosef("++++++loopDoWorks: %v", result)
		transfers, err := result.GetArray("rows")
		if err != nil {
			logger.Verbosef("+++++++++err", err)
			return
		}

		for _, transfer := range transfers {
			raw, err := hex.DecodeString(transfer.(string))
			if err != nil {
				logger.Verbosef("+++++++++err", err)
				return
			}
			if len(raw) < 8 {
				logger.Verbosef("+++++++++Invalid data")
				return
			}
			id := binary.LittleEndian.Uint64(raw[:8])
			tx := chain.NewTransaction(uint32(time.Now().Unix()) + TX_EXPIRATION)

			refBlockId := e.GetRefBlockId()
			tx.SetReferenceBlock(refBlockId)
			action := chain.NewAction(
				&chain.PermissionLevel{Actor: executor, Permission: chain.NewName("active")},
				chain.NewName(address),
				chain.NewName("dowork"),
				executor,
				id,
			)
			tx.Actions = append(tx.Actions, action)
			sign, err := tx.Sign(e.mtgExecutorKey, e.chainId)
			if err != nil {
				panic(err)
			}
			r, err := e.chainApiPush.PushTransaction(tx, []string{sign.String()}, false)
			// logger.Verbosef("+++++loopExecGroupEvents: PushTransaction evt: %v, err: %v", r, err)
			if err != nil {
				if r != nil {
					msg, err := r.GetString("error", "details", 0, "message")
					logger.Verbosef("PushTransaction ret: err: %s %v", msg, err)
				} else {
					logger.Verbosef("PushTransaction ret: err: %v", err)
				}
			} else {
				console, err := r.GetString("processed", "action_traces", 0, "console")
				if err != nil {
					panic(err)
				}
				logger.Verbosef("++++++dowork:%s => %s", address, console)
				counter += 1
			}
		}
	}
}

func (e *Engine) GetSubmitedEvent(address string, nonce uint64, limit int) (map[uint64]bool, error) {
	key := fmt.Sprintf("%d", nonce)
	result, err := e.chainApiGetState.GetTableRows(
		false,          //json bool,
		address,        //code string,
		address,        //scope string,
		"submittedevs", //table string,
		key,            //lowerbound string,
		"",             //upperbound string,
		limit,          //limit int,
		"i64",          //keyType string,
		1,              //indexPosition int
		false,          //reverse bool,
		false,          //showPayer bool,
	)
	if err != nil {
		return nil, err
	}

	rows, err := result.GetArray("rows")
	if err != nil {
		return nil, err
	}

	submitedEvent := make(map[uint64]bool)
	for _, row := range rows {
		nonce := row.(string)
		if len(nonce) != 16 {
			return nil, fmt.Errorf("bad nonce value")
		}

		b, err := hex.DecodeString(nonce)
		if err != nil {
			return nil, err
		}
		_nonce := binary.LittleEndian.Uint64(b)
		submitedEvent[_nonce] = true
	}
	return submitedEvent, nil
}

func (e *Engine) GetPendingEvents(address string, limit int) ([]*TxEvent, error) {
	result, err := e.chainApiGetState.GetTableRows(
		false,         //json bool,
		address,       //code string,
		address,       //scope string,
		"pendingevts", //table string,
		"",            //lowerbound string,
		"",            //upperbound string,
		limit,         //limit int,
		"i64",         //keyType string,
		1,             //indexPosition int
		false,         //reverse bool,
		false,         //showPayer bool,
	)
	if err != nil {
		return nil, err
	}

	rows, err := result.GetArray("rows")
	if err != nil {
		return nil, err
	}

	pendingEvents := make([]*TxEvent, 0, len(rows))
	for _, row := range rows {
		rawEvent, err := hex.DecodeString(row.(string))
		if err != nil {
			return nil, err
		}
		event, err := decodeTxEvent(rawEvent)
		if err != nil {
			return nil, err
		}
		pendingEvents = append(pendingEvents, event)
	}
	return pendingEvents, nil
}

func (e *Engine) loopPushGroupEvents(address string) {
	for e.IsPublisher() {
		nonce, err := e.GetAddressNonce(address)
		if err != nil {
			logger.Verbosef("+++GetAddressNonce(%v) => %v", address, err)
			nonce = 0
		}
		evts, err := e.storeListGroupEvents(address, nonce, 100)
		logger.Verbosef("Engine.loopPushGroupEvents, address: %s nonce: %d, len(evts) %d", address, nonce, len(evts))

		if err != nil {
			panic(err)
		}
		submittedEvents, err := e.GetSubmitedEvent(address, nonce, 100)
		logger.Verbosef("++++++++++++++submittedEvents: %v, err: %v", submittedEvents, err)
		sendCount := 0
		for _, evt := range evts {
			if err == nil {
				if _, ok := submittedEvents[evt.Nonce]; ok {
					continue
				}
			}
			if e.eventStatus[evt.Nonce].Add(TX_EXPIRATION * time.Second).Before(time.Now()) {
				e.eventStatus[evt.Nonce] = time.Now()
				err := e.pushEvent(address, evt, true)
				logger.Verbosef("pushEvent(%v, %v) => (err: %v)", address, evt, err)
				sendCount += 1
			}
		}
		if sendCount == 0 {
			time.Sleep(ClockTick)
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
			go e.loopPushGroupEvents(c)
			go e.loopExecGroupEvents(c)
			go e.loopExecPendingEvents(c)
			go e.loopDoWorks(c)
		}
	}
}

func decodeTxEvent(data []byte) (*TxEvent, error) {
	var err error
	dec := chain.NewDecoder(data)
	t := &TxEvent{}
	t.nonce, err = dec.UnpackUint64()
	if err != nil {
		return nil, err
	}

	if _, err := dec.Unpack(&t.process); err != nil {
		return nil, err
	}

	if _, err := dec.Unpack(&t.asset); err != nil {
		return nil, err
	}

	{
		length, err := dec.UnpackLength()
		if err != nil {
			return nil, err
		}
		t.members = make([]chain.Uint128, length)
		for i := 0; i < length; i++ {
			if _, err := dec.Unpack(&t.members[i]); err != nil {
				return nil, err
			}
		}
	}

	t.threshold, err = dec.UnpackInt32()
	if err != nil {
		return nil, err
	}

	if _, err := dec.Unpack(&t.amount); err != nil {
		return nil, err
	}

	t.extra, err = dec.UnpackBytes()
	if err != nil {
		return nil, err
	}

	t.timestamp, err = dec.UnpackUint64()
	if err != nil {
		return nil, err
	}
	return t, nil
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

	signatureCount := len(evt.Signature) / 65
	txEvent.signatures = make([]secp256k1.Signature, signatureCount)
	for i := 0; i < signatureCount; i += 1 {
		copy(txEvent.signatures[i].Data[:], evt.Signature[i*65:i*65+65])
	}
	return txEvent, nil
}

func (e *Engine) pushEvent(address string, evt *encoding.Event, errorEvent bool) error {
	if len(evt.Signature)/65 < e.threshold {
		panic("not enough signatures")
	}
	refBlockId := e.GetRefBlockId()

	tx, err := BuildEventTransaction(e.mixinContract, e.mtgPublisherContract, address, evt, refBlockId)
	if err != nil {
		return err
	}

	signature, err := tx.Sign(e.key, e.chainId)
	if err != nil {
		return err
	}
	signatures := []string{signature.String()}

	r, err := e.chainApiPush.PushTransaction(tx, signatures, false)
	if err != nil {
		if evt.Nonce == 0 {
			return err
		}

		var reason string
		if r != nil {
			reason, err = r.GetString("error", "details", 0, "message")
		} else {
			reason = err.Error()
		}
		logger.Verbosef("error message %v", reason)
		if len(reason) > 256 {
			reason = reason[:256]
		}
		tx, err := BuildErrorEventTransaction(e.mtgPublisherContract, address, evt, refBlockId, reason)
		if err != nil {
			return err
		}

		signature, err := tx.Sign(e.key, e.chainId)
		if err != nil {
			return err
		}
		signatures = []string{signature.String()}
		r, err = e.chainApiPush.PushTransaction(tx, signatures, false)
		if err != nil {
			return err
		}
	}
	console, err := r.GetString("processed", "action_traces", 0, "console")
	if err != nil {
		panic(err)
	}
	logger.Verbosef("++++++pushEvent:%s => %s", address, console)
	return nil
}

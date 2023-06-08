package quorum

import (
	"encoding/hex"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/domains/ethereum"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
)

const (
	ClockTick = 3 * time.Second
	// event MixinTransaction(bytes);
	EventTopic = "0xdb53e751d28ed0d6e3682814bf8d23f7dd7b29c94f74a56fbb7f88e9dca9f39b"
	// function mixin(bytes calldata raw) public returns (bool)
	EventMethod = "0x5cae8005"

	GasLimit = 8000000
	GasPrice = 50000000

	NotifierMinimumBalance = 0.02
	NotifierMaximumBalance = 0.1
)

type Configuration struct {
	Store      string `toml:"store"`
	RPC        string `toml:"rpc"`
	ChainId    int64  `toml:"chain"`
	Base       uint64 `toml:"base"`
	PrivateKey string `toml:"key"`
}

type Engine struct {
	db      *badger.DB
	rpc     *RPC
	chainId int64
	key     string
}

func Build(conf *Configuration) *Engine {
	db := openBadger(conf.Store)
	return &Engine{db: db}
}

func Boot(conf *Configuration) (*Engine, error) {
	db := openBadger(conf.Store)
	rpc, err := NewRPC(conf.RPC, conf.Base)
	if err != nil {
		return nil, err
	}
	e := &Engine{db: db, rpc: rpc, chainId: conf.ChainId}
	if conf.PrivateKey != "" {
		priv, err := crypto.HexToECDSA(conf.PrivateKey)
		if err != nil {
			panic(err)
		}
		e.key = hex.EncodeToString(crypto.FromECDSA(priv))
	}
	go e.loopGetLogs(conf.Base)
	go e.loopHandleContracts()
	return e, nil
}

func (e *Engine) Hash(b []byte) []byte {
	return crypto.Keccak256(b)
}

func (e *Engine) VerifyAddress(address string, _ []byte) error {
	err := ethereum.VerifyAddress(address)
	if err != nil {
		return err
	}

	// TODO ABI
	return nil
}

func (e *Engine) SetupNotifier(address string) error {
	seed := e.Hash([]byte(e.key + address))
	key, err := crypto.ToECDSA(seed)
	if err != nil {
		panic(err)
	}
	notifier := hex.EncodeToString(crypto.FromECDSA(key))
	nonce, err := e.rpc.GetAddressNonce(pub(notifier))
	if err != nil {
		panic(err)
	} else if nonce > 0 {
		logger.Verbosef("Engine.SetupNotifier(%s) => nonce %d", address, nonce)
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
	return false
}

func (e *Engine) EstimateCost(events []*encoding.Event) (common.Integer, error) {
	// TODO should do it
	return common.Zero, nil
}

func (e *Engine) EnsureSendGroupEvents(address string, events []*encoding.Event) error {
	return e.storeWriteGroupEvents(address, events)
}

func (e *Engine) ReceiveGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	return e.storeListContractEvents(address, offset, limit)
}

func (e *Engine) ReadGroupEventTransaction(address string, nonce uint64) (string, error) {
	return e.storeReadGroupEventTransaction(address, nonce)
}

func (e *Engine) IsPublisher() bool {
	return e.key != ""
}

func (e *Engine) loopGetLogs(base uint64) {
	logger.Verbosef("Engine.loopGetLogs(%d)", base)

	for {
		offset := e.storeReadContractLogsOffset()
		if offset < base {
			offset = base
		}
		logs, err := e.rpc.GetLogs(EventTopic, offset, offset+10)
		logger.Verbosef("loopGetLogs(%d) => GetLogs(%d) => %d, %v", base, offset, len(logs), err)
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}
		for _, log := range logs {
			evt, err := encoding.DecodeEvent(log.data)
			logger.Verbosef("loopGetLogs(%s) => DecodeEvent(%x) => %v, %v", log.address, log.data, evt, err)
			if err != nil {
				continue
			}
			err = e.storeWriteContractEvent(log.address, evt)
			if err != nil {
				panic(err)
			}
		}
		height, err := e.rpc.GetBlockHeight()
		if err != nil || offset+10 > height {
			time.Sleep(ClockTick)
			continue
		}
		err = e.storeWriteContractLogsOffset(offset + 10)
		if err != nil {
			panic(err)
		}
	}
}

func (e *Engine) loopSendGroupEvents(address string) {
	logger.Verbosef("Engine.loopSendGroupEvents(%s)", address)
	notifier := e.storeReadContractNotifier(address)

	for e.IsPublisher() {
		balance, err := e.rpc.GetAddressBalance(pub(notifier))
		if err != nil {
			logger.Verbosef("loopSendGroupEvents(%s) => GetAddressBalance(%s) => %v", address, pub(notifier), err)
			time.Sleep(5 * time.Second)
			continue
		}
		if balance.Cmp(decimal.NewFromFloat(NotifierMinimumBalance/2)) < 0 {
			time.Sleep(5 * time.Second)
			continue
		}
		nonce, err := e.rpc.GetAddressNonce(pub(notifier))
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		evts, err := e.storeListGroupEvents(address, nonce, 100)
		if err != nil {
			panic(err)
		}
		for _, evt := range evts {
			id, raw := e.signGroupEventTransaction(address, evt, notifier)
			// TODO should have a thread to index all mixin calls on address
			err := e.storeWriteGroupEventTransaction(address, evt.Nonce, id)
			if err != nil {
				panic(err)
			}
			res, err := e.rpc.SendRawTransaction(raw)
			logger.Verbosef("loopSendGroupEvents(%s) => SendRawTransaction(%s, %s) => %s, %v", address, id, raw, res, err)
		}
		time.Sleep(ClockTick)
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
			go e.loopSendGroupEvents(c)
		}
		if !e.IsPublisher() {
			continue
		}

		nonce, err := e.rpc.GetAddressNonce(pub(e.key))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}
		for _, c := range all {
			notifier := e.storeReadContractNotifier(c)
			balance, err := e.rpc.GetAddressBalance(pub(notifier))
			if err != nil {
				break
			}
			if balance.Cmp(decimal.NewFromFloat(NotifierMinimumBalance)) > 0 {
				continue
			}
			id, raw := e.signContractNotifierDepositTransaction(pub(notifier), e.key, decimal.NewFromFloat(NotifierMaximumBalance), nonce)
			res, err := e.rpc.SendRawTransaction(raw)
			logger.Verbosef("loopHandleContracts => SendRawTransaction(%s, %s) => %s, %v", id, raw, res, err)
			nonce = nonce + 1
		}
	}
}

func pub(priv string) string {
	key, _ := crypto.HexToECDSA(priv)
	return crypto.PubkeyToAddress(key.PublicKey).Hex()
}

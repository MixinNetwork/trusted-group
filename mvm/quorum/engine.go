package quorum

import (
	"encoding/hex"
	"fmt"
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

	ContractAgeLimit = 16
	GasLimit         = 1000000
	GasPrice         = 100000000000
)

type Configuration struct {
	Store      string `toml:"store"`
	RPC        string `toml:"rpc"`
	ChainId    int64  `toml:"chain"`
	PrivateKey string `toml:"key"`
}

type Engine struct {
	db      *badger.DB
	rpc     *RPC
	chainId int64
	key     string
}

func Boot(conf *Configuration) (*Engine, error) {
	db := openBadger(conf.Store)
	rpc, err := NewRPC(conf.RPC)
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
	go e.loopHandleContracts()
	return e, nil
}

func (e *Engine) Hash(b []byte) []byte {
	return crypto.Keccak256(b)
}

func (e *Engine) VerifyAddress(address string, hash []byte) error {
	err := ethereum.VerifyAddress(address)
	if err != nil {
		return err
	}
	height, err := e.rpc.GetBlockHeight()
	if err != nil {
		panic(err)
	}
	birth, err := e.rpc.GetContractBirthBlock(address, string(hash))
	if err != nil {
		return err
	}
	if height < birth+ContractAgeLimit {
		return fmt.Errorf("too young %d %d", birth, height)
	}
	// TODO ABI
	return e.storeWriteContractLogsOffset(address, birth)
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
		return fmt.Errorf("notifier used %d", nonce)
	}
	old := e.storeReadContractNotifier(address)
	if old == notifier {
		return nil
	} else if old != "" {
		panic(old)
	}
	return e.storeWriteContractNotifier(address, notifier)
}

func (e *Engine) EstimateCost(events []*encoding.Event) (common.Integer, error) {
	// TODO should do it
	return common.Zero, nil
}

func (e *Engine) EnsureSendGroupEvents(address string, events []*encoding.Event) error {
	return e.storeWriteGroupEvents(address, events)
}

func (e *Engine) ReceiveGroupEvents(block uint64) ([]*encoding.Event, error) {
	height, err := e.rpc.GetBlockHeight()
	if err != nil {
		return nil, err
	}
	if block > height {
		return nil, fmt.Errorf("block in the future %d %d", block, height)
	}
	logs, err := e.rpc.GetLogs(EventTopic, block, block)
	if err != nil {
		return nil, err
	}
	var events []*encoding.Event
	for _, b := range logs {
		evt, err := encoding.DecodeEvent(b)
		logger.Verbosef("ReceiveGroupEvents(%d) => DecodeEvent(%x) => %v, %v", block, b, evt, err)
		if err != nil {
			continue
		}
		events = append(events, evt)
	}
	return events, nil
}

func (e *Engine) IsPublisher() bool {
	return e.key != ""
}

func (e *Engine) loopSendGroupEvents(address string) {
	logger.Verbosef("Engine.loopSendGroupEvents(%s)", address)
	notifier := e.storeReadContractNotifier(address)

	for e.IsPublisher() {
		balance, err := e.rpc.GetAddressBalance(pub(notifier))
		if err != nil {
			panic(err)
		}
		if balance.Cmp(decimal.NewFromInt(1)) < 0 {
			time.Sleep(5 * time.Second)
			continue
		}
		nonce, err := e.rpc.GetAddressNonce(pub(notifier))
		if err != nil {
			panic(err)
		}
		evts, err := e.storeListGroupEvents(address, nonce, 100)
		if err != nil {
			panic(err)
		}
		for _, evt := range evts {
			id, raw := e.signGroupEventTransaction(address, evt, notifier)
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
			panic(err)
		}
		for _, c := range all {
			notifier := e.storeReadContractNotifier(c)
			balance, err := e.rpc.GetAddressBalance(pub(notifier))
			if err != nil {
				panic(err)
			}
			if balance.Cmp(decimal.NewFromInt(10)) > 0 {
				continue
			}
			id, raw := e.signContractNotifierDepositTransaction(pub(notifier), e.key, decimal.NewFromInt(100), nonce)
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

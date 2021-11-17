package quorum

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/domains/ethereum"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
)

const (
	ClockTick  = 3 * time.Second
	EventTopic = "0x90b180992b9780963499859256fa3d63e86d4e68dc14d9c6ae2bf38a0953a8a1"
)

type Configuration struct {
	Store      string `toml:"store"`
	RPC        string `toml:"rpc"`
	PrivateKey string `toml:"key"`
}

type Engine struct {
	db        *badger.DB
	rpc       *RPC
	key       string
	publisher bool
}

func Boot(conf *Configuration) (*Engine, error) {
	return nil, nil
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
	if err != nil && strings.Contains(err.Error(), "malformed") {
		return err
	} else if err != nil {
		panic(err)
	}
	if height < birth+128 {
		return fmt.Errorf("too young %d %d", birth, height)
	}
	// TODO ABI
	e.storeWriteContractLogsOffset(address, birth)
	return nil
}

func (e *Engine) SetupNotifier(address string) error {
	// seed = hash(e.key + address)
	// key from seed
	// read contract notifier state
	key := ""
	return e.storeWriteContractNotifier(address, key, "initial")
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

func (e *Engine) loopGetLogs(address string) {
	nonce := e.storeReadLastContractEventNonce(address) + 1
	for {
		offset := e.storeReadContractLogsOffset(address)
		logs, err := e.rpc.GetLogs(address, EventTopic, offset, offset+10)
		if err != nil {
			panic(err)
		}
		var evts []*encoding.Event
		for _, b := range logs {
			evt, err := encoding.DecodeEvent(b)
			if err != nil {
				panic(err)
			}
			evts = append(evts, evt)
		}
		sort.Slice(evts, func(i, j int) bool { return evts[i].Nonce < evts[j].Nonce })
		for _, evt := range evts {
			if evt.Nonce < nonce {
				continue
			}
			if evt.Nonce > nonce {
				break
			}
			e.storeWriteContractEvent(address, evt)
			nonce = nonce + 1
		}
		e.storeWriteContractLogsOffset(address, offset+10)
		if len(logs) == 0 {
			time.Sleep(ClockTick * 5)
		}
	}
}

func (e *Engine) loopSendGroupEvents(address string) {
	// for loop all group events
	// ensure the events are in RPC
	// batch events per transaction
	// there should be only one node engines send transactions
	// check events available before sending the transaction
	for e.publisher {
		time.Sleep(ClockTick)
	}
}

func (e *Engine) loopHandleContracts() {
	for {
		// read all contracts
		// see if notifier setup => setup
		// see if they are running
		// then loopGetLogs
		// then loopSendGroupEvents
	}
}

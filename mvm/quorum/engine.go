package quorum

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type Engine struct {
}

func Boot() (*Engine, error) {
	return nil, nil
}

func (e *Engine) VerifyAddress(address string) error {
	// format
	// ABI
	panic(0)
}

func (e *Engine) EstimateCost(events []*encoding.Event) (common.Integer, error) {
	panic(0)
}

func (e *Engine) EnsureSendGroupEvents(address string, events []*encoding.Event) error {
	// local store write group events
	panic(0)
}

func (e *Engine) ReceiveGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	// rpc.eth_getLogs(fromBlock: offset, toBlock: offset+10, address: address, topics: groupTransfer)
	panic(0)
}

func (e *Engine) loopSendEvents() {
	// for loop all group events
	// ensure the events are in RPC
}

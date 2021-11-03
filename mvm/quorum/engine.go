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

func (e *Engine) EstimateCost(events []*encoding.Event) (common.Integer, error) {
	panic(0)
}

func (e *Engine) SendGroupEvents(address string, events []*encoding.Event) error {
	panic(0)
}

func (e *Engine) ReceiveGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	panic(0)
}

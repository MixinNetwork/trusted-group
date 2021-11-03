package machine

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type Store interface {
	WriteGroupEvent(event *encoding.Event) error
	ListGroupEvents(id string, limit int) ([]*encoding.Event, error)
	ExpireGroupEvents(events []*encoding.Event) error

	ReadAccount(id string, asset string) (*Account, error)
	WriteAccountChange(id string, asset string, amount common.Integer, credit bool) error

	ReadEngineGroupEventsOffset(id string) (uint64, error)
	WriteEngineGroupEventsOffset(id string, offset uint64) error

	ListProcesses() ([]*Process, error)
	WriteProcess(p *Process) error
}

type Engine interface {
	EstimateCost(events []*encoding.Event) (common.Integer, error)
	SendGroupEvents(address string, events []*encoding.Event) error
	ReceiveGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error)
}

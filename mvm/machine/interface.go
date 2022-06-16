package machine

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type Store interface {
	CheckPendingGroupEventIdentifier(id string) (bool, error)
	WritePendingGroupEventAndNonce(event *encoding.Event, id string) error
	ListPendingGroupEvents(limit int) ([]*encoding.Event, error)
	ReadGroupEventSignatures(pid string, nonce uint64) ([][]byte, bool, error)
	WritePendingGroupEventSignatures(pid string, nonce uint64, partials [][]byte) error
	WriteSignedGroupEventAndExpirePending(event *encoding.Event) error
	ListSignedGroupEvents(pid string, limit int) ([]*encoding.Event, error)
	ExpireGroupEventsWithCost(events []*encoding.Event, cost common.Integer) error

	CheckAccountSnapshot(as *AccountSnapshot) (bool, error)
	WriteAccountSnapshot(as *AccountSnapshot) error

	ReadEngineGroupEventsOffset(pid string) (uint64, error)
	WriteEngineGroupEventsOffset(pid string, offset uint64) error

	ListProcesses() ([]*Process, error)
	WriteProcess(p *Process) error

	WriteAsset(a *Asset) error
	ReadAsset(id string) (*Asset, error)
}

type Engine interface {
	VerifyAddress(addr string, extra []byte) error
	SetupNotifier(addr string) error
	VerifyEvent(address string, event *encoding.Event) bool
	EstimateCost(events []*encoding.Event) (common.Integer, error)
	EnsureSendGroupEvents(address string, events []*encoding.Event) error
	ReceiveGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error)
	SignEvent(address string, event *encoding.Event) []byte
}

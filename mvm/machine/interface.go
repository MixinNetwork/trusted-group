package machine

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type Store interface {
	CheckPendingGroupEventIdentifier(id string) (bool, error)
	WritePendingGroupEventAndNonce(event *encoding.Event, id string) error
	ListPendingGroupEvents(limit int) ([]*encoding.Event, error)
	ReadPendingGroupEventSignatures(pid string, nonce uint64) ([][]byte, error)
	WritePendingGroupEventSignatures(pid string, nonce uint64, partials [][]byte) error
	WriteSignedGroupEventAndExpirePending(event *encoding.Event) error
	ListSignedGroupEvents(pid string, limit int) ([]*encoding.Event, error)
	ExpireGroupEventsWithCost(events []*encoding.Event, cost common.Integer) error

	CheckAccountSnapshot(as *AccountSnapshot) (bool, error)
	WriteAccountSnapshot(as *AccountSnapshot) error

	ReadEngineGroupEventsOffset() (uint64, error)
	WriteEngineGroupEventsOffset(offset uint64) error

	ListProcesses() ([]*Process, error)
	WriteProcess(p *Process) error
}

type Engine interface {
	Hash(b []byte) []byte
	VerifyAddress(addr string, extra []byte) error
	SetupNotifier(addr string) error
	AddProcess(id, address string) error
	EstimateCost(events []*encoding.Event) (common.Integer, error)
	EnsureSendGroupEvents(address string, events []*encoding.Event) error
	ReceiveGroupEvents(offset uint64) ([]*encoding.Event, error)
}

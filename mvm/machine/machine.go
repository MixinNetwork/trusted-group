package machine

import (
	"sync"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/shopspring/decimal"
	"golang.org/x/net/context"
)

const (
	ProcessRegistrationAssetId = "c94ac88f-4671-3976-b60a-09064f1811e8"
)

type Machine struct {
	Store     Store
	engines   map[string]Engine
	processes map[string]*Process
	mutex     *sync.Mutex
}

func Boot() (*Machine, error) {
	return nil, nil
}

func (m *Machine) Loop(ctx context.Context) {
	processes, err := m.Store.ListProcesses()
	if err != nil {
		panic(err)
	}
	for _, p := range processes {
		m.processes[p.Identifier] = p
		p.Spawn(ctx, m.Store)
	}
}

func (m *Machine) AddEngine(platform string, engine Engine) {
	switch platform {
	case ProcessPlatformQuorum:
	default:
		return
	}
	m.engines[platform] = engine
}

func (m *Machine) AddProcess(pid string, platform, address string, out *mtg.Output) {
	if out.AssetID != ProcessRegistrationAssetId {
		return
	}
	if out.Amount.Cmp(decimal.NewFromInt(1)) < 0 {
		return
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	engine := m.engines[platform]
	if engine == nil {
		return
	}
	for _, old := range m.processes {
		if old.Identifier == pid {
			return
		}
		if old.Address == address {
			return
		}
	}

	err := engine.VerifyAddress(address)
	if err != nil {
		logger.Verbosef("VerifyAddress(%s) => %s", address, err)
	}
	proc := &Process{
		Identifier: pid,
		Platform:   platform,
		Address:    address,
		Credit:     common.Zero,
		Nonce:      0,
	}
	err = m.Store.WriteProcess(proc)
	if err != nil {
		panic(err)
	}
	m.processes[pid] = proc
}

func (m *Machine) WriteGroupEvent(pid string, out *mtg.Output, extra []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	proc := m.processes[pid]
	if proc == nil {
		return
	}
	amount := common.NewIntegerFromString(out.Amount.String())
	evt := &encoding.Event{
		Asset:     out.AssetID,
		Members:   []string{out.Sender},
		Threshold: 1,
		Amount:    amount,
		Memo:      extra,
		Nonce:     proc.Nonce,
	}
	err := m.Store.WriteGroupEventAndNonce(pid, evt)
	if err != nil {
		panic(err)
	}
	proc.Nonce = proc.Nonce + 1
}

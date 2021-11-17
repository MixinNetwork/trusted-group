package machine

import (
	"context"
	"sync"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/tip/messenger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/drand/kyber"
	"github.com/drand/kyber/share"
	"github.com/shopspring/decimal"
)

const (
	ProcessRegistrationAssetId = "c94ac88f-4671-3976-b60a-09064f1811e8"
)

type Configuration struct {
	Poly  string `toml:"poly"`
	Share string `toml:"share"`
}

type Machine struct {
	Store       Store
	share       *share.PriShare
	commitments []kyber.Point
	group       *mtg.Group
	messenger   messenger.Messenger
	engines     map[string]Engine
	processes   map[string]*Process
	mutex       *sync.Mutex
}

func Boot(group *mtg.Group, store Store, m messenger.Messenger) (*Machine, error) {
	return &Machine{
		Store:     store,
		group:     group,
		messenger: m,
		engines:   make(map[string]Engine),
		processes: make(map[string]*Process),
		mutex:     new(sync.Mutex),
	}, nil
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
	go m.loopReceiveGroupMessages(ctx)
	m.loopSignGroupEvents(ctx)
}

func (m *Machine) AddEngine(platform string, engine Engine) {
	switch platform {
	case ProcessPlatformQuorum:
	default:
		return
	}
	m.engines[platform] = engine
}

func (m *Machine) AddProcess(ctx context.Context, pid string, platform, address string, out *mtg.Output) {
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
	err = engine.SetupNotifier(address)
	if err != nil {
		logger.Verbosef("SetupNotifier(%s) => %s", address, err)
	}
	proc := &Process{
		Identifier: pid,
		Platform:   platform,
		Address:    address,
		Credit:     common.Zero,
		Nonce:      0,
		machine:    m,
	}
	err = m.Store.WriteProcess(proc)
	if err != nil {
		panic(err)
	}
	m.processes[pid] = proc
	proc.Spawn(ctx, m.Store)
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
		Process:   proc.Identifier,
		Asset:     out.AssetID,
		Members:   []string{out.Sender},
		Threshold: 1,
		Amount:    amount,
		Memo:      extra,
		Timestamp: uint64(out.CreatedAt.UnixNano()),
		Nonce:     proc.Nonce,
	}
	err := m.Store.WritePendingGroupEventAndNonce(evt)
	if err != nil {
		panic(err)
	}
	proc.Nonce = proc.Nonce + 1
}

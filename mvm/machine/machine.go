package machine

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"sync"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/tip/crypto"
	"github.com/MixinNetwork/tip/messenger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/drand/kyber"
	"github.com/drand/kyber/group/mod"
	"github.com/drand/kyber/pairing/bn256"
	"github.com/drand/kyber/share"
	"github.com/shopspring/decimal"
)

const (
	ProcessRegistrationAssetId = "965e5c6e-434c-3fa9-b780-c50f43cd955c"
)

type Configuration struct {
	Poly  string `toml:"poly"`
	Share string `toml:"share"`
}

type Machine struct {
	store     Store
	group     *mtg.Group
	share     *share.PriShare
	poly      *share.PubPoly
	messenger messenger.Messenger
	platform  string
	engine    Engine
	processes map[string]*Process
	mutex     *sync.Mutex
}

func Boot(conf *Configuration, group *mtg.Group, store Store, m messenger.Messenger) (*Machine, error) {
	pb, err := hex.DecodeString(conf.Poly)
	if err != nil {
		return nil, err
	}
	commitments := unmarshalCommitments(pb)
	suite := bn256.NewSuiteG2()
	poly := share.NewPubPoly(suite, suite.Point().Base(), commitments)
	sb, err := hex.DecodeString(conf.Share)
	if err != nil {
		return nil, err
	}
	share := unmarshalPrivShare(sb)
	logger.Printf("Machine.Boot(%s)", poly.Commit().String())
	return &Machine{
		store:     store,
		group:     group,
		share:     share,
		poly:      poly,
		messenger: m,
		processes: make(map[string]*Process),
		mutex:     new(sync.Mutex),
	}, nil
}

func (m *Machine) Loop(ctx context.Context) {
	go m.loopReceiveEvents(ctx)
	processes, err := m.store.ListProcesses()
	if err != nil {
		panic(err)
	}
	for _, p := range processes {
		m.processes[p.Identifier] = p
		m.Spawn(ctx, p)
	}
	go m.loopReceiveGroupMessages(ctx)
	m.loopSignGroupEvents(ctx)
}

func (m *Machine) SetEngine(platform string, engine Engine) {
	switch platform {
	case ProcessPlatformQuorum:
	case ProcessPlatformEos:
	default:
		return
	}

	m.platform = platform
	m.engine = engine
}

func (m *Machine) AddProcess(ctx context.Context, pid string, platform, address string, out *mtg.Output, extra []byte) {
	if pid != out.Sender {
		logger.Verbosef("AddProcess(%s, %s, %s) => sender %s", pid, platform, address, out.Sender)
		return
	}
	if out.AssetID != ProcessRegistrationAssetId {
		logger.Verbosef("AddProcess(%s, %s, %s) => asset %s", pid, platform, address, out.AssetID)
		return
	}
	if out.Amount.Cmp(decimal.NewFromInt(1)) < 0 {
		logger.Verbosef("AddProcess(%s, %s, %s) => amount %s", pid, platform, address, out.Amount)
		return
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.platform != platform {
		logger.Verbosef("AddProcess(%s, %s, %s) => engine %s", pid, platform, address, m.platform)
		return
	}
	for _, old := range m.processes {
		if old.Identifier == out.Sender {
			logger.Verbosef("AddProcess(%s, %s, %s) => sender %s", pid, platform, address, out.Sender)
			return
		}
		if old.Address == address {
			logger.Verbosef("AddProcess(%s, %s, %s) => address %s", pid, platform, address, address)
			return
		}
	}

	err := m.engine.VerifyAddress(address, extra)
	if err != nil {
		logger.Verbosef("VerifyAddress(%s) => %s", address, err)
		return
	}
	err = m.engine.SetupNotifier(address)
	if err != nil {
		logger.Verbosef("SetupNotifier(%s) => %s", address, err)
		return
	}
	m.engine.AddProcess(out.Sender, address)
	proc := &Process{
		Identifier: out.Sender,
		Address:    address,
		Credit:     common.Zero,
		Nonce:      0,
	}
	err = m.store.WriteProcess(proc)
	if err != nil {
		panic(err)
	}
	m.processes[proc.Identifier] = proc
	m.Spawn(ctx, proc)
}

func (m *Machine) WriteGroupEvent(pid string, out *mtg.Output, extra []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	proc := m.processes[pid]
	if proc == nil {
		return
	}
	done, err := m.store.CheckPendingGroupEventIdentifier(out.UTXOID)
	if err != nil {
		panic(err)
	} else if done {
		return
	}

	amount := common.NewIntegerFromString(out.Amount.String())
	evt := &encoding.Event{
		Process:   proc.Identifier,
		Asset:     out.AssetID,
		Members:   []string{out.Sender},
		Threshold: 1,
		Amount:    amount,
		Extra:     extra,
		Timestamp: uint64(out.CreatedAt.UnixNano()),
		Nonce:     proc.Nonce,
	}
	as := buildAccountSnapshot(evt, true)
	err = m.store.WriteAccountSnapshot(as)
	if err != nil {
		panic(err)
	}
	err = m.store.WritePendingGroupEventAndNonce(evt, out.UTXOID)
	if err != nil {
		panic(err)
	}
	proc.Nonce = proc.Nonce + 1
}

func unmarshalPrivShare(b []byte) *share.PriShare {
	var ps share.PriShare
	ps.V = mod.NewInt64(0, bn256.Order).SetBytes(b[4:])
	ps.I = int(binary.BigEndian.Uint32(b[:4]))
	return &ps
}

func unmarshalCommitments(b []byte) []kyber.Point {
	var commits []kyber.Point
	for i, l := 0, len(b)/128; i < l; i++ {
		point, err := crypto.PubKeyFromBytes(b[i*128 : (i+1)*128])
		if err != nil {
			panic(err)
		}
		commits = append(commits, point)
	}
	return commits
}

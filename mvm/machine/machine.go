package machine

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/tip/crypto"
	"github.com/MixinNetwork/tip/crypto/en256"
	"github.com/MixinNetwork/tip/messenger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/drand/kyber"
	"github.com/drand/kyber/group/mod"
	"github.com/drand/kyber/share"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

type Configuration struct {
	Poly             string `toml:"poly"`
	Share            string `toml:"share"`
	ProcessFeeAsset  string `toml:"process-fee-asset"`
	ProcessFeeAmount string `toml:"process-fee-amount"`
}

type Machine struct {
	store      Store
	mixin      *mixin.Client
	group      *mtg.Group
	share      *share.PriShare
	poly       *share.PubPoly
	feeAssetId string
	feeAmount  decimal.Decimal
	messenger  messenger.Messenger
	engines    map[string]Engine
	processes  map[string]*Process
	procLock   *sync.RWMutex
	signerLock *sync.Mutex
}

func Boot(conf *Configuration, group *mtg.Group, store Store, m messenger.Messenger, mixin *mixin.Client) (*Machine, error) {
	pb, err := hex.DecodeString(conf.Poly)
	if err != nil {
		return nil, err
	}
	feeAmount, _ := decimal.NewFromString(conf.ProcessFeeAmount)
	if feeAmount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid process fee amount %s", conf.ProcessFeeAmount)
	}
	commitments := unmarshalCommitments(pb)
	suite := en256.NewSuiteG2()
	poly := share.NewPubPoly(suite, suite.Point().Base(), commitments)
	sb, err := hex.DecodeString(conf.Share)
	if err != nil {
		return nil, err
	}
	share := unmarshalPrivShare(sb)
	logger.Printf("Machine.Boot(%s)", poly.Commit().String())

	if !poly.Check(share) {
		panic("invalid machine.share: poly check failed")
	}

	return &Machine{
		store:      store,
		mixin:      mixin,
		group:      group,
		share:      share,
		poly:       poly,
		feeAssetId: conf.ProcessFeeAsset,
		feeAmount:  feeAmount,
		messenger:  m,
		engines:    make(map[string]Engine),
		processes:  make(map[string]*Process),
		procLock:   new(sync.RWMutex),
		signerLock: new(sync.Mutex),
	}, nil
}

func (m *Machine) Loop(ctx context.Context) {
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

func (m *Machine) AddEngine(platform string, engine Engine) {
	switch platform {
	case ProcessPlatformQuorum:
	default:
		return
	}
	m.engines[platform] = engine
}

func (m *Machine) AddProcess(ctx context.Context, pid string, platform, address string, out *mtg.Output, extra []byte) bool {
	if pid != out.Sender {
		logger.Verbosef("AddProcess(%s, %s, %s) => sender %s", pid, platform, address, out.Sender)
		return false
	}
	if out.AssetID != m.feeAssetId {
		logger.Verbosef("AddProcess(%s, %s, %s) => asset %s", pid, platform, address, out.AssetID)
		return false
	}
	if out.Amount.Cmp(m.feeAmount) < 0 {
		logger.Verbosef("AddProcess(%s, %s, %s) => amount %s", pid, platform, address, out.Amount)
		return false
	}
	m.procLock.Lock()
	defer m.procLock.Unlock()

	engine := m.engines[platform]
	if engine == nil {
		logger.Verbosef("AddProcess(%s, %s, %s) => engine %s", pid, platform, address, platform)
		return false
	}
	for _, old := range m.processes {
		if old.Identifier == out.Sender {
			logger.Verbosef("AddProcess(%s, %s, %s) => sender %s", pid, platform, address, out.Sender)
			return false
		}
		if old.Address == address {
			logger.Verbosef("AddProcess(%s, %s, %s) => address %s", pid, platform, address, address)
			return false
		}
	}

	err := engine.VerifyAddress(address, extra)
	if err != nil {
		logger.Verbosef("VerifyAddress(%s) => %s", address, err)
		return false
	}
	err = engine.SetupNotifier(address)
	if err != nil {
		logger.Verbosef("SetupNotifier(%s) => %s", address, err)
		return false
	}
	proc := &Process{
		Identifier: out.Sender,
		Platform:   platform,
		Address:    address,
		Credit:     common.Zero,
		Nonce:      0,
	}
	proc.Asset = strings.Contains(string(extra), "META")
	err = m.store.WriteProcess(proc)
	if err != nil {
		panic(err)
	}
	m.processes[proc.Identifier] = proc
	m.Spawn(ctx, proc)

	return true
}

func (m *Machine) WriteGroupEvent(ctx context.Context, pid string, out *mtg.Output, extra []byte) {
	m.procLock.RLock()
	defer m.procLock.RUnlock()

	proc := m.processes[pid]
	if proc == nil {
		return
	}
	if proc.Asset {
		meta, err := m.fetchAssetMeta(ctx, out.AssetID)
		if err != nil {
			panic(err)
		}
		extra = append(meta, extra...)
	}
	if len(extra) > encoding.EventExtraMaxSize {
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
	as := proc.buildAccountSnapshot(evt, true)
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

func OutputGrouper(out *mtg.Output) string {
	op, err := parseOperation(out.Memo)
	if err != nil {
		return ""
	}
	return op.Process
}

func unmarshalPrivShare(b []byte) *share.PriShare {
	var ps share.PriShare
	ps.V = mod.NewInt64(0, en256.Order).SetBytes(b[4:])
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

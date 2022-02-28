package machine

import (
	"bytes"
	"context"
	"encoding/binary"
	"sort"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/tip/crypto"
	"github.com/MixinNetwork/tip/crypto/en256"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/drand/kyber/sign/tbls"
)

const (
	SignTypeTBLS      = 1
	SignTypeSECP256K1 = 2
)

func (m *Machine) getProcess(pid string) *Process {
	m.procLock.RLock()
	defer m.procLock.RUnlock()

	return m.processes[pid]
}

func (m *Machine) loopSignGroupEvents(ctx context.Context) {
	sm := make(map[string]time.Time)
	for {
		time.Sleep(3 * time.Second)
		events, err := m.store.ListPendingGroupEvents(100)
		if err != nil {
			panic(err)
		}

		for _, e := range events {
			if e.Process == "b2a47a3a-99ff-33a8-8c7b-d7fae9821509" { // FIXME remove this hack
				e.Signature = make([]byte, 64)
				m.writeSignedGroupEventAndExpirePending(e, SignTypeTBLS)
				continue
			}
			lst := sm[e.ID()].Add(time.Minute * 5)
			if lst.After(time.Now()) {
				continue
			}
			sm[e.ID()] = time.Now()
			logger.Verbosef("Machine.loopSignGroupEvents() => %d, %v", e.Nonce, e)

			if e.Signature != nil {
				panic(e)
			}
			msg := e.Encode()
			process := m.getProcess(e.Process)
			if process.Platform == ProcessPlatformEOS {
				e.Signature = m.engines[ProcessPlatformEOS].SignEvent(process.Address, e)
			} else {
				scheme := tbls.NewThresholdSchemeOnG1(en256.NewSuiteG2())
				partial, err := scheme.Sign(m.share, msg)
				if err != nil {
					panic(err)
				}
				e.Signature = partial
			}

			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			err = m.queueMessage(ctx, append(e.Encode(), threshold...))
			if err != nil {
				panic(err)
			}
			err = m.appendPendingGroupEventSignature(e, msg, e.Signature, process.SignType())
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *Machine) loopReceiveGroupMessages(ctx context.Context) {
	sm := make(map[string]time.Time)
	for {
		peer, b, err := m.messenger.ReceiveMessage(ctx)
		if err != nil {
			logger.Verbosef("Machine.ReceiveMessage() => %s", err)
			panic(err)
		}
		evt, err := encoding.DecodeEvent(b[:len(b)-8])
		if err != nil {
			logger.Verbosef("DecodeEvent(%x) => %s", b, err)
			continue
		}
		process := m.getProcess(evt.Process)
		if process == nil {
			logger.Verbosef("getProcess(%s) => %v", evt.Process, evt)
			continue
		}
		if process.Platform == ProcessPlatformEOS {
			m.handleEOSGroupMessages(ctx, process.Address, evt, sm)
			continue
		}

		sig := evt.Signature
		evt.Signature = nil
		msg := evt.Encode()

		partials, fullSignature, err := m.store.ReadGroupEventSignatures(evt.Process, evt.Nonce, SignTypeTBLS)
		logger.Verbosef("ReadGroupEventSignatures(%s, %d) => %v %v %v", evt.Process, evt.Nonce, partials, fullSignature, err)
		if err != nil {
			panic(err)
		}

		switch true {
		case len(sig) == 64:
			err = crypto.Verify(m.poly.Commit(), msg, sig)
			if err != nil && evt.Timestamp > 1638789832002675803 { // FIXME remove this timestamp check
				logger.Verbosef("crypto.Verify(%x, %x) => %v %v", msg, sig, evt, err)
				continue
			}
			evt.Signature = sig
			logger.Verbosef("loopReceiveGroupMessages(%x) => WriteSignedGroupEventAndExpirePending(%v)", b, evt)
			err = m.writeSignedGroupEventAndExpirePending(evt, SignTypeTBLS)
			if err != nil {
				panic(err)
			}
		case fullSignature:
			if sm[evt.ID()].Add(time.Minute * 5).After(time.Now()) {
				continue
			}
			evt.Signature = partials[0]
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			m.messenger.QueueMessage(ctx, peer, append(evt.Encode(), threshold...))
			sm[evt.ID()] = time.Now()
		default:
			// FIXME ensure valid partial signature
			err = m.appendPendingGroupEventSignature(evt, msg, sig, SignTypeTBLS)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *Machine) appendPendingGroupEventSignature(e *encoding.Event, msg, partial []byte, signType int) error {
	m.signerLock.Lock()
	defer m.signerLock.Unlock()

	partials, fullSignature, err := m.store.ReadGroupEventSignatures(e.Process, e.Nonce, signType)
	if err != nil {
		return err
	}
	if fullSignature {
		return nil
	}

	if checkSignedWith(partials, partial) {
		return nil
	}
	partials = append(partials, partial)

	if len(partials) < m.group.GetThreshold() {
		return m.store.WritePendingGroupEventSignatures(e.Process, e.Nonce, partials, signType)
	}

	if signType == SignTypeTBLS {
		e.Signature = m.recoverSignature(msg, partials)
		logger.Verbosef("loopSignGroupEvents() => WriteSignedGroupEventAndExpirePending(%v) recover", e)
		return m.store.WriteSignedGroupEventAndExpirePending(e, SignTypeTBLS)
	} else {
		return m.eosWriteFullSignatures(e, partials)
	}
}

func (m *Machine) writeSignedGroupEventAndExpirePending(e *encoding.Event, signType int) error {
	m.signerLock.Lock()
	defer m.signerLock.Unlock()

	return m.store.WriteSignedGroupEventAndExpirePending(e, signType)
}

func (m *Machine) recoverSignature(msg []byte, partials [][]byte) []byte {
	scheme := tbls.NewThresholdSchemeOnG1(en256.NewSuiteG2())
	sig, err := scheme.Recover(m.poly, msg, partials, m.group.GetThreshold(), len(m.group.GetMembers()))
	if err != nil {
		panic(err)
	}
	err = crypto.Verify(m.poly.Commit(), msg, sig)
	if err != nil {
		panic(err)
	}
	return sig
}

func checkSignedWith(partials [][]byte, s []byte) bool {
	for _, p := range partials {
		if bytes.Compare(p, s) == 0 {
			return true
		}
	}
	return false
}

func (m *Machine) handleEOSGroupMessages(ctx context.Context, address string, evt *encoding.Event, sm map[string]time.Time) {
	if len(evt.Signature) == 0 || len(evt.Signature)%65 != 0 {
		logger.Verbosef("++++handleEOSGroupMessages: invalid signature length: %d", len(evt.Signature))
		return
	}

	if !m.engines[ProcessPlatformEOS].VerifyEvent(address, evt) {
		logger.Verbosef("VerifyEvent(%v, %v) return false", address, evt)
		return
	}

	_, fullSignature, err := m.store.ReadGroupEventSignatures(evt.Process, evt.Nonce, SignTypeSECP256K1)
	if err != nil {
		panic(err)
	}

	lst, ok := sm[evt.ID()]
	if !ok {
		sm[evt.ID()] = time.Now()
	} else {
		if fullSignature && lst.Add(time.Minute*5).Before(time.Now()) {
			partial := m.engines[ProcessPlatformEOS].SignEvent(address, evt)
			evt.Signature = partial
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			m.messenger.BroadcastMessage(ctx, append(evt.Encode(), threshold...))
			sm[evt.ID()] = time.Now()
		}
	}

	if fullSignature {
		return
	}
	sig := evt.Signature
	evt.Signature = nil
	m.appendPendingGroupEventSignature(evt, nil, sig, SignTypeSECP256K1)
}

func (m *Machine) eosWriteFullSignatures(e *encoding.Event, partials [][]byte) error {
	e.Signature = nil
	sort.Slice(partials, func(i, j int) bool {
		return bytes.Compare(partials[i], partials[j]) < 0
	})
	for _, partial := range partials {
		e.Signature = append(e.Signature, partial...)
	}
	e.Signature = append(e.Signature, byte(len(partials)))
	return m.store.WriteSignedGroupEventAndExpirePending(e, SignTypeSECP256K1)
}

func (m *Machine) queueMessage(ctx context.Context, b []byte) error {
	for _, p := range m.group.GetMembers() {
		err := m.messenger.QueueMessage(ctx, p, b)
		if err != nil {
			return err
		}
	}
	return nil
}

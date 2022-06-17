package machine

import (
	"bytes"
	"context"
	"encoding/binary"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/tip/crypto"
	"github.com/MixinNetwork/tip/crypto/en256"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/drand/kyber/sign/tbls"
)

const (
	messagePeriod = time.Hour
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
			lst := sm[e.ID()].Add(messagePeriod)
			if lst.After(time.Now()) {
				continue
			}
			sm[e.ID()] = time.Now()
			logger.Verbosef("Machine.loopSignGroupEvents() => %d, %v", e.Nonce, e)

			if e.Signature != nil {
				panic(e)
			}
			msg := e.Encode()
			scheme := tbls.NewThresholdSchemeOnG1(en256.NewSuiteG2())
			partial, err := scheme.Sign(m.share, msg)
			if err != nil {
				panic(err)
			}
			e.Signature = partial

			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			err = m.queueMessage(ctx, append(e.Encode(), threshold...))
			if err != nil {
				panic(err)
			}
			err = m.appendPendingGroupEventSignature(e, msg, e.Signature)
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
		if len(evt.Extra) > encoding.EventExtraMaxSize {
			logger.Verbosef("DecodeEvent(%x) => %d", b, len(evt.Extra))
			continue
		}
		process := m.getProcess(evt.Process)
		if process == nil {
			logger.Verbosef("getProcess(%s) => %v", evt.Process, evt)
			continue
		}

		sig := evt.Signature
		evt.Signature = nil
		msg := evt.Encode()

		partials, fullSignature, err := m.store.ReadGroupEventSignatures(evt.Process, evt.Nonce)
		logger.Verbosef("ReadGroupEventSignatures(%s, %d) => %v %v %v", evt.Process, evt.Nonce, partials, fullSignature, err)
		if err != nil {
			panic(err)
		}

		switch true {
		case len(sig) == 64:
			err = crypto.Verify(m.poly.Commit(), msg, sig)
			if err != nil {
				logger.Verbosef("crypto.Verify(%x, %x) => %v %v", msg, sig, evt, err)
				continue
			}
			evt.Signature = sig
			logger.Verbosef("loopReceiveGroupMessages(%x) => WriteSignedGroupEventAndExpirePending(%v)", b, evt)
			err = m.writeSignedGroupEventAndExpirePending(evt)
			if err != nil {
				panic(err)
			}
		case fullSignature:
			if sm[evt.ID()].Add(messagePeriod).After(time.Now()) {
				continue
			}
			evt.Signature = partials[0]
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			m.messenger.QueueMessage(ctx, peer, append(evt.Encode(), threshold...))
			sm[evt.ID()] = time.Now()
		default:
			// FIXME ensure valid partial signature
			err = m.appendPendingGroupEventSignature(evt, msg, sig)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *Machine) appendPendingGroupEventSignature(e *encoding.Event, msg, partial []byte) error {
	m.signerLock.Lock()
	defer m.signerLock.Unlock()

	partials, fullSignature, err := m.store.ReadGroupEventSignatures(e.Process, e.Nonce)
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
		return m.store.WritePendingGroupEventSignatures(e.Process, e.Nonce, partials)
	}

	e.Signature = m.recoverSignature(msg, partials)
	logger.Verbosef("loopSignGroupEvents() => WriteSignedGroupEventAndExpirePending(%v) recover", e)
	return m.store.WriteSignedGroupEventAndExpirePending(e)
}

func (m *Machine) writeSignedGroupEventAndExpirePending(e *encoding.Event) error {
	m.signerLock.Lock()
	defer m.signerLock.Unlock()

	return m.store.WriteSignedGroupEventAndExpirePending(e)
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

func (m *Machine) queueMessage(ctx context.Context, b []byte) error {
	for _, p := range m.group.GetMembers() {
		err := m.messenger.QueueMessage(ctx, p, b)
		if err != nil {
			return err
		}
	}
	return nil
}

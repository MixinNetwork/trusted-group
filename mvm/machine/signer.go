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

func (m *Machine) loopSignGroupEvents(ctx context.Context) {
	sm := make(map[string]time.Time)
	for {
		time.Sleep(3 * time.Second)
		events, err := m.store.ListPendingGroupEvents(100)
		if err != nil {
			panic(err)
		}
		for _, e := range events {
			lst := sm[e.ID()].Add(time.Minute * 5)
			if lst.After(time.Now()) {
				continue
			}
			sm[e.ID()] = time.Now()
			logger.Verbosef("Machine.loopSignGroupEvents() => %v", e)

			e.Signature = nil
			msg := e.Encode()
			scheme := tbls.NewThresholdSchemeOnG1(en256.NewSuiteG2())
			partial, err := scheme.Sign(m.share, msg)
			if err != nil {
				panic(err)
			}

			e.Signature = partial
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			err = m.messenger.SendMessage(ctx, append(e.Encode(), threshold...))
			if err != nil {
				panic(err)
			}

			err = m.appendPendingGroupEventSignature(e, msg, partial)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *Machine) loopReceiveGroupMessages(ctx context.Context) {
	sm := make(map[string]time.Time)
	for {
		_, b, err := m.messenger.ReceiveMessage(ctx)
		if err != nil {
			logger.Verbosef("Machine.ReceiveMessage() => %s", err)
			panic(err)
		}
		evt, err := encoding.DecodeEvent(b[:len(b)-8])
		if err != nil {
			logger.Verbosef("DecodeEvent(%x) => %s", b, err)
			continue
		}
		sig := evt.Signature
		evt.Signature = nil
		msg := evt.Encode()

		partials, err := m.store.ReadPendingGroupEventSignatures(evt.Process, evt.Nonce)
		if err != nil {
			panic(err)
		}

		switch true {
		case len(sig) == 64:
			err = crypto.Verify(m.poly.Commit(), msg, sig)
			if err != nil && evt.Timestamp > 1638789832002675803 { // FIXME remove this timestamp check
				logger.Verbosef("crypto.Verify(%x, %x) => %v", msg, sig, err)
				continue
			}
			evt.Signature = sig
			logger.Verbosef("loopReceiveGroupMessages(%x) => WriteSignedGroupEventAndExpirePending(%v)", b, evt)
			err = m.writeSignedGroupEventAndExpirePending(evt)
			if err != nil {
				panic(err)
			}
		case checkFullSignature(partials):
			if sm[evt.ID()].Add(time.Minute * 5).After(time.Now()) {
				continue
			}
			evt.Signature = partials[0]
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			m.messenger.SendMessage(ctx, append(evt.Encode(), threshold...))
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

	partials, err := m.store.ReadPendingGroupEventSignatures(e.Process, e.Nonce)
	if err != nil {
		return err
	}
	if checkFullSignature(partials) {
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

func checkFullSignature(partials [][]byte) bool {
	return len(partials) == 1 && len(partials[0]) == 64
}

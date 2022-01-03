package machine

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sort"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/tip/crypto"
	"github.com/MixinNetwork/tip/crypto/en256"
	"github.com/MixinNetwork/trusted-group/mvm/constants"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/drand/kyber/sign/tbls"
)

func (m *Machine) getProcessInfo(processId string) (string, string, bool) {
	m.procLock.Lock()
	defer m.procLock.Unlock()
	process, ok := m.processes[processId]
	if !ok {
		return "", "", false
	}
	return process.Platform, process.Address, true
}

func (m *Machine) GetLastSendTime(id string) time.Time {
	m.smLock.Lock()
	defer m.smLock.Unlock()
	return m.sm[id]
}

func (m *Machine) SetLastSendTime(id string, t time.Time) {
	m.smLock.Lock()
	defer m.smLock.Unlock()
	m.sm[id] = t
}

func (m *Machine) loopSignGroupEvents(ctx context.Context) {
	for {
		time.Sleep(3 * time.Second)
		events, err := m.store.ListPendingGroupEvents(100)
		if err != nil {
			panic(err)
		}

		for _, e := range events {
			lst := m.GetLastSendTime(e.ID()).Add(time.Minute * 5)
			if lst.After(time.Now()) {
				continue
			}
			m.SetLastSendTime(e.ID(), time.Now())
			logger.Verbosef("Machine.loopSignGroupEvents() => %d, %v", e.Nonce, e)

			var partial []byte
			msg := e.Encode()
			platform, address, ok := m.getProcessInfo(e.Process)
			if !ok {
				panic(fmt.Errorf("unknown process %s", e.Process))
			}
			if platform == ProcessPlatformEos {
				partial = m.engines[ProcessPlatformEos].SignEvent(address, e)
				e.Signature = partial
			} else {
				e.Signature = nil
				scheme := tbls.NewThresholdSchemeOnG1(en256.NewSuiteG2())
				partial, err := scheme.Sign(m.share, msg)
				if err != nil {
					panic(err)
				}
				e.Signature = partial
			}

			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			err = m.messenger.SendMessage(ctx, append(e.Encode(), threshold...))
			if err != nil {
				panic(err)
			}
			if platform == ProcessPlatformEos {
				err = m.appendPendingGroupEventSignature(e, msg, partial, constants.SignTypeSECP256K1)
			} else {
				err = m.appendPendingGroupEventSignature(e, msg, partial, constants.SignTypeTBLS)
			}
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *Machine) loopReceiveGroupMessages(ctx context.Context) {
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
		platform, address, ok := m.getProcessInfo(evt.Process)
		if !ok {
			continue
		}
		if platform == ProcessPlatformEos {
			m.handleEosGroupMessages(ctx, address, evt)
			continue
		}

		sig := evt.Signature
		evt.Signature = nil
		msg := evt.Encode()

		partials, fullSignature, err := m.store.ReadPendingGroupEventSignatures(evt.Process, evt.Nonce, constants.SignTypeTBLS)
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
			err = m.writeSignedGroupEventAndExpirePending(evt, constants.SignTypeTBLS)
			if err != nil {
				panic(err)
			}
		case fullSignature:
			if m.GetLastSendTime(evt.ID()).Add(time.Minute * 5).After(time.Now()) {
				continue
			}
			evt.Signature = partials[0]
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			m.messenger.SendMessage(ctx, append(evt.Encode(), threshold...))
			m.SetLastSendTime(evt.ID(), time.Now())
		default:
			// FIXME ensure valid partial signature
			err = m.appendPendingGroupEventSignature(evt, msg, sig, constants.SignTypeTBLS)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *Machine) appendPendingGroupEventSignature(e *encoding.Event, msg, partial []byte, signType int) error {
	m.signerLock.Lock()
	defer m.signerLock.Unlock()

	partials, fullSignature, err := m.store.ReadPendingGroupEventSignatures(e.Process, e.Nonce, signType)
	if err != nil {
		return err
	}
	logger.Verbosef("+++++++nonce: %d, len(partials): %d, fullSignature: %v", e.Nonce, len(partials), fullSignature)
	if fullSignature {

		return nil
	}

	if checkSignedWith(partials, partial) {
		return nil
	}
	partials = append(partials, partial)
	logger.Verbosef("+++++++partials: %v, threshold: %d", len(partials), m.group.GetThreshold())

	if len(partials) < m.group.GetThreshold() {
		return m.store.WritePendingGroupEventSignatures(e.Process, e.Nonce, partials, signType)
	}

	if signType == constants.SignTypeTBLS {
		e.Signature = m.recoverSignature(msg, partials)
		logger.Verbosef("loopSignGroupEvents() => WriteSignedGroupEventAndExpirePending(%v) recover", e)
		return m.store.WriteSignedGroupEventAndExpirePending(e, constants.SignTypeTBLS)
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

func checkFullSignature(partials [][]byte) bool {
	return len(partials) == 1 && len(partials[0]) == 64
}

func (m *Machine) handleEosGroupMessages(ctx context.Context, address string, evt *encoding.Event) {
	if len(evt.Signature) == 0 || len(evt.Signature)%65 != 0 {
		logger.Verbosef("++++handleEosGroupMessages: invalid signature length: %d", len(evt.Signature))
		return
	}

	if !m.engines[ProcessPlatformEos].VerifyEvent(address, evt) {
		logger.Verbosef("VerifyEvent(%v, %v) return false", address, evt)
		return
	}

	_, fullSignature, err := m.store.ReadPendingGroupEventSignatures(evt.Process, evt.Nonce, constants.SignTypeSECP256K1)
	if err != nil {
		panic(err)
	}

	lst := m.GetLastSendTime(evt.ID())
	if lst.Add(time.Minute * 5).Before(time.Now()) {
		partial := m.engines[ProcessPlatformEos].SignEvent(address, evt)
		evt.Signature = partial
		threshold := make([]byte, 8)
		binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
		m.messenger.SendMessage(ctx, append(evt.Encode(), threshold...))
		m.SetLastSendTime(evt.ID(), time.Now())
	}

	if fullSignature {
		return
	}
	sig := evt.Signature
	evt.Signature = nil
	m.appendPendingGroupEventSignature(evt, nil, sig, constants.SignTypeSECP256K1)
}

func (m *Machine) eosWriteFullSignatures(e *encoding.Event, partials [][]byte) error {
	e.Signature = nil
	sortBytesArray(partials)
	for _, partial := range partials {
		e.Signature = append(e.Signature, partial...)
	}
	e.Signature = append(e.Signature, byte(len(partials)))
	return m.store.WriteSignedGroupEventAndExpirePending(e, constants.SignTypeSECP256K1)
}

func sortBytesArray(arr [][]byte) {
	sort.Slice(arr, func(i, j int) bool {
		return bytes.Compare(arr[i], arr[j]) < 0
	})
}

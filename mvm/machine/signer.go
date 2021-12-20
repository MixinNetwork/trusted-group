package machine

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
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

func (m *Machine) loopSignGroupEvents(ctx context.Context) {
	sm := make(map[string]time.Time)
	for {
		time.Sleep(3 * time.Second)
		events, err := m.store.ListPendingGroupEvents(100)
		if err != nil {
			panic(err)
		}

		for _, e := range events {
			proc := m.processes[e.Process]
			if proc == nil {
				panic(fmt.Errorf("unknown process %v", e.Process))
			}
			if proc.Platform == ProcessPlatformEos {
				m.signEosEvents(ctx, proc, e, sm)
				continue
			}
			e.Signature = nil
			logger.Verbosef("Machine.loopSignGroupEvents() => %v", e)
			msg := e.Encode()
			partials, fullSignature, err := m.store.ReadPendingGroupEventSignatures(e.Process, e.Nonce, constants.SignTypeTBLS)
			if err != nil {
				panic(err)
			}

			if fullSignature {
				e.Signature = partials[0]
				logger.Verbosef("loopSignGroupEvents() => WriteSignedGroupEventAndExpirePending(%v) full", e)
				err = m.store.WriteSignedGroupEventAndExpirePending(e, constants.SignTypeTBLS)
				if err != nil {
					panic(err)
				}
				continue
			}
			if len(partials) >= m.group.GetThreshold() {
				e.Signature = m.recoverSignature(msg, partials)
				logger.Verbosef("loopSignGroupEvents() => WriteSignedGroupEventAndExpirePending(%v) recover", e)
				err = m.store.WriteSignedGroupEventAndExpirePending(e, constants.SignTypeTBLS)
				if err != nil {
					panic(err)
				}
				continue
			}

			scheme := tbls.NewThresholdSchemeOnG1(en256.NewSuiteG2())
			partial, err := scheme.Sign(m.share, msg)
			if err != nil {
				panic(err)
			}
			lst := sm[hex.EncodeToString(partial)].Add(time.Minute * 5)
			if checkSignedWith(partials, partial) && lst.After(time.Now()) {
				continue
			}
			sm[hex.EncodeToString(partial)] = time.Now()

			e.Signature = partial
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			err = m.messenger.SendMessage(ctx, append(e.Encode(), threshold...))
			if err != nil {
				panic(err)
			}

			if checkSignedWith(partials, partial) {
				continue
			}
			partials = append(partials, partial)
			err = m.store.WritePendingGroupEventSignatures(e.Process, e.Nonce, partials, constants.SignTypeTBLS)
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
		proc := m.processes[evt.Process]
		if proc == nil {
			logger.Verbosef("unknown process %v", evt.Process)
			continue
		}

		if proc.Platform == ProcessPlatformEos {
			m.handleEosGroupMessages(ctx, proc, evt, sm)
			continue
		}

		if len(evt.Signature) == 64 {
			sig := evt.Signature
			evt.Signature = nil
			msg := evt.Encode()
			if evt.Timestamp > 1638789832002675803 { // FIXME remove this timestamp check
				err = crypto.Verify(m.poly.Commit(), msg, sig)
				if err != nil {
					logger.Verbosef("crypto.Verify(%x, %x) => %v", msg, sig, err)
					continue
				}
			}
			evt.Signature = sig
			logger.Verbosef("loopReceiveGroupMessages(%x) => WriteSignedGroupEventAndExpirePending(%v)", b, evt)
			err = m.store.WriteSignedGroupEventAndExpirePending(evt, constants.SignTypeTBLS)
			if err != nil {
				panic(err)
			}
			continue
		}

		partials, fullSignature, err := m.store.ReadPendingGroupEventSignatures(evt.Process, evt.Nonce, constants.SignTypeTBLS)
		if err != nil {
			panic(err)
		}
		if fullSignature {
			if sm[evt.ID()].Add(time.Minute * 5).Before(time.Now()) {
				evt.Signature = partials[0]
				threshold := make([]byte, 8)
				binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
				m.messenger.SendMessage(ctx, append(evt.Encode(), threshold...))
				sm[evt.ID()] = time.Now()
			}
			continue
		}
		if checkSignedWith(partials, evt.Signature) {
			continue
		}
		partials = append(partials, evt.Signature) // FIXME ensure valid partial signature
		err = m.store.WritePendingGroupEventSignatures(evt.Process, evt.Nonce, partials, constants.SignTypeTBLS)
		if err != nil {
			panic(err)
		}
	}
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

func (m *Machine) signEosEvents(ctx context.Context, proc *Process, e *encoding.Event, sm map[string]time.Time) {
	platform := proc.Platform
	if platform != ProcessPlatformEos {
		panic("Invalid platform")
	}

	e.Signature = nil
	partials, fullSignature, err := m.store.ReadPendingGroupEventSignatures(e.Process, e.Nonce, constants.SignTypeSECP256K1)
	if err != nil {
		panic(err)
	}

	if fullSignature || len(partials) >= m.group.GetThreshold() {
		signature := e.Signature
		e.Signature = nil
		sortBytesArray(partials)
		for _, partial := range partials {
			e.Signature = append(e.Signature, partial...)
		}
		e.Signature = append(e.Signature, byte(len(partials)))
		err = m.store.WriteSignedGroupEventAndExpirePending(e, constants.SignTypeSECP256K1)
		if err != nil {
			panic(err)
		}
		e.Signature = signature
		return
	}

	address := proc.Address
	partial, err := m.engines[ProcessPlatformEos].SignTx(address, e)
	if err != nil {
		logger.Verbosef("++SignTx return error: %v", err)
		return
	}

	key := hex.EncodeToString(partial)
	lst := sm[key].Add(time.Minute * 5)
	if lst.Before(time.Now()) {
		sm[key] = time.Now()
		e.Signature = partial
		threshold := make([]byte, 8)
		binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
		err = m.messenger.SendMessage(ctx, append(e.Encode(), threshold...))
		if err != nil {
			panic(err)
		}
	}

	if !checkSignedWith(partials, partial) {
		partials = append(partials, partial)
		sortBytesArray(partials)
	}

	err = m.store.WritePendingGroupEventSignatures(e.Process, e.Nonce, partials, constants.SignTypeSECP256K1)
	if err != nil {
		panic(err)
	}
}

func (m *Machine) handleEosGroupMessages(ctx context.Context, proc *Process, evt *encoding.Event, sm map[string]time.Time) {
	if len(evt.Signature) == 0 || len(evt.Signature)%65 != 0 {
		logger.Verbosef("++++handleEosGroupMessages: invalid signature length: %d", len(evt.Signature))
		return
	}

	address := proc.Address
	if !m.engines[proc.Platform].VerifyEvent(address, evt) {
		logger.Verbosef("VerifyEvent(%v, %v) return false", address, evt)
		return
	}

	partials, fullSignature, err := m.store.ReadPendingGroupEventSignatures(evt.Process, evt.Nonce, constants.SignTypeSECP256K1)
	if err != nil {
		panic(err)
	}
	if fullSignature {
		if sm[evt.ID()].Add(time.Minute * 5).Before(time.Now()) {
			sm[evt.ID()] = time.Now()
			partial, err := m.engines[ProcessPlatformEos].SignTx(address, evt)
			if err != nil {
				logger.Verbosef("++SignTx return error: %v", err)
				return
			}
			evt.Signature = partial
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			m.messenger.SendMessage(ctx, append(evt.Encode(), threshold...))
		}
		return
	}

	signatureMap := make(map[string][]byte)
	signatures := make([][]byte, 0, len(evt.Signature)/65)
	for i := 0; i < len(evt.Signature); i += 65 {
		signature := evt.Signature[i : i+65]
		if !checkSignedWith(partials, signature) {
			signatures = append(signatures, signature)
		}
		signatureMap[hex.EncodeToString(signature)] = signature
	}
	if len(signatureMap) != len(evt.Signature)/65 {
		// duplicate signatures
		return
	}
	if len(signatures) == 0 {
		return
	}
	for _, signature := range signatures {
		partials = append(partials, signature)
	}
	sortBytesArray(partials)
	err = m.store.WritePendingGroupEventSignatures(evt.Process, evt.Nonce, partials, constants.SignTypeSECP256K1)
	if err != nil {
		panic(err)
	}
}

func sortBytesArray(arr [][]byte) {
	sort.Slice(arr, func(i, j int) bool {
		return bytes.Compare(arr[i], arr[j]) < 0
	})
}

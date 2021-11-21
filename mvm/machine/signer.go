package machine

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/tip/crypto"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/drand/kyber/pairing/bn256"
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
			e.Signature = nil
			logger.Verbosef("loopSignGroupEvents => %v", e)
			msg := m.engines[ProcessPlatformQuorum].Hash(e.Encode()) // FIXME
			partials, err := m.store.ReadPendingGroupEventSignatures(e.Process, e.Nonce)
			if err != nil {
				panic(err)
			}

			if len(partials) >= m.group.GetThreshold() {
				e.Signature = m.recoverSignature(msg, partials)
				err = m.store.WriteSignedGroupEventAndExpirePending(e)
				if err != nil {
					panic(err)
				}
				continue
			}

			scheme := tbls.NewThresholdSchemeOnG1(bn256.NewSuiteG2())
			partial, err := scheme.Sign(m.share, msg)
			if err != nil {
				panic(err)
			}
			lst := sm[hex.EncodeToString(partial)].Add(time.Minute * 60)
			if checkSignedWith(partials, partial) && lst.After(time.Now()) {
				continue
			}
			sm[hex.EncodeToString(partial)] = time.Now()

			e.Signature = partial
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(lst.UnixNano()))
			err = m.messenger.SendMessage(ctx, append(e.Encode(), threshold...))
			if err != nil {
				panic(err)
			}

			if checkSignedWith(partials, partial) {
				continue
			}
			partials = append(partials, partial)
			err = m.store.WritePendingGroupEventSignatures(e.Process, e.Nonce, partials)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (m *Machine) loopReceiveGroupMessages(ctx context.Context) {
	sm := make(map[string]time.Time)
	for {
		b, err := m.messenger.ReceiveMessage(ctx)
		if err != nil {
			logger.Verbosef("ReceiveMessage() => %s", err)
			panic(err)
		}
		evt, err := encoding.DecodeEvent(b[:len(b)-8])
		if err != nil {
			logger.Verbosef("DecodeEvent(%s) => %s", hex.EncodeToString(b), err)
			continue
		}
		if len(evt.Signature) == 64 {
			sig := evt.Signature
			evt.Signature = nil
			err = crypto.Verify(m.poly.Commit(), evt.Encode(), sig)
			if err != nil {
				continue
			}
			evt.Signature = sig
			err = m.store.WriteSignedGroupEventAndExpirePending(evt)
			if err != nil {
				panic(err)
			}
			continue
		}

		partials, err := m.store.ReadPendingGroupEventSignatures(evt.Process, evt.Nonce)
		if err != nil {
			panic(err)
		}
		if len(partials) >= m.group.GetThreshold() && sm[evt.ID()].Add(time.Minute*60).Before(time.Now()) {
			evt.Signature = nil
			evt.Signature = m.recoverSignature(evt.Encode(), partials)
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(time.Now().UnixNano()))
			m.messenger.SendMessage(ctx, append(evt.Encode(), threshold...))
			sm[evt.ID()] = time.Now()
			continue
		}
		if checkSignedWith(partials, evt.Signature) {
			continue
		}
		partials = append(partials, evt.Signature)
		err = m.store.WritePendingGroupEventSignatures(evt.Process, evt.Nonce, partials)
		if err != nil {
			panic(err)
		}
	}
}

func (m *Machine) recoverSignature(msg []byte, partials [][]byte) []byte {
	scheme := tbls.NewThresholdSchemeOnG1(bn256.NewSuiteG2())
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

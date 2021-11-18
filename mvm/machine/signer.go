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
	for {
		time.Sleep(3 * time.Second)
		events, err := m.store.ListPendingGroupEvents(100)
		if err != nil {
			panic(err)
		}
		for _, e := range events {
			e.Signature = nil
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
			if checkSignedWith(partials, partial) {
				continue
			}

			e.Signature = partial
			now := time.Now().Unix() / 60
			threshold := make([]byte, 8)
			binary.BigEndian.PutUint64(threshold, uint64(now))
			err = m.messenger.SendMessage(ctx, append(e.Encode(), threshold...))
			if err != nil {
				panic(err)
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
		partials, err := m.store.ReadPendingGroupEventSignatures(evt.Process, evt.Nonce)
		if err != nil {
			panic(err)
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

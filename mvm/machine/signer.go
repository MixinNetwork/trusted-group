package machine

import (
	"context"

	"github.com/MixinNetwork/tip/crypto"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/drand/kyber/pairing/bn256"
	"github.com/drand/kyber/share"
	"github.com/drand/kyber/sign/tbls"
)

func (m *Machine) loopSignGroupEvents(ctx context.Context) {
	for {
		events, err := m.Store.ListPendingGroupEvents(100)
		if err != nil {
			panic(err)
		}
		for _, e := range events {
			partials, err := m.Store.ReadPendingGroupEventSignatures(e.Process, e.Nonce)
			if err != nil {
				panic(err)
			}
			if len(partials) >= m.group.GetThreshold() {
				e.Signature = nil
				suite := bn256.NewSuiteG2()
				scheme := tbls.NewThresholdSchemeOnG1(bn256.NewSuiteG2())
				poly := share.NewPubPoly(suite, suite.Point().Base(), m.commitments)
				sig, err := scheme.Recover(poly, e.Encode(), partials, len(m.commitments), len(partials))
				if err != nil {
					panic(err)
				}
				err = crypto.Verify(poly.Commit(), e.Encode(), sig)
				if err != nil {
					panic(err)
				}
				e.Signature = sig
				err = m.Store.WriteSignedGroupEvent(e)
				if err != nil {
					panic(err)
				}
			}
			if len(partials) > 0 {
				continue
			}
			scheme := tbls.NewThresholdSchemeOnG1(bn256.NewSuiteG2())
			partial, err := scheme.Sign(m.share, e.Encode())
			if err != nil {
				panic(err)
			}
			e.Signature = partial
			err = m.messenger.SendMessage(ctx, e.Encode())
			if err != nil {
				panic(err)
			}
			err = m.Store.WritePendingGroupEventSignatures(e.Process, e.Nonce, [][]byte{partial})
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
			panic(err)
		}
		evt, err := encoding.DecodeEvent(b)
		if err != nil {
			panic(err)
		}
		// TODO validate evt and partial
		partials, err := m.Store.ReadPendingGroupEventSignatures(evt.Process, evt.Nonce)
		if err != nil {
			panic(err)
		}
		partials = append(partials, evt.Signature)
		err = m.Store.WritePendingGroupEventSignatures(evt.Process, evt.Nonce, partials)
		if err != nil {
			panic(err)
		}
	}
}

package machine

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/fox-one/mixin-sdk-go"
)

const (
	ProcessPlatformQuorum = "quorum"

	ProcessCreditMulplifier = 10
)

type Process struct {
	Identifier string
	Address    string
	Credit     common.Integer
	Nonce      uint64
}

func (m *Machine) Spawn(ctx context.Context, p *Process) {
	logger.Verbosef("Spawn(%s, %s, %s, %d)", p.Identifier, p.Address, p.Nonce)
	go m.loopSendEvents(ctx, p)
}

func (m *Machine) loopSendEvents(ctx context.Context, p *Process) {
	for {
		events, err := m.store.ListSignedGroupEvents(p.Identifier, 100)
		if err != nil {
			panic(err)
		}
		if len(events) == 0 {
			time.Sleep(5 * time.Second)
			continue
		}
		cost, err := m.engine.EstimateCost(events)
		if err != nil {
			panic(err)
		}
		if p.Credit.Cmp(cost.Mul(ProcessCreditMulplifier)) < 0 {
			time.Sleep(1 * time.Minute)
			continue
		}

		err = m.engine.EnsureSendGroupEvents(p.Address, events)
		if err != nil {
			panic(err)
		}
		err = m.store.ExpireGroupEventsWithCost(events, cost)
		if err != nil {
			panic(err)
		}
		if cost.Sign() > 0 {
			p.Credit = p.Credit.Sub(cost)
		}
	}
}

func (m *Machine) loopReceiveEvents(ctx context.Context) {
	processed := make(map[string]bool)
	for {
		offset, err := m.store.ReadEngineGroupEventsOffset()
		if err != nil {
			panic(err)
		}
		events, err := m.engine.ReceiveGroupEvents(offset)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		for _, e := range events {
			if processed[e.ID()] {
				continue
			}
			as := buildAccountSnapshot(e, false)
			enough, err := m.store.CheckAccountSnapshot(as)
			if err != nil {
				panic(err)
			} else if !enough {
				logger.Verbosef("Process(%s, %d) => balance %s %s", e.Process, e.Nonce, e.Asset, e.Amount)
				time.Sleep(1 * time.Minute)
				break
			}
			processed[e.ID()] = true
			err = m.store.WriteAccountSnapshot(as)
			if err != nil {
				panic(err)
			}

			err = buildGroupTransaction(ctx, m.group, e)
			if err != nil {
				panic(err)
			}
		}
		err = m.store.WriteEngineGroupEventsOffset(offset + 1)
		if err != nil {
			panic(err)
		}
	}
}

func buildGroupTransaction(ctx context.Context, group *mtg.Group, evt *encoding.Event) error {
	traceId := mixin.UniqueConversationID(group.GenesisId(), fmt.Sprintf("%s:EVENT#%d", evt.Process, evt.Nonce))
	logger.Verbosef("Process(%s, %d) => buildGroupTransaction(%s, %v, %d, %s) => %s",
		evt.Process, evt.Nonce, evt.Asset, evt.Members, evt.Threshold, evt.Amount, traceId)
	amount := evt.Amount.String()
	memo := base64.RawURLEncoding.EncodeToString(evt.Extra)
	return group.BuildTransaction(ctx, evt.Asset, evt.Members, evt.Threshold, amount, memo, traceId)
}

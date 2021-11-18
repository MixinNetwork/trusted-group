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
	Platform   string
	Address    string
	Credit     common.Integer
	Nonce      uint64
}

func (m *Machine) Spawn(ctx context.Context, p *Process) {
	logger.Verbosef("Spawn(%s, %s, %s, %d)", p.Identifier, p.Platform, p.Address, p.Nonce)
	go m.loopSendEvents(ctx, p)
	go m.loopReceiveEvents(ctx, p)
}

func (m *Machine) loopSendEvents(ctx context.Context, p *Process) {
	engine := m.engines[p.Platform]
	for {
		events, err := m.store.ListSignedGroupEvents(p.Identifier, 100)
		if err != nil {
			panic(err)
		}
		if len(events) == 0 {
			time.Sleep(5 * time.Second)
			continue
		}
		cost, err := engine.EstimateCost(events)
		if err != nil {
			panic(err)
		}
		if p.Credit.Cmp(cost.Mul(ProcessCreditMulplifier)) < 0 {
			time.Sleep(1 * time.Minute)
			continue
		}

		err = engine.EnsureSendGroupEvents(p.Address, events)
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

func (m *Machine) loopReceiveEvents(ctx context.Context, p *Process) {
	engine := m.engines[p.Platform]
	processed := make(map[uint64]bool)
	for {
		offset, err := m.store.ReadEngineGroupEventsOffset(p.Identifier)
		if err != nil {
			panic(err)
		}
		events, err := engine.ReceiveGroupEvents(p.Address, offset, 100)
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}
		for _, e := range events {
			if processed[e.Nonce] {
				continue
			}
			account, err := m.store.ReadAccount(p.Identifier, e.Asset)
			if err != nil {
				panic(err)
			}
			if account.Balance.Cmp(e.Amount) < 0 {
				logger.Verbosef("Process(%s, %d) => balance %s %s %s", p.Identifier, p.Nonce, e.Asset, account.Balance, e.Amount)
				time.Sleep(1 * time.Minute)
				break
			}
			processed[e.Nonce] = true
			as := p.buildAccountSnapshot(e, false)
			err = m.store.WriteAccountSnapshot(as)
			if err != nil {
				panic(err)
			}

			err = p.buildGroupTransaction(ctx, m.group, e)
			if err != nil {
				panic(err)
			}
			err = m.store.WriteEngineGroupEventsOffset(p.Identifier, e.Nonce)
			if err != nil {
				panic(err)
			}
		}
		if len(events) < 100 {
			time.Sleep(5 * time.Second)
		}
	}
}

func (p *Process) buildGroupTransaction(ctx context.Context, group *mtg.Group, evt *encoding.Event) error {
	logger.Verbosef("Process(%s, %d) => buildGroupTransaction(%s, %v, %d, %s)",
		p.Identifier, evt.Nonce, evt.Asset, evt.Members, evt.Threshold, evt.Amount)
	amount := evt.Amount.String()
	traceId := mixin.UniqueConversationID(p.Identifier, fmt.Sprintf("EVENT#%d", evt.Nonce))
	memo := base64.RawURLEncoding.EncodeToString(evt.Extra)
	return group.BuildTransaction(ctx, evt.Asset, evt.Members, evt.Threshold, amount, memo, traceId)
}

package machine

import (
	"context"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
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

	machine *Machine
}

func (p *Process) Spwan(ctx context.Context, store Store) {
	go p.loopSendEvents(ctx, store)
	go p.loopReceiveEvents(ctx, store)
}

func (p *Process) Engine() Engine {
	return p.machine.engines[p.Platform]
}

func (p *Process) loopSendEvents(ctx context.Context, store Store) {
	for {
		events, err := store.ListGroupEvents(p.Identifier, 100)
		if err != nil {
			panic(err)
		}
		cost, err := p.Engine().EstimateCost(events)
		if err != nil {
			panic(err)
		}
		if p.Credit.Cmp(cost.Mul(ProcessCreditMulplifier)) < 0 {
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, e := range events {
			err = store.WriteAccountChange(p.Identifier, e.Asset, e.Amount, true)
			if err != nil {
				panic(err)
			}
		}

		err = p.Engine().SendGroupEvents(p.Address, events)
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}
		store.ExpireGroupEvents(events)
	}
}

func (p *Process) loopReceiveEvents(ctx context.Context, store Store) {
	for {
		offset, err := store.ReadEngineGroupEventsOffset(p.Identifier)
		if err != nil {
			panic(err)
		}
		events, err := p.Engine().ReceiveGroupEvents(p.Address, offset, 100)
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}
		for _, e := range events {
			account, err := store.ReadAccount(p.Identifier, e.Asset)
			if err != nil {
				panic(err)
			}
			if account.Balance.Cmp(e.Amount) < 0 {
				time.Sleep(1 * time.Minute)
				continue
			}
			err = store.WriteAccountChange(p.Identifier, e.Asset, e.Amount, false)
			if err != nil {
				panic(err)
			}

			err = p.buildGroupTransaction(ctx, nil, e)
			if err != nil {
				panic(err)
			}
			store.WriteEngineGroupEventsOffset(p.Identifier, e.Nonce)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (p *Process) buildGroupTransaction(ctx context.Context, group *mtg.Group, event *encoding.Event) error {
	amount := event.Amount.String()
	traceId := mixin.UniqueConversationID(p.Identifier, fmt.Sprintf("EVENT#%d", event.Nonce))
	return group.BuildTransaction(ctx, event.Asset, event.Members, event.Threshold, amount, event.Memo, traceId)
}

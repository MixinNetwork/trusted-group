package machine

import (
	"context"
	"strings"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type CollectibleToken struct {
	Id     string
	Symbol string
	Name   string
}

func (m *Machine) WriteNFOGroupEvent(ctx context.Context, pid string, out *mtg.CollectibleOutput, extra []byte) {
	logger.Verbosef("Machine.WriteNFOGroupEvent(%s, %v, %x)", pid, out, extra)
	m.procLock.RLock()
	defer m.procLock.RUnlock()

	proc := m.processes[pid]
	if proc == nil {
		return
	}
	if proc.Asset {
		meta, err := m.fetchCollectibleToken(ctx, out.TokenId)
		if err != nil {
			panic(err)
		}
		extra = append(meta, extra...)
	}
	if len(extra) > encoding.EventExtraMaxSize {
		return
	}

	done, err := m.store.CheckPendingGroupEventIdentifier(out.OutputId)
	if err != nil {
		panic(err)
	} else if done {
		return
	}

	amount := common.NewIntegerFromString(out.Amount.String())
	if amount.Cmp(common.NewInteger(1)) != 0 {
		panic(out.Amount)
	}
	evt := &encoding.Event{
		Process:   proc.Identifier,
		Asset:     out.TokenId,
		Members:   out.Senders,
		Threshold: int(out.SendersThreshold),
		Amount:    amount,
		Extra:     extra,
		Timestamp: uint64(out.CreatedAt.UnixNano()),
		Nonce:     proc.Nonce,
	}
	as := proc.buildAccountSnapshot(evt, true)
	err = m.store.WriteAccountSnapshot(as)
	if err != nil {
		panic(err)
	}
	err = m.store.WritePendingGroupEventAndNonce(evt, out.OutputId)
	if err != nil {
		panic(err)
	}
	proc.Nonce = proc.Nonce + 1
}

func (m *Machine) fetchCollectibleToken(ctx context.Context, id string) ([]byte, error) {
	old, err := m.store.ReadCollectibleToken(id)
	if err != nil {
		return nil, err
	} else if old != nil {
		return encodeCollectibleMeta(old.Symbol, old.Name), nil
	}
	token, err := m.mixin.ReadCollectiblesToken(ctx, id)
	if err != nil {
		return nil, err
	}
	if token.TokenID != id {
		panic(token.TokenID)
	}
	err = m.store.WriteCollectibleToken(&CollectibleToken{
		Id:     id,
		Symbol: token.Token,
		Name:   token.Group,
	})
	return encodeCollectibleMeta(token.Token, token.Group), err
}

func encodeCollectibleMeta(symbol, name string) []byte {
	symbol = "NFT#" + symbol
	name = "Collectible " + name
	enc := common.NewEncoder()
	enc.WriteInt(len(symbol))
	enc.Write([]byte(symbol))
	enc.WriteInt(len(name))
	enc.Write([]byte(name))
	return enc.Bytes()
}

func matchCollectibleMeta(symbol, name string) bool {
	return strings.HasPrefix(symbol, "NFT#") && strings.HasPrefix(name, "Collectible ")
}

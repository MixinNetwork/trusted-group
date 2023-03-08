package machine

import (
	"context"
	"strings"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type CollectibleToken struct {
	Id               string
	CollectionId     string
	TokenNumber      string
	CollectionName   string
	CollectionSymbol string
}

func (m *Machine) WriteNFOGroupEvent(ctx context.Context, pid string, out *mtg.CollectibleOutput, extra []byte) {
	logger.Verbosef("Machine.WriteNFOGroupEvent(%s, %v, %x)", pid, out, extra)
	m.procLock.RLock()
	defer m.procLock.RUnlock()

	proc := m.processes[pid]
	if proc == nil {
		return
	}
	meta, err := m.fetchCollectibleToken(ctx, out.TokenId)
	if err != nil {
		panic(err)
	}
	if len(meta) == 0 {
		return
	}
	if proc.Asset {
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
	} else if old != nil && old.CollectionId != "" {
		return encodeCollectibleMeta(old), nil
	}
	token, err := m.mixin.ReadCollectiblesToken(ctx, id)
	if err != nil {
		return nil, err
	}
	if token.TokenID != id {
		panic(token.TokenID)
	}
	old = &CollectibleToken{
		Id:               id,
		CollectionId:     token.Group,
		TokenNumber:      token.Token,
		CollectionName:   token.Meta.Group,
		CollectionSymbol: "NFT",
	}
	err = m.store.WriteCollectibleToken(old)
	return encodeCollectibleMeta(old), err
}

func encodeCollectibleMeta(t *CollectibleToken) []byte {
	if len(t.CollectionId) != 32 {
		return nil
	}
	symbol := "NFT#" + t.CollectionId + "#" + t.TokenNumber + "#" + t.CollectionSymbol
	name := "Collectible " + t.CollectionName
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

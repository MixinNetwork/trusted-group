package machine

import (
	"context"

	"github.com/MixinNetwork/mixin/common"
)

type CollectibleToken struct {
	Id     string
	Symbol string
	Name   string
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

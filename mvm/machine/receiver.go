package machine

import (
	"context"
	"encoding/base64"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type Asset struct {
	Id     string
	Symbol string
	Name   string
}

func (m *Machine) ProcessOutput(ctx context.Context, out *mtg.Output) {
	op, err := parseOperation(out.Memo)
	if err != nil {
		logger.Verbosef("parseOperation(%s) => %s", out.Memo, err)
		return
	}
	switch op.Purpose {
	case encoding.OperationPurposeAddProcess:
		ok := m.AddProcess(ctx, op.Process, op.Platform, op.Address, out, op.Extra)
		if ok && op.Platform == ProcessPlatformEOS {
			m.WriteGroupEvent(ctx, op.Process, out, op.Extra)
		}
	case encoding.OperationPurposeGroupEvent:
		m.WriteGroupEvent(ctx, op.Process, out, op.Extra)
	}
}

func (m *Machine) ProcessCollectibleOutput(context.Context, *mtg.CollectibleOutput) {
}

func parseOperation(memo string) (*encoding.Operation, error) {
	b, err := base64.RawURLEncoding.DecodeString(memo)
	if err != nil {
		return nil, err
	}
	return encoding.DecodeOperation(b)
}

func (m *Machine) fetchAssetMeta(ctx context.Context, id string) ([]byte, error) {
	old, err := m.store.ReadAsset(id)
	if err != nil {
		return nil, err
	} else if old != nil {
		return encodeAssetMeta(old.Symbol, old.Name), nil
	}
	asset, err := m.mixin.ReadAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	err = m.store.WriteAsset(&Asset{
		Id:     id,
		Symbol: asset.Symbol,
		Name:   asset.Name,
	})
	return encodeAssetMeta(asset.Symbol, asset.Name), err
}

func encodeAssetMeta(symbol, name string) []byte {
	enc := common.NewEncoder()
	enc.WriteInt(len(symbol))
	enc.Write([]byte(symbol))
	enc.WriteInt(len(name))
	enc.Write([]byte(name))
	return enc.Bytes()
}

package machine

import (
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/shopspring/decimal"
	"golang.org/x/net/context"
)

const (
	ProcessRegistrationAssetId = "c94ac88f-4671-3976-b60a-09064f1811e8"
)

type Machine struct {
	Store Store
}

func Boot() (*Machine, error) {
	return nil, nil
}

func (m *Machine) Loop(ctx context.Context) {
	processes, err := m.Store.ListProcesses()
	if err != nil {
		panic(err)
	}
	for _, p := range processes {
		p.Spwan(ctx, m.Store)
	}
}

func (m *Machine) AddProcess(id string, platform, address string, out *mtg.Output) {
	if out.AssetID != ProcessRegistrationAssetId {
		return
	}
	if out.Amount.Cmp(decimal.NewFromInt(1)) < 0 {
		return
	}
	switch platform {
	case ProcessPlatformQuorum:
		m.Store.WriteProcess(nil)
	default:
	}
}

func (m *Machine) WriteGroupEvent(out *mtg.Output) {
	panic(0)
}

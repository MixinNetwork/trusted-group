package machine

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type AccountSnapshot struct {
	Process string
	Nonce   uint64
	Asset   string
	Amount  common.Integer
	Credit  bool
}

func (p *Process) buildAccountSnapshot(e *encoding.Event, credit bool) *AccountSnapshot {
	if p.Identifier != e.Process {
		panic(e.Process)
	}
	return &AccountSnapshot{
		Process: p.Identifier,
		Nonce:   e.Nonce,
		Asset:   e.Asset,
		Amount:  e.Amount,
		Credit:  credit,
	}
}

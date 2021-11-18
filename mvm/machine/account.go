package machine

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
)

type Account struct {
	Process string
	Asset   string
	Balance common.Integer
}

type AccountSnapshot struct {
	Process string
	Nonce   uint64
	Asset   string
	Amount  common.Integer
	Credit  bool
}

func (p *Process) buildAccountSnapshot(e *encoding.Event, credit bool) *AccountSnapshot {
	return &AccountSnapshot{
		Process: p.Identifier,
		Nonce:   e.Nonce,
		Asset:   e.Asset,
		Amount:  e.Amount,
		Credit:  credit,
	}
}

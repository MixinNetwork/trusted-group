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

func buildAccountSnapshot(e *encoding.Event, credit bool) *AccountSnapshot {
	if e.Process == "" {
		panic(e)
	}
	return &AccountSnapshot{
		Process: e.Process,
		Nonce:   e.Nonce,
		Asset:   e.Asset,
		Amount:  e.Amount,
		Credit:  credit,
	}
}

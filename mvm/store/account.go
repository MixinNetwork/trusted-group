package store

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
)

func (bs *BadgerStore) ReadAccount(pid string, asset string) (*machine.Account, error) {
	panic(0)
}

func (bs *BadgerStore) WriteAccountChange(pid string, asset string, amount common.Integer, credit bool) error {
	panic(0)
}

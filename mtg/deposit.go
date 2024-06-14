package mtg

import (
	"context"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/logger"
)

type DepositEntry struct {
	Destination string
	Tag         string
}

type SafeDepositView struct {
	DepositHash  string `json:"deposit_hash"`
	DepositIndex int64  `json:"deposit_index"`
	Sender       string `json:"sender"`
	Destination  string `json:"destination"`
	Tag          string `json:"tag"`
}

func (e DepositEntry) String() string {
	return fmt.Sprintf("%s:%s", e.Destination, e.Tag)
}

func (grp *Group) readOutputDepositUntilSufficient(ctx context.Context, id string) (*SafeDepositView, error) {
	for {
		var deposit *SafeDepositView
		err := grp.mixin.Get(ctx, fmt.Sprintf("/safe/outputs/%s/deposit", id), nil, &deposit)
		logger.Verbosef("Group.readOutputDeposit(%s) => %v %v\n", id, deposit, err)
		if err != nil && CheckRetryableError(err) {
			time.Sleep(3 * time.Second)
			continue
		}
		return deposit, err
	}
}

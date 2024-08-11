package mtg

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

func (e DepositEntry) UniqueKey() string {
	return fmt.Sprintf("%s:%s", e.Destination, e.Tag)
}

func (grp *Group) readOutputDepositUntilSufficient(ctx context.Context, id string) (*SafeDepositView, error) {
	key := fmt.Sprintf("readOutputDepositUntilSufficient(%s)", id)
	val, err := grp.store.ReadCache(ctx, key)
	if err != nil {
		panic(err)
	}
	if val != "" {
		var r SafeDepositView
		err = json.Unmarshal([]byte(val), &r)
		if err != nil {
			panic(err)
		}
		return &r, nil
	}
	r, err := grp.readOutputDepositUntilSufficientImpl(ctx, id)
	if err != nil || r == nil {
		return r, err
	}
	b, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	err = grp.store.WriteCache(ctx, key, string(b))
	if err != nil {
		panic(err)
	}
	return r, nil
}

func (grp *Group) readOutputDepositUntilSufficientImpl(ctx context.Context, id string) (*SafeDepositView, error) {
	for {
		var deposit *SafeDepositView
		err := grp.mixin.Get(ctx, fmt.Sprintf("/safe/outputs/%s/deposit", id), nil, &deposit)
		logger.Verbosef("Group.readOutputDeposit(%s) => %v %v\n", id, deposit, err)
		if err != nil {
			if CheckRetryableError(err) {
				time.Sleep(3 * time.Second)
				continue
			}
			if strings.Contains(err.Error(), "not found") {
				return nil, nil
			}
		}
		return deposit, err
	}
}

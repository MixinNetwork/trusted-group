package models

import (
	"context"
	"multisig/configs"
	"sort"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client"
)

func ReadMultisigs(ctx context.Context, amount string) (*bot.MultisigUTXO, error) {
	mixin := configs.AppConfig.Mixin
	outputs, err := bot.ReadMultisigs(ctx, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
	if err != nil {
		return nil, err
	}
	for _, output := range outputs {
		members := output.Members
		sort.Slice(members, func(i, j int) bool {
			return members[i] < members[j]
		})
		strings.Join(members, ",")
	}
	return nil, nil
}

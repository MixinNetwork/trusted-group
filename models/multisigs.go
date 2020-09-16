package models

import (
	"context"
	"multisig/configs"
	"sort"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
)

func ReadMultisigs(ctx context.Context, amount string) (*bot.MultisigUTXO, error) {
	mixin := configs.AppConfig.Mixin
	receivers := []string{mixin.AppID}
	for _, user := range mixin.Users {
		receivers = append(receivers, user.UserID)
	}
	sort.Slice(receivers, func(i, j int) bool {
		return receivers[i] < receivers[j]
	})
	receiverStr := strings.Join(receivers, ",")
	outputs, err := bot.ReadMultisigs(ctx, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
	if err != nil {
		return nil, err
	}
	for _, output := range outputs {
		members := output.Members
		sort.Slice(members, func(i, j int) bool {
			return members[i] < members[j]
		})
		if receiverStr == strings.Join(members, ",") && number.FromString(amount).Cmp(number.FromString(output.Amount)) == 0 {
			return output, nil
		}
	}
	return nil, nil
}

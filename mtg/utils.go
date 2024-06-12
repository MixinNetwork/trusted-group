package mtg

import (
	"context"
	"strings"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
)

const (
	contextKeyEnvironment = "environment"
)

func EnableTestEnvironment(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKeyEnvironment, "test")
}

func CheckTestEnvironment(ctx context.Context) bool {
	val := ctx.Value(contextKeyEnvironment)
	if val == nil {
		return false
	}
	env, ok := val.(string)
	return ok && env == "test"
}

func UniqueId(a, b string) string {
	return mixin.UniqueConversationID(a, b)
}

func CheckRetryableError(err error) bool {
	es := err.Error()
	switch {
	case strings.Contains(es, "Client.Timeout exceeded"):
	case strings.Contains(es, "Bad Gateway"):
	case strings.Contains(es, "Internal Server Error"):
	case strings.Contains(es, "invalid character '<' looking for beginning of value"):
	default:
		return false
	}
	return true
}

func NewMixAddress(ctx context.Context, members []string, threshold byte, storage bool) (*mixin.MixAddress, bool, error) {
	if CheckTestEnvironment(ctx) {
		for i, m := range members {
			_, err := mixinnet.AddressFromString(m)
			if err == nil {
				break
			}
			members[i] = UniqueId(m, m)
		}
	}

	thr := threshold
	if storage {
		threshold = 1
	}

	isUuidMembers := false
	ma, err := mixin.NewMainnetMixAddress(members, threshold)
	if err != nil {
		ma, err = mixin.NewMixAddress(members, threshold)
		if err != nil {
			return nil, false, err
		}
		isUuidMembers = true
	}
	ma.Threshold = thr
	return ma, isUuidMembers, nil
}

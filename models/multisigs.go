package models

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"multisig/configs"
	"sort"
	"strings"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
)

// cnb b9f49cf777dc4d03bc54cd1367eebca319f8603ea1ce18910d09e2c540c630d8

func ReadMultisig(ctx context.Context, amount string) (*bot.MultisigUTXO, error) {
	mixin := configs.AppConfig.Mixin
	receivers := []string{mixin.AppID}
	for _, user := range mixin.Users {
		receivers = append(receivers, user.UserID)
	}
	sort.Slice(receivers, func(i, j int) bool {
		return receivers[i] < receivers[j]
	})
	receiverStr := strings.Join(receivers, ",")
	outputs, err := bot.ReadMultisigs(ctx, 500, "", mixin.AppID, mixin.SessionID, mixin.PrivateKey)
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

type Input struct {
	Hash  string `json:"hash"`
	Index int64  `json:"index"`
}

type Output struct {
	Mask   string   `json:"mask"`
	Keys   []string `json:"keys"`
	Amount string   `json:"amount"`
	Script string   `json:"script"`
}

type Transaction struct {
	Inputs  []*Input  `json:"inputs"`
	Outputs []*Output `json:"outputs"`
	Asset   string    `json:"asset"`
	Extra   string    `json:"extra"`
	Hash    string    `json:"hash"`
}

type signerInput struct {
	Inputs []struct {
		Hash  crypto.Hash `json:"hash"`
		Index int         `json:"index"`
	} `json:"inputs"`
	Outputs []struct {
		Type   uint8          `json:"type"`
		Mask   crypto.Key     `json:"mask"`
		Keys   []crypto.Key   `json:"keys"`
		Amount common.Integer `json:"amount"`
		Script common.Script  `json:"script"`
	}
	Asset crypto.Hash `json:"asset"`
	Extra string      `json:"extra"`
}

func buildTransaction(data []byte) (string, error) {
	var raw signerInput
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return "", err
	}

	tx := common.NewTransaction(raw.Asset)
	for _, in := range raw.Inputs {
		tx.AddInput(in.Hash, in.Index)
	}

	for _, out := range raw.Outputs {
		if out.Mask.HasValue() {
			tx.Outputs = append(tx.Outputs, &common.Output{
				Type:   out.Type,
				Amount: out.Amount,
				Keys:   out.Keys,
				Script: out.Script,
				Mask:   out.Mask,
			})
		}
	}

	extra, err := hex.DecodeString(raw.Extra)
	if err != nil {
		return "", err
	}
	tx.Extra = extra

	signed := tx.AsLatestVersion()
	return hex.EncodeToString(signed.Marshal()), nil
}

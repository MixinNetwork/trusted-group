package models

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"multisig/configs"
	"multisig/session"
	"sort"
	"strings"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
)

const (
	MultisigStateSigned  = "signed"
	MultisigStateInitial = "initial"
)

// cnb b9f49cf777dc4d03bc54cd1367eebca319f8603ea1ce18910d09e2c540c630d8

func ReadMultisig(ctx context.Context, amount, memo string) (*bot.MultisigUTXO, error) {
	mixin := configs.AppConfig.Mixin
	receivers := mixin.Receivers
	receivers = append(receivers, mixin.AppID)
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
		if receiverStr == strings.Join(members, ",") && number.FromString(amount).Cmp(number.FromString(output.Amount)) == 0 && output.Memo == memo {
			return output, nil
		}
	}
	return nil, nil
}

func LoopingSignMultisig(ctx context.Context) error {
	network := NewMixinNetwork("http://35.234.74.25:8239")
	for {
		err := handleMultisig(ctx, network)
		if err != nil {
			time.Sleep(time.Second)
			session.Logger(ctx).Errorf("handleMultisig %#v", err)
			continue
		}
		time.Sleep(10 * time.Second)
	}
}

func handleMultisig(ctx context.Context, network *MixinNetwork) error {
	mixin := configs.AppConfig.Mixin
	outputs, err := bot.ReadMultisigs(ctx, 500, "", mixin.AppID, mixin.SessionID, mixin.PrivateKey)
	if err != nil {
		return err
	}
	for _, output := range outputs {
		if output.State == MultisigStateSigned {
			request, err := bot.CreateMultisig(ctx, "sign", output.SignedTx, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
			if err != nil {
				return err
			}
			if request.State == MultisigStateInitial {
				pin, err := bot.EncryptPIN(ctx, mixin.Pin, mixin.PinToken, mixin.SessionID, mixin.PrivateKey, uint64(time.Now().UnixNano()))
				if err != nil {
					return err
				}
				request, err = bot.SignMultisig(ctx, request.RequestId, pin, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
				if err != nil {
					return err
				}
			}
			if mixin.Master {
				payment, err := FindPaymentByMemo(ctx, output.Memo)
				if err != nil {
					return err
				}
				if payment != nil {
					continue
				}
				data, err := hex.DecodeString(output.SignedTx)
				if err != nil {
					return err
				}
				var stx common.SignedTransaction
				err = common.MsgpackUnmarshal(data, &stx)
				if err != nil {
					return err
				}
				if len(stx.Signatures) > 0 && len(stx.Signatures[0]) < int(output.Threshold) {
					return nil
				}
				_, err = network.SendRawTransaction(output.SignedTx)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
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

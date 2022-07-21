package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/config"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/mdp/qrterminal"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
)

func invokeProcessCmd(c *cli.Context) error {
	ctx := context.Background()

	cp := c.String("machine")
	if strings.HasPrefix(cp, "~/") {
		usr, _ := user.Current()
		cp = filepath.Join(usr.HomeDir, (cp)[2:])
	}
	conf, err := config.ReadConfiguration(cp)
	if err != nil {
		return err
	}

	kp := c.String("key")
	if strings.HasPrefix(kp, "~/") {
		usr, _ := user.Current()
		kp = filepath.Join(usr.HomeDir, (kp)[2:])
	}
	kb, err := os.ReadFile(kp)
	if err != nil {
		return err
	}

	var key struct {
		PIN        string `json:"pin"`
		ClientId   string `json:"client_id"`
		SessionId  string `json:"session_id"`
		PINToken   string `json:"pin_token"`
		PrivateKey string `json:"private_key"`
	}
	err = json.Unmarshal(kb, &key)
	if err != nil {
		return err
	}

	s := &mixin.Keystore{
		ClientID:   key.ClientId,
		SessionID:  key.SessionId,
		PrivateKey: key.PrivateKey,
		PinToken:   key.PINToken,
	}
	client, err := mixin.NewFromKeystore(s)
	if err != nil {
		return err
	}
	err = client.VerifyPin(ctx, key.PIN)
	if err != nil {
		return err
	}

	token := uuid.FromStringOrNil(c.String("token"))
	if token.String() == c.String("token") {
		return doCollectible(ctx, client, conf, c, key.PIN)
	}
	return doPayment(ctx, client, conf, c)
}

func doPayment(ctx context.Context, client *mixin.Client, conf *config.Configuration, c *cli.Context) error {
	amount, err := decimal.NewFromString(c.String("amount"))
	if err != nil {
		return err
	}
	if amount.Cmp(decimal.NewFromFloat(0.00000001)) < 0 {
		return fmt.Errorf("invalid amount %s", amount)
	}

	trace, err := uuid.NewV4()
	if err != nil {
		return err
	}
	extra, _ := hex.DecodeString(c.String("extra"))
	op := &encoding.Operation{
		Purpose: encoding.OperationPurposeGroupEvent,
		Process: c.String("process"),
		Extra:   extra,
	}
	input := mixin.TransferInput{
		AssetID: c.String("asset"),
		Amount:  amount,
		TraceID: trace.String(),
	}
	input.OpponentMultisig.Receivers = conf.MTG.Genesis.Members
	input.OpponentMultisig.Threshold = uint8(conf.MTG.Genesis.Threshold)
	input.Memo = base64.RawURLEncoding.EncodeToString(op.Encode())
	pay, err := client.VerifyPayment(ctx, input)
	if err != nil {
		return err
	}
	url := "mixin://codes/" + pay.CodeID
	fmt.Println(url)
	qrterminal.GenerateHalfBlock(url, qrterminal.H, os.Stdout)
	return nil
}

func doCollectible(ctx context.Context, client *mixin.Client, conf *config.Configuration, c *cli.Context, pin string) error {
	token, err := client.ReadCollectiblesToken(ctx, c.String("token"))
	if err != nil {
		return err
	} else if token == nil {
		return fmt.Errorf("invalid token id %s", c.String("token"))
	}
	outputs, err := readUnspentCollectibleOutputs(ctx, client, []string{client.ClientID}, 1, time.Time{}, 500)
	if err != nil {
		return err
	}
	var out *mixin.CollectibleOutput
	for _, o := range outputs {
		if o.TokenID == token.TokenID {
			out = o
			break
		}
	}
	if out == nil {
		return fmt.Errorf("no token found %s", token.TokenID)
	}

	trace, err := uuid.NewV4()
	if err != nil {
		return err
	}
	extra, _ := hex.DecodeString(c.String("extra"))
	op := &encoding.Operation{
		Purpose: encoding.OperationPurposeGroupEvent,
		Process: c.String("process"),
		Extra:   extra,
	}
	extra = []byte(base64.RawURLEncoding.EncodeToString(op.Encode()))

	raw, err := buildRawCollectibleTransaction(ctx, client, conf, token, out, extra, trace.String())
	if err != nil {
		return err
	}
	req, err := client.CreateCollectibleRequest(ctx, "SIGN", hex.EncodeToString(raw.PayloadMarshal()))
	if err != nil {
		return err
	}
	req, err = client.SignCollectibleRequest(ctx, req.RequestID, pin)
	if err != nil {
		return err
	}
	hash, err := client.SendRawTransaction(ctx, req.RawTransaction)
	if err != nil {
		return err
	}
	fmt.Println(req.RawTransaction, hash)
	return nil
}

func buildRawCollectibleTransaction(ctx context.Context, client *mixin.Client, conf *config.Configuration, token *mixin.CollectibleToken, utxo *mixin.CollectibleOutput, extra []byte, traceId string) (*common.VersionedTransaction, error) {
	ver := common.NewTransaction(crypto.Hash(token.MixinID))
	ver.Extra = mtg.BuildExtraNFO(extra)

	if utxo.Amount.Cmp(decimal.NewFromInt(1)) != 0 {
		panic(utxo.OutputID)
	}
	ver.AddInput(crypto.Hash(utxo.TransactionHash), utxo.OutputIndex)

	keys, err := client.BatchReadGhostKeys(ctx, []*mixin.GhostInput{{
		Receivers: conf.MTG.Genesis.Members,
		Index:     0,
		Hint:      traceId,
	}})
	if err != nil {
		return nil, err
	}

	out := keys[0].DumpOutput(uint8(conf.MTG.Genesis.Threshold), utxo.Amount)
	ver.Outputs = append(ver.Outputs, newCommonOutput(out))
	return ver.AsLatestVersion(), nil
}

func newCommonOutput(out *mixin.Output) *common.Output {
	cout := &common.Output{
		Type:   common.OutputTypeScript,
		Amount: common.NewIntegerFromString(out.Amount.String()),
		Script: common.Script(out.Script),
		Mask:   crypto.Key(out.Mask),
	}
	for _, k := range out.Keys {
		ck := crypto.Key(k)
		cout.Keys = append(cout.Keys, &ck)
	}
	return cout
}

func readUnspentCollectibleOutputs(ctx context.Context, client *mixin.Client, members []string, threshold uint8, offset time.Time, limit int) ([]*mixin.CollectibleOutput, error) {
	params := make(map[string]string)
	if !offset.IsZero() {
		params["offset"] = offset.UTC().Format(time.RFC3339Nano)
	}
	if limit > 0 {
		params["limit"] = fmt.Sprint(limit)
	}
	if threshold < 1 || int(threshold) > len(members) {
		return nil, fmt.Errorf("invalid members %v %d", members, threshold)
	}
	params["members"] = mixin.HashMembers(members)
	params["threshold"] = fmt.Sprint(threshold)
	params["state"] = "unspent"

	var outputs []*mixin.CollectibleOutput
	err := client.Get(ctx, "/collectibles/outputs", params, &outputs)
	if err != nil {
		return nil, err
	}
	return outputs, nil
}

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/MixinNetwork/trusted-group/mvm/config"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
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

	trace, err := uuid.NewV4()
	if err != nil {
		return err
	}
	op := &encoding.Operation{
		Purpose: encoding.OperationPurposeGroupEvent,
		Process: c.String("process"),
		Extra:   []byte(c.String("extra")),
	}

	amount, err := decimal.NewFromString(c.String("amount"))
	if err != nil {
		panic(err)
	}

	input := mixin.TransferInput{
		AssetID: machine.ProcessRegistrationAssetId,
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

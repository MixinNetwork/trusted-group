package main

import (
	"context"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/config"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/urfave/cli/v2"
)

func unlockRequestCmd(c *cli.Context) error {
	logger.SetLevel(logger.VERBOSE)
	ctx := context.Background()

	cp := c.String("config")
	if strings.HasPrefix(cp, "~/") {
		usr, _ := user.Current()
		cp = filepath.Join(usr.HomeDir, (cp)[2:])
	}
	conf, err := config.ReadConfiguration(cp)
	if err != nil {
		return err
	}

	s := &mixin.Keystore{
		ClientID:   conf.MTG.App.ClientId,
		SessionID:  conf.MTG.App.SessionId,
		PrivateKey: conf.MTG.App.PrivateKey,
		PinToken:   conf.MTG.App.PinToken,
	}
	client, err := mixin.NewFromKeystore(s)
	if err != nil {
		return err
	}
	err = client.VerifyPin(ctx, conf.MTG.App.PIN)
	if err != nil {
		return err
	}

	req, err := client.CreateMultisig(ctx, mixin.MultisigActionUnlock, c.String("raw"))
	if err != nil {
		return err
	}
	return client.UnlockMultisig(ctx, req.RequestID, conf.MTG.App.PIN)
}

package main

import (
	"context"
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/tip/messenger"
	"github.com/MixinNetwork/trusted-group/mvm/config"
	"github.com/MixinNetwork/trusted-group/mvm/eos"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
	"github.com/MixinNetwork/trusted-group/mvm/quorum"
	"github.com/MixinNetwork/trusted-group/mvm/store"
	"github.com/urfave/cli/v2"
)

func bootCmd(c *cli.Context) error {
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

	bp := c.String("dir")
	if strings.HasPrefix(bp, "~/") {
		usr, _ := user.Current()
		bp = filepath.Join(usr.HomeDir, (bp)[2:])
	}
	db, err := store.OpenBadger(ctx, bp)
	if err != nil {
		return err
	}
	defer db.Close()

	group, err := mtg.BuildGroup(ctx, db, conf.MTG)
	if err != nil {
		return err
	}

	messenger, err := messenger.NewMixinMessenger(ctx, conf.Messenger)
	if err != nil {
		return err
	}
	im, err := machine.Boot(conf.Machine, group, db, messenger)
	if err != nil {
		return err
	}

	platform := c.String("platform")
	if platform == machine.ProcessPlatformQuorum {
		en, err := quorum.Boot(conf.Quorum)
		if err != nil {
			return err
		}
		im.SetEngine(machine.ProcessPlatformQuorum, en)
	} else if platform == machine.ProcessPlatformEos {
		en, err := eos.Boot(conf.Eos)
		if err != nil {
			return err
		}
		im.SetEngine(machine.ProcessPlatformEos, en)
	} else {
		return cli.Exit(fmt.Errorf("unsupported platform %s", platform), 1)
	}

	go im.Loop(ctx)

	group.AddWorker(im)
	group.Run(ctx)

	return nil
}

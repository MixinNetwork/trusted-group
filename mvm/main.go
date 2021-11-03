package main

import (
	"context"
	"flag"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/nfo/store"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
)

func main() {
	logger.SetLevel(logger.VERBOSE)
	ctx := context.Background()

	bp := flag.String("d", "~/.mixin/mvm/data", "database directory path")
	cp := flag.String("c", "~/.mixin/mvm/config.toml", "configuration file path")
	flag.Parse()

	if strings.HasPrefix(*cp, "~/") {
		usr, _ := user.Current()
		*cp = filepath.Join(usr.HomeDir, (*cp)[2:])
	}
	conf, err := mtg.Setup(*cp)
	if err != nil {
		panic(err)
	}

	if strings.HasPrefix(*bp, "~/") {
		usr, _ := user.Current()
		*bp = filepath.Join(usr.HomeDir, (*bp)[2:])
	}
	db, err := store.OpenBadger(ctx, *bp)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	im, err := machine.Boot()
	if err != nil {
		panic(err)
	}
	go im.Loop()

	group, err := mtg.BuildGroup(ctx, db, conf)
	if err != nil {
		panic(err)
	}
	grw := machine.NewGroupReceiver()
	group.AddWorker(grw)
	group.Run(ctx)
}

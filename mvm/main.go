package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

const AppVersion = "0.5.2"

func main() {
	app := &cli.App{
		Name:                 "mvm",
		Usage:                "MVM (Mixin Virtual Machine) is a smart contract platform built with MTG.",
		Version:              AppVersion,
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:   "boot",
				Usage:  "Boot a MVM node",
				Action: bootCmd,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Value:   "~/.mixin/mvm/config.toml",
						Usage:   "The configuration file path",
					},
					&cli.StringFlag{
						Name:    "dir",
						Aliases: []string{"d"},
						Value:   "~/.mixin/mvm/data",
						Usage:   "The database directory path",
					},
					&cli.IntFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Value:   9000,
						Usage:   "The RPC server http port",
					},
					&cli.BoolFlag{
						Name:    "profile",
						Aliases: nil,
						Value:   false,
						Usage:   "Enable profile",
					},
				},
			},
			{
				Name:   "unlockrequest",
				Usage:  "Unlock a MTG transaction request",
				Action: unlockRequestCmd,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Value:   "~/.mixin/mvm/config.toml",
						Usage:   "The configuration file path",
					},
					&cli.StringFlag{
						Name:  "raw",
						Usage: "The raw transaction to unlock",
					},
				},
			},
			{
				Name:   "publish",
				Usage:  "Publish a MVM app",
				Action: publishAppCmd,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "machine",
						Aliases: []string{"m"},
						Value:   "~/.mixin/mvm/config.toml",
						Usage:   "The MVM members and threshold configuration",
					},
					&cli.StringFlag{
						Name:    "key",
						Aliases: []string{"k"},
						Usage:   "The app key JSON file",
					},
					&cli.StringFlag{
						Name:    "platform",
						Aliases: []string{"p"},
						Value:   "quorum",
						Usage:   "The smart contract platform",
					},
					&cli.StringFlag{
						Name:    "address",
						Aliases: []string{"a"},
						Usage:   "The smart contract address",
					},
					&cli.StringFlag{
						Name:    "extra",
						Aliases: []string{"e"},
						Usage:   "The extra",
					},
				},
			},
			{
				Name:   "invoke",
				Usage:  "Invoke a MVM app",
				Action: invokeProcessCmd,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "machine",
						Aliases: []string{"m"},
						Value:   "~/.mixin/mvm/config.toml",
						Usage:   "The MVM members and threshold configuration",
					},
					&cli.StringFlag{
						Name:    "key",
						Aliases: []string{"k"},
						Usage:   "The app key JSON file",
					},
					&cli.StringFlag{
						Name:    "process",
						Aliases: []string{"p"},
						Usage:   "The app ID",
					},
					&cli.StringFlag{
						Name:  "asset",
						Usage: "Asset ID",
					},
					&cli.StringFlag{
						Name:  "amount",
						Value: "0.123",
						Usage: "Asset amount",
					},
					&cli.StringFlag{
						Name:  "token",
						Usage: "Collectible Token ID",
					},
					&cli.StringFlag{
						Name:    "extra",
						Aliases: []string{"e"},
						Usage:   "The extra",
					},
				},
			},
			{
				Name:   "decode",
				Usage:  "Decode a MVM message",
				Action: decodeMsgCmd,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

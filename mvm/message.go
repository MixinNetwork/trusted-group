package main

import (
	"encoding/base64"
	"fmt"

	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/urfave/cli/v2"
)

func decodeMsgCmd(c *cli.Context) error {
	b, err := base64.RawURLEncoding.DecodeString(c.Args().First())
	if err != nil {
		return err
	}
	evt, err := encoding.DecodeEvent(b[:len(b)-8])
	if err != nil {
		return err
	}
	fmt.Println(evt)
	return nil
}

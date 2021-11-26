package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/tip/messenger"
	"github.com/MixinNetwork/trusted-group/mvm/eos"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
	"github.com/MixinNetwork/trusted-group/mvm/quorum"
	"github.com/pelletier/go-toml"
)

type Configuration struct {
	MTG       *mtg.Configuration            `toml:"mtg"`
	Machine   *machine.Configuration        `toml:"machine"`
	Quorum    *quorum.Configuration         `toml:"quorum"`
	Eos       *eos.Configuration            `toml:"eos"`
	Messenger *messenger.MixinConfiguration `toml:"messenger"`
}

func ReadConfiguration(path string) (*Configuration, error) {
	if strings.HasPrefix(path, "~/") {
		usr, _ := user.Current()
		path = filepath.Join(usr.HomeDir, (path)[2:])
	}
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Configuration
	err = toml.Unmarshal(f, &conf)
	return &conf, err
}

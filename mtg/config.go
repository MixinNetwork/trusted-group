package mtg

import (
	"io/ioutil"

	"github.com/pelletier/go-toml"
)

type Configuration struct {
	App struct {
		AppId             string `toml:"app-id"`
		SessionId         string `toml:"session-id"`
		SessionPrivateKey string `toml:"session-private-key"`
		ServerPublicKey   string `toml:"server-public-key"`
		SpendPrivateKey   string `toml:"spend-private-key"`
	} `toml:"app"`
	Genesis struct {
		Members   []string `toml:"members"`
		Threshold int      `toml:"threshold"`
		Epoch     uint64   `toml:"epoch"`
	} `toml:"genesis"`
	StoreDir         string `toml:"store-dir"`
	GroupSize        int    `toml:"group-size"`
	LoopWaitDuration int64  `toml:"loop-wait-duration"`
}

func Setup(path string) (*Configuration, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Configuration
	err = toml.Unmarshal(f, &conf)
	return &conf, err
}

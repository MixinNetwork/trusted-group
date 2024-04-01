package mtg

import (
	"io/ioutil"

	"github.com/pelletier/go-toml"
)

type Configuration struct {
	App struct {
		AppId             string `toml:"app_id"`
		SessionId         string `toml:"session_id"`
		SessionPrivateKey string `toml:"session_private_key"`
		ServerPublicKey   string `toml:"server_public_key"`
		SpendPrivateKey   string `toml:"spend_private_key"`
	} `toml:"app"`
	Genesis struct {
		Members   []string `toml:"members"`
		Threshold int      `toml:"threshold"`
		Timestamp int64    `toml:"timestamp"`
	} `toml:"genesis"`
	GroupSize        int   `toml:"group-size"`
	LoopWaitDuration int64 `toml:"loop-wait-duration"`
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

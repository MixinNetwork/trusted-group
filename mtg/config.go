package mtg

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
	Project          string `toml:"project"`
	StoreDir         string `toml:"store-dir"`
	GroupSize        int    `toml:"group-size"`
	LoopWaitDuration int64  `toml:"loop-wait-duration"`
}

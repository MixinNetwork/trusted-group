package machine

const (
	ProcessPlatformQuorum = "quorum"
)

type Process struct {
	Platform   string
	Identifier string
	Balance    string
	Cost       string
	Credit     string
}

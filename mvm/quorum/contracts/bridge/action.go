package main

const (
	ActionWithdrawal = "WITHDRAWL"
	ActionTransfer   = "TRANSFER"
)

type Action struct {
	Category    string
	Destination string
	Tag         string
	Receivers   []string
	Threshold   int64
	Extra       string
}

func decryptData() {
	// bare user private key is on the metamask client side
	// they encrypt everything with bare private key and bot public key
	// they access most stuffs from the client side with the private key
}

func (p *Proxy) decodeAction(u *User, data string) (*Action, error) {
	// decrypt data at first
	// check data format according to the registry
	// if storage => do the storage contract query => p.Read(nil, big.Int)
	// else return null => because we only allow storage interface
	panic(0)
}

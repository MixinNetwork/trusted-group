package encoding

import "github.com/MixinNetwork/mixin/common"

type Event struct {
	Process   string
	Asset     string
	Receivers []string
	Threshold int
	Amount    common.Integer
	Memo      string
	Nonce     uint64
	Signature []byte
}

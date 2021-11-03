package encoding

import "github.com/MixinNetwork/mixin/common"

//
// MTG => VM
// nonce || asset || amount || memo || members || threshold || sig
//
// VM => MTG
// nonce || asset || amount || memo || members || threshold
//
type Event struct {
	Process   string
	Asset     string
	Members   []string
	Threshold int
	Amount    common.Integer
	Memo      string
	Nonce     uint64
	Signature []byte
}

package quorum

import "github.com/MixinNetwork/trusted-group/mvm/encoding"

func (e *Engine) storeReadContractLogsOffset(address string) uint64 {
	panic(0)
}

func (e *Engine) storeWriteContractLogsOffset(address string, offset uint64) {
	panic(0)
}

func (e *Engine) storeReadLastContractEventNonce(address string) uint64 {
	panic(0)
}

func (e *Engine) storeWriteContractEvent(address string, evt *encoding.Event) {
	panic(0)
}

func (e *Engine) storeListContractEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	panic(0)
}

func (e *Engine) storeWriteGroupEvents(address string, events []*encoding.Event) error {
	panic(0)
}

func (e *Engine) storeWriteContractNotifier(address string, key, state string) error {
	panic(0)
}

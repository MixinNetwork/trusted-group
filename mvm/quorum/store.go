package quorum

import (
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/dgraph-io/badger/v3"
)

const (
	prefixQuorumContractNotifier = "QUORUM:CONTRACT:NOTIFIER:"
)

func (e *Engine) storeWriteContractNotifier(address, key string) error {
	panic(0)
}

func (e *Engine) storeReadContractNotifier(address string) string {
	panic(0)
}

func (e *Engine) storeListContractNotifiers() ([]string, error) {
	panic(0)
}

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

func (e *Engine) storeListGroupEvents(address string, offset uint64, limit int) ([]*encoding.Event, error) {
	panic(0)
}

func openBadger(dir string) *badger.DB {
	opts := badger.DefaultOptions(dir)
	db, err := badger.Open(opts)
	if err != nil {
		panic(err)
	}
	return db
}

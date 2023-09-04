package mtg

import (
	"context"

	"github.com/MixinNetwork/mixin/crypto"
)

type Store interface {
	WriteProperty(key, val []byte) error
	ReadProperty(key []byte) ([]byte, error)

	WriteIteration(ir *Iteration) error
	ListIterations() ([]*Iteration, error)

	WriteOutput(utxo *Output, traceId string) error
	WriteOutputs(utxos []*Output, traceId string) error

	ListOutputsForTransaction(traceId string) ([]*Output, error)
	ListOutputsForAsset(groupId string, state, assetId string, limit int) ([]*Output, error)

	WriteAction(act *Action) error
	ListActions(limit int) ([]*UnifiedOutput, error)

	WriteTransaction(tx *Transaction) error
	ReadTransactionByTraceId(traceId string) (*Transaction, error)
	ReadTransactionByHash(hash crypto.Hash) (*Transaction, error)
	ListTransactions(state int, limit int) ([]*Transaction, error)
	DeleteTransaction(tx *Transaction) error

	WriteCollectibleOutput(utxo *CollectibleOutput, traceId string) error
	WriteCollectibleOutputs(utxos []*CollectibleOutput, traceId string) error
	ListCollectibleOutputsForTransaction(traceId string) ([]*CollectibleOutput, error)
	ListCollectibleOutputsForToken(state, tokenId string, limit int) ([]*CollectibleOutput, error)

	WriteCollectibleTransaction(traceId string, tx *CollectibleTransaction) error
	ReadCollectibleTransaction(traceId string) (*CollectibleTransaction, error)
	ReadCollectibleTransactionByHash(hash crypto.Hash) (*CollectibleTransaction, error)
	ListCollectibleTransactions(state int, limit int) ([]*CollectibleTransaction, error)
}

type Worker interface {
	// handle the output and a true return value interrupts workers loop
	ProcessOutput(context.Context, *Output) bool
	ProcessCollectibleOutput(context.Context, *CollectibleOutput) bool
}

package main

import (
	"context"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	ctx := context.Background()
	logger.SetLevel(logger.DEBUG)

	conn, err := ethclient.Dial(GethRPC)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	store, err := OpenStorage(DataPath)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	proxy := NewProxy(ctx, ProxyKeyStore, conn)
	go proxy.Run(ctx, store)

	StartHTTP(proxy, store)
}

package main

import "github.com/ethereum/go-ethereum/ethclient"

func main() {
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

	proxy := NewProxy(ProxyKeyStore, conn)
	proxy.Run(store)
}

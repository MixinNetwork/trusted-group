package main

import (
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/eos"
	"github.com/MixinNetwork/trusted-group/mvm/ethereum"
)

func main() {
	group, err := mtg.BuildGroup(nil, nil, nil)
	if err != nil {
		panic(err)
	}
	rw := eos.NewGroupReceiver()
	group.AddWorker(rw)
	erw := ethereum.NewGroupReceiver()
	group.AddWorker(erw)
	group.Run(nil)
}

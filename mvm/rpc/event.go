package rpc

import (
	"context"
	"fmt"

	"github.com/MixinNetwork/trusted-group/mvm/store"
)

func getEVMEvent(ctx context.Context, impl *RPC, params []any) (string, error) {
	if len(params) != 1 {
		return "", fmt.Errorf("invalid params count %d", len(params))
	}

	snap, err := readNetworkSnapshot(fmt.Sprint(params[0]))
	if err != nil {
		return "", err
	} else if snap == "" {
		return "", fmt.Errorf("no snapshot in kernel %v", params[0])
	}

	tx, err := readSnapshotTransaction(snap)
	if err != nil {
		return "", err
	} else if len(tx) != 64 {
		panic(snap)
	}

	event, err := impl.store.ReadGroupEvent(utxoId(tx, 0))
	if err != nil {
		return "", err
	} else if event == nil {
		return "", fmt.Errorf("event not found %s", tx)
	}

	address, err := findProcessAddress(impl.store, event.Process)
	if err != nil {
		return "", err
	} else if address == "" {
		return "", fmt.Errorf("process not found %s", event.Process)
	}

	return impl.engine.ReadGroupEventTransaction(address, event.Nonce)
}

func readNetworkSnapshot(id string) (string, error) {
	panic(0)
}

func readSnapshotTransaction(snap string) (string, error) {
	panic(0)
}

func utxoId(tx string, index int) string {
	panic(0)
}

func findProcessAddress(store *store.BadgerStore, pid string) (string, error) {
	procs, err := store.ListProcesses()
	if err != nil {
		return "", err
	}
	for _, p := range procs {
		if p.Identifier == pid {
			return p.Address, nil
		}
	}
	return "", nil
}

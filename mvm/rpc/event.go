package rpc

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/trusted-group/mvm/store"
	"github.com/gofrs/uuid"
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
	snapshot, err := bot.NetworkSnapshot(context.Background(), id)
	if err != nil {
		return "", err
	}
	return snapshot.SnapshotHash, nil
}

func readSnapshotTransaction(snap string) (string, error) {
	nodes := []string{
		"http://mixin-node0.exinpool.com:8239",
		"http://mixin-node-01.b1.run:8239",
		"http://lehigh.hotot.org:8239",
	}
	m := NewMixinNetwork(nodes[time.Now().Unix()%3])
	snapshot, err := m.GetSnapshot(snap)
	if err != nil {
		return "", err
	}
	return snapshot.Transaction.Hash, nil
}

func utxoId(tx string, index int) string {
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%s:%d", tx, index)))
	s := h.Sum(nil)
	s[6] = (s[6] & 0x0f) | 0x30
	s[8] = (s[8] & 0x3f) | 0x80
	sid, err := uuid.FromBytes(s)
	if err != nil {
		panic(err)
	}
	return sid.String()
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

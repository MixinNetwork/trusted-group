package main

import (
	"context"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/tip/messenger"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
)

func RunMonitor(ctx context.Context, messenger messenger.Messenger, store machine.Store) {
	startedAt := time.Now()

	for {
		time.Sleep(30 * time.Minute)
		msg, err := bundleMachineState(ctx, store, startedAt)
		if err != nil {
			logger.Verbosef("Monitor.bundleMachineState() => %v", err)
			continue
		}
		err = messenger.BroadcastMessage(ctx, msg)
		logger.Verbosef("Monitor.BroadcastMessage(%x) => %v", msg, err)
	}
}

func bundleMachineState(ctx context.Context, store machine.Store, startedAt time.Time) ([]byte, error) {
	state := fmt.Sprintf("Run time :%s\n", time.Now().Sub(startedAt).String())
	procs, err := store.ListProcesses()
	if err != nil {
		return nil, err
	}
	state = state + fmt.Sprintf("Total processes count: %d\n", len(procs))
	events, err := store.ListPendingGroupEvents(100)
	if err != nil {
		return nil, err
	}
	state = state + fmt.Sprintf("Pending group events count: %d", len(events))
	return []byte(state), nil
}

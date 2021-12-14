package rpc

import (
	"encoding/binary"
	"time"

	"github.com/MixinNetwork/trusted-group/mvm/store"
)

const (
	outputsDrainingKey = "outputs-draining-checkpoint"
)

func getInfo(store *store.BadgerStore) (map[string]interface{}, error) {
	odc, err := readDrainingCheckpoint(store, outputsDrainingKey)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"group": map[string]interface{}{
			"outputs": map[string]interface{}{
				"draining": odc,
			},
		},
	}, nil
}

func readDrainingCheckpoint(store *store.BadgerStore, key string) (time.Time, error) {
	val, err := store.ReadProperty([]byte(key))
	if err != nil || len(val) == 0 {
		return time.Time{}, err
	}
	ts := int64(binary.BigEndian.Uint64(val))
	return time.Unix(0, ts), nil
}

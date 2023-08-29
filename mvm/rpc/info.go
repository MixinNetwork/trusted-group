package rpc

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	"github.com/MixinNetwork/trusted-group/mvm/config"
	"github.com/MixinNetwork/trusted-group/mvm/crypto"
	"github.com/MixinNetwork/trusted-group/mvm/crypto/en256"
	"github.com/MixinNetwork/trusted-group/mvm/store"
	"github.com/drand/kyber"
	"github.com/drand/kyber/share"
)

const (
	outputsDrainingKey = "outputs-draining-checkpoint"
)

func getInfo(store *store.BadgerStore) (map[string]any, error) {
	odc, err := readDrainingCheckpoint(store, outputsDrainingKey)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"group": map[string]any{
			"outputs": map[string]any{
				"draining": odc,
			},
		},
	}, nil
}

func getMTGKeys(conf *config.Configuration) (map[string]string, error) {
	if conf == nil || conf.Machine == nil || conf.Machine.Poly == "" {
		return nil, errors.New("invalid config machine")
	}
	pb, err := hex.DecodeString(conf.Machine.Poly)
	if err != nil {
		return nil, err
	}

	commitments, err := unmarshalCommitments(pb)
	if err != nil {
		return nil, err
	}
	suite := en256.NewSuiteG2()
	poly := share.NewPubPoly(suite, suite.Point().Base(), commitments)
	return map[string]string{"mtg": poly.Commit().String()}, nil
}

func readDrainingCheckpoint(store *store.BadgerStore, key string) (time.Time, error) {
	val, err := store.ReadProperty([]byte(key))
	if err != nil || len(val) == 0 {
		return time.Time{}, err
	}
	ts := int64(binary.BigEndian.Uint64(val))
	return time.Unix(0, ts), nil
}

func unmarshalCommitments(b []byte) ([]kyber.Point, error) {
	var commits []kyber.Point
	for i, l := 0, len(b)/128; i < l; i++ {
		point, err := crypto.PubKeyFromBytes(b[i*128 : (i+1)*128])
		if err != nil {
			return commits, err
		}
		commits = append(commits, point)
	}
	return commits, nil
}

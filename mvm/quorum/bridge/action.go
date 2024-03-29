package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mtg"
	"github.com/gofrs/uuid/v5"
)

type Action struct {
	Destination string   `json:"destination,omitempty"`
	Tag         string   `json:"tag,omitempty"`
	Receivers   []string `json:"receivers,omitempty"`
	Threshold   int64    `json:"threshold,omitempty"`
	Extra       string   `json:"extra,omitempty"`
}

func decryptData(data string, collectible bool) ([]byte, error) {
	mep := mtg.DecodeMixinExtra(data)
	if mep == nil {
		return nil, fmt.Errorf("invalid mixin extra pack %s", data)
	}
	if collectible {
		return []byte(mep.M), nil
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(mep.M)
	if err != nil {
		return nil, err
	}
	// TODO encryption and decryption
	return b, nil
}

func (p *Proxy) decodeAction(u *User, memo, assetId string, collectible bool) (*Action, error) {
	b, err := decryptData(memo, collectible)
	logger.Verbosef("Proxy.decryptData(%s) => %x %v", memo, b, err)
	if err != nil || len(b) != 68 {
		return nil, nil
	}

	if uuid.FromBytesOrNil(b[:16]).String() != MVMRegistryId {
		return nil, nil
	}
	if hex.EncodeToString(b[16:36]) != strings.ToLower(MVMStorageContract[2:]) {
		return nil, nil
	}
	k := new(big.Int).SetBytes(b[36:])
	val, err := p.storage.Read(nil, k)
	if err != nil {
		return nil, err
	}
	logger.Verbosef("Proxy.storage.Read(%x) => %x %v", k.Bytes(), val, err)

	var act Action
	err = json.Unmarshal(val, &act)
	if err != nil {
		return nil, nil
	}
	logger.Verbosef("Proxy.decodeAction(%v, %s, %v) => %v %v", u, memo, collectible, act, err)
	if act.Destination != "" && collectible {
		return nil, nil
	}

	if act.Destination != "" {
		asset, err := store.readAsset(assetId)
		if err != nil {
			panic(err)
		}
		chainId := crypto.NewHash([]byte(asset.ChainID))
		if verifyDestination(chainId, act.Destination) != nil {
			return nil, nil
		}
		if len(act.Receivers) > 0 || act.Threshold != 0 {
			return nil, nil
		}
		return &act, nil
	}

	if len(act.Receivers) > 0 {
		if len(act.Receivers) > 7 {
			return nil, nil
		}
		if act.Threshold <= 0 || act.Threshold > int64(len(act.Receivers)) {
			return nil, nil
		}
		for _, r := range act.Receivers {
			if id, _ := uuid.FromString(r); id.String() != r {
				return nil, nil
			}
		}
		return &act, nil
	}

	return nil, nil
}

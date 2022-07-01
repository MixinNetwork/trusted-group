package main

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EIP191 recover sig to pub
func EcrecoverEIP191(address, sig string) (*common.Address, error) {
	data := []byte(fmt.Sprintf("MVM:Bridge:Proxy:%s:%s", CurvePublicKey(ServerPublic), address))
	data = []byte("0x" + hex.EncodeToString(crypto.Keccak256Hash(data).Bytes()))
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	hash := crypto.Keccak256Hash([]byte(msg))

	buf, err := hex.DecodeString(sig)
	if err != nil {
		return nil, err
	}
	if len(buf) != 65 {
		return nil, fmt.Errorf("invalid length of signture: %d", len(buf))
	}
	if buf[64] != 27 && buf[64] != 28 && buf[64] != 1 && buf[64] != 0 {
		return nil, fmt.Errorf("invalid signature type")
	}
	if buf[64] >= 27 {
		buf[64] -= 27
	}
	recoverPub, err := crypto.Ecrecover(hash.Bytes(), buf)
	if err != nil {
		return nil, err
	}
	pubKey, err := crypto.UnmarshalPubkey(recoverPub)
	if err != nil {
		return nil, err
	}
	addr := crypto.PubkeyToAddress(*pubKey)
	return &addr, nil
}

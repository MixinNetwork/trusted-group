package main

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestVerify(t *testing.T) {
	assert := assert.New(t)

	privateKey, err := crypto.HexToECDSA("0123456789012345678901234567890123456789012345678901234567890123")
	assert.Nil(err)

	source := "0xDFDF68C62D32063e1405911aE35a040F93D7A9C8"
	data := []byte(fmt.Sprintf("MVM:Bridge:Proxy:8MfEmL3g8s-PoDpZ4OcDCUDQPDiH4u1_OmxB0Aaknzg:%s", source))
	data = []byte("0x" + hex.EncodeToString(crypto.Keccak256Hash(data).Bytes()))
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	hash := crypto.Keccak256Hash([]byte(msg))
	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	assert.Nil(err)
	pub, err := crypto.Ecrecover(hash.Bytes(), signature)
	assert.Nil(err)
	pubKey, err := crypto.UnmarshalPubkey(pub)
	assert.Nil(err)
	address := crypto.PubkeyToAddress(*pubKey)
	assert.Equal("0x14791697260E4c9A71f18484C9f997B308e59325", address.Hex())

	buf, _ := hex.DecodeString("b5d480b5fd08b19e976f6d7b68aaf4460227e56d387ab9ef51112b31fcaf11eb3488dc5d92764b55862c4f5cca26294dd982ef7fd3cf0107e023000f6e9136761c")
	if buf[64] == 27 || buf[64] == 28 {
		buf[64] -= 27
	}
	pub, err = crypto.Ecrecover(hash.Bytes(), buf)
	assert.Nil(err)
	pubKey, err = crypto.UnmarshalPubkey(pub)
	assert.Nil(err)
	address = crypto.PubkeyToAddress(*pubKey)
	assert.Equal(source, address.Hex())
}

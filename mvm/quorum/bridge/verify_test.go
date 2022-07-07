package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fox-one/msgpack"
	"github.com/gofrs/uuid"
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

func TestMsgpack(t *testing.T) {
	assert := assert.New(t)

	var result map[string]interface{}
	buf, _ := hex.DecodeString("81a3666f6fa3626172")
	err := msgpack.Unmarshal(buf, &result)
	assert.Nil(err)
	assert.Equal("bar", result["foo"].(string))

	type mixinExtraPack struct {
		T uuid.UUID
		G string `msgpack:",omitempty"`
		M string `msgpack:",omitempty"`
	}

	buf, err = base64.RawURLEncoding.DecodeString("gqFUxBBuwL1_EcBD2pdeKorZ664LoU2sZ2FObWIyLWpZbUZ5")
	assert.Nil(err)

	buf, err = decryptData("gqFUxBBuwL1_EcBD2pdeKorZ664LoU2sZ2FObWIyLWpZbUZ5")
	assert.Nil(err)
	assert.Equal("gaNmb2-jYmFy", base64.RawURLEncoding.EncodeToString(buf))

	buf, err = decryptData("gqFUxBBuwL1_EcBD2pdeKorZ664LoU3ZW3ZXY0ljbmJPTW1PNU16cWpOLUlTcE84a0dZalJtSkwtVHY5Sk5TVmdoX1Q5eGV5cThBVk9Hb2NCVXBnSXVXZ2NmXzN2cjg1a3JjTEJTcVVlSGE4bmNKSDNIeGM")
	assert.Nil(err)
	assert.Equal("vWcIcnbOMmO5MzqjN-ISpO8kGYjRmJL-Tv9JNSVgh_T9xeyq8AVOGocBUpgIuWgcf_3vr85krcLBSqUeHa8ncJH3Hxc", base64.RawURLEncoding.EncodeToString(buf))
	assert.Equal(68, len(buf))
	assert.Equal(MVMRegistryId, uuid.FromBytesOrNil(buf[:16]).String())
	assert.Equal(hex.EncodeToString(buf[16:36]), strings.ToLower(MVMStorageContract[2:]))
}

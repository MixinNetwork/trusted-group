package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fox-one/msgpack"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/curve25519"
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

func TestAction(t *testing.T) {
	assert := assert.New(t)

	privbuf, _ := hex.DecodeString("9df4a2127eea10989282bcbc80b903ee9cc5a6d3fadf50b4b6fc6dcc3826c6899ddb7d881fe4a47283eff5c2fffc9046833b2b2726d82be33dd7e88060fadf58")
	pubbuf, _ := hex.DecodeString("9ddb7d881fe4a47283eff5c2fffc9046833b2b2726d82be33dd7e88060fadf58")
	priv := ed25519.PrivateKey(privbuf)
	pub := ed25519.PublicKey(pubbuf)

	key := SharedKey(pub[:])
	keyOther := testSharedKey(ServerPublic, priv.Seed())
	assert.Equal(hex.EncodeToString(key[:]), hex.EncodeToString(keyOther[:]))
	assert.Equal("866c29ba53b6bcdac9d981c5ff1aac246f4ea86799b52b7222d5cd776ea36f20", hex.EncodeToString(key[:]))

	action := &Action{
		Destination: "bc1qad5k5xjz0h0x97yc39j9ax0cyewhse605eqq8p",
		Tag:         "",
		Extra:       "This is memo",
	}

	buf, err := json.Marshal(action)
	assert.Nil(err)
	cipher := aesEncryptCBC(key[:], buf)
	result, err := aesDecryptCBC(key[:], cipher)
	assert.Nil(err)
	var actionNew Action
	err = json.Unmarshal(result, &actionNew)
	assert.Nil(err)
	assert.Equal(actionNew.Destination, action.Destination)

	data := "8337f27cf356e5a63f79d7362d5dbdafcbb826208ee22bc66f4861644a92a473904f081b2b8460b03b9cda1df1bdc57ffc5df9fa5a6940ffdcce25f9b470891cea885af9daf4e55e28ad73e2095e8384ab56e59f2e6478076dc1e7d377c0ade32992972c5cda1b6499615761029a844f"
	buf, err = hex.DecodeString(data)
	assert.Nil(err)
	result, err = aesDecryptCBC(key[:], buf)
	assert.Nil(err)
	err = json.Unmarshal(result, &actionNew)
	assert.Nil(err)
	assert.Equal(actionNew.Destination, action.Destination)
	assert.Equal(actionNew.Extra, action.Extra)
}

func testSharedKey(public string, seed []byte) [32]byte {
	buf, err := hex.DecodeString(public)
	if err != nil {
		panic(err)
	}

	curve25519Public, err := PublicKeyToCurve25519(ed25519.PublicKey(buf))
	if err != nil {
		panic(err)
	}

	var dst, priv, pub [32]byte
	PrivateKeyToCurve25519(&priv, seed)
	copy(pub[:], curve25519Public[:])
	curve25519.ScalarMult(&dst, &priv, &pub)
	return dst
}

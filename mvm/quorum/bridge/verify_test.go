package main

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/assert"
)

func TestVerify(t *testing.T) {
	assert := assert.New(t)

	types := apitypes.Types{
		"EIP712Domain": []apitypes.Type{},
		"Message": []apitypes.Type{
			apitypes.Type{
				Name: "data",
				Type: "string",
			},
		},
	}
	domain := apitypes.TypedDataDomain{}
	primaryType := "Message"
	message := apitypes.TypedDataMessage{
		"data": "test",
	}
	data := apitypes.TypedData{
		Types:       types,
		PrimaryType: primaryType,
		Domain:      domain,
		Message:     message,
	}

	buf := EIP712Hash(data)
	assert.Equal("db1e257f42232aee5d38ee5c6e1edc097011c0c242365b063ea863ce11364c1a", hex.EncodeToString(buf))

	sig, _ := hex.DecodeString("f6cda8eaf5137e8cc15d48d03a002b0512446e2a7acbc576c01cfbe40ad9345663ccda8884520d98dece9a8bfe38102851bdae7f69b3d8612b9808e6337801601b")
	hash, _ := hex.DecodeString("db1e257f42232aee5d38ee5c6e1edc097011c0c242365b063ea863ce11364c1a")
	address, err := Ecrecover(hash, sig)
	assert.Nil(err)
	assert.Equal("0x29C76e6aD8f28BB1004902578Fb108c507Be341b", address.Hex())

	buf = MessageHash("0x29C76e6aD8f28BB1004902578Fb108c507Be341b")
	assert.Equal("906b4645eee4ace95fca8a4a02124f78be8a4a6f8f178f40128a86d6a7233023", hex.EncodeToString(buf))

	privateKey, err := crypto.HexToECDSA("0123456789012345678901234567890123456789012345678901234567890123")
	assert.Nil(err)

	dat := []byte("MVM:Bridge:Proxy:N5qVP3ipty4a-K4gO9t_86Nb6rmN3BmAz6MfnsLb_6E:0x12266b2BbdEAb152f8A0CF83c3997Bc8dbAD0be0")
	dat = []byte("0x" + hex.EncodeToString(crypto.Keccak256Hash(dat).Bytes()))
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(dat), dat)
	hsh := crypto.Keccak256Hash([]byte(msg))
	signature, err := crypto.Sign(hsh.Bytes(), privateKey)
	assert.Nil(err)
	pub, err := crypto.Ecrecover(hsh.Bytes(), signature)
	assert.Nil(err)
	pubKey, err := crypto.UnmarshalPubkey(pub)
	assert.Nil(err)
	address = crypto.PubkeyToAddress(*pubKey)
	assert.Equal(address.Hex(), "0x14791697260E4c9A71f18484C9f997B308e59325")

	buf, _ = hex.DecodeString("a25788c4f62f24d3d58aa029c08cbce1c7746a0cd2c206ee8bebfbf4a26f603f2ae7e47d45735840ba0297a3a3637d4225626606459fff5bc7ae36de369ad5431c")
	if buf[64] == 27 || buf[64] == 28 {
		buf[64] -= 27
	}
	pub, err = crypto.Ecrecover(hsh.Bytes(), buf)
	assert.Nil(err)
	pubKey, err = crypto.UnmarshalPubkey(pub)
	assert.Nil(err)
	address = crypto.PubkeyToAddress(*pubKey)
	assert.Equal(address.Hex(), "0xaE9ADd61e9Fa5c203a5139DBe3dfC86c838C1239")
}

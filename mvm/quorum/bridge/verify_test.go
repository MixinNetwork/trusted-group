package main

import (
	"encoding/hex"
	"testing"

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
}

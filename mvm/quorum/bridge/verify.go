package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// MVM || Bridge || Proxy || ServerPublicKey(in config.go) || 0x123...ABC
func MessageHash(address string) ([]byte, error) {
	msg := apitypes.TypedDataMessage{
		"data": fmt.Sprintf("MVM:Bridge:Proxy:%s:%s", ServerPublic, address),
	}
	types := apitypes.Types{
		"EIP712Domain": []apitypes.Type{},
		"Message": []apitypes.Type{
			apitypes.Type{
				Name: "data",
				Type: "string",
			},
		},
	}
	typed := apitypes.TypedData{
		Types:       types,
		PrimaryType: "Message",
		Message:     msg,
	}
	return EIP712Hash(typed)
}

func EIP712Hash(typedData apitypes.TypedData) ([]byte, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, err
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	return crypto.Keccak256(rawData), nil
}

func Ecrecover(hash, signature []byte) (*common.Address, error) {
	sig := make([]byte, len(signature))
	copy(sig, signature)
	if len(sig) != 65 {
		return nil, fmt.Errorf("invalid length of signture: %d", len(sig))
	}

	if sig[64] != 27 && sig[64] != 28 && sig[64] != 1 && sig[64] != 0 {
		return nil, fmt.Errorf("invalid signature type")
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	recoverPub, err := crypto.Ecrecover(hash, sig)
	if err != nil {
		return nil, fmt.Errorf("can not ecrecover: %v", err)
	}
	pubKey, err := crypto.UnmarshalPubkey(recoverPub)
	if err != nil {
		return nil, fmt.Errorf("can not unmarshal pubkey: %v", err)
	}

	address := crypto.PubkeyToAddress(*pubKey)
	return &address, nil
}

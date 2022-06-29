package main

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func recoverTypedSignature(signature string) {
	// buf,err:=	hex.DecodeString(signature)
}

// MVM || Bridge || Proxy || ServerPublicKey(in config.go) || 0x123...ABC
func MessageHash(address string) []byte {
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

func Ecrecover(hash, signature []byte) (common.Address, error) {
	var address common.Address
	sig := make([]byte, len(signature))
	copy(sig, signature)
	if len(sig) != 65 {
		return address, fmt.Errorf("invalid length of signture: %d", len(sig))
	}

	if sig[64] != 27 && sig[64] != 28 && sig[64] != 1 && sig[64] != 0 {
		return address, fmt.Errorf("invalid signature type")
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	recoverPub, err := crypto.Ecrecover(hash, sig)
	if err != nil {
		return address, fmt.Errorf("can not ecrecover: %v", err)
	}
	pubKey, err := crypto.UnmarshalPubkey(recoverPub)
	if err != nil {
		return address, fmt.Errorf("can not unmarshal pubkey: %v", err)
	}

	return crypto.PubkeyToAddress(*pubKey), nil
}

// sanitizeData doesn't need
// only supports version V4
func EIP712Hash(typedData apitypes.TypedData) []byte {
	domainSeparator := hashStruct("EIP712Domain", typedData.Domain.Map(), typedData.Types, "V4")
	typedDataHash := hashStruct(typedData.PrimaryType, typedData.Message, typedData.Types, "V4")
	rawData := []byte{0x19, 0x1}
	rawData = append(rawData, domainSeparator...)
	rawData = append(rawData, typedDataHash...)
	return crypto.Keccak256(rawData)
}

// https://docs.metamask.io/guide/signing-data.html#signing-data-with-metamask
// only signTypedData_v4 supports
func hashStruct(primaryType string, data apitypes.TypedDataMessage, types apitypes.Types, version string) []byte {
	return crypto.Keccak256(encodeData(primaryType, data, types, version))
}

// for bytes32
// rawEncode combine the output only
func encodeData(primaryType string, data map[string]interface{}, types apitypes.Types, version string) []byte {
	output := hashType(primaryType, types)
	for _, field := range types[primaryType] {
		buf := encodeField(types, field.Name, field.Type, data[field.Name], version)
		output = append(output, buf...)
	}
	return output
}

func encodeField(types apitypes.Types, name string, field string, value interface{}, version string) []byte {
	if len(types[field]) != 0 {
		if version == "V4" && value == nil {
			return bytes.Repeat([]byte{0x0}, 32)
		}
	}

	if field == "string" {
		v := value.(string)
		return crypto.Keccak256([]byte(v))
	}
	return []byte{}
}

func hashType(primaryType string, types apitypes.Types) []byte {
	return crypto.Keccak256([]byte(encodeType(primaryType, types)))
}

func encodeType(primaryType string, types apitypes.Types) string {
	unsortedDeps := findTypeDependencies(primaryType, types, []string{})
	k := -1
	for i, dep := range unsortedDeps {
		if dep == primaryType {
			k = i
			break
		}
	}
	var deps []string
	// delete primaryType from unsortedDeps
	if k > -1 {
		deps = append(deps, unsortedDeps[:k]...)
		deps = append(deps, unsortedDeps[k+1:]...)
	}
	sort.Strings(deps)
	deps = append([]string{primaryType}, deps...)

	var result string
	for _, dep := range deps {
		children := types[dep]

		var params []string
		for _, child := range children {
			params = append(params, fmt.Sprintf("%s %s", child.Type, child.Name))
		}
		result += fmt.Sprintf("%s(%s)", dep, strings.Join(params, ","))
	}
	return result
}

// [primaryType] = primaryType.match(/^\w*/u);
func findTypeDependencies(primaryType string, types apitypes.Types, results []string) []string {
	reg := regexp.MustCompile(`^\w*`)
	primaryType = reg.FindString(primaryType)
	for _, r := range results {
		if r == primaryType {
			return results
		}
	}
	if len(types[primaryType]) == 0 {
		return results
	}

	results = append(results, primaryType)
	for _, v := range types[primaryType] {
		results = findTypeDependencies(v.Type, types, results)
	}
	return results
}

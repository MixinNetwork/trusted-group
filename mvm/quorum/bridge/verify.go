package main

import (
	"fmt"
	"regexp"
	"sort"

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

// sanitizeData doesn't need
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

// https://docs.metamask.io/guide/signing-data.html#signing-data-with-metamask
// only signTypedData_v4 supports
func hashStruct(primaryType string, data apitypes.TypedDataMessage, types apitypes.Types, version string) []byte {
	return crypto.Keccak256([]byte(encodeData(primaryType, data, types, version)))
}

func encodeData(primaryType string, data map[string]interface{}, types apitypes.Types, version string) string {
	var output string
	for _, field := range types[primaryType] {
		arrays := encodeField(types, field.Name, field.Type, data[field.Name], version)
		output += arrays[1]
	}
	return output
}

func encodeField(types apitypes.Types, name string, field string, value interface{}, version string) []string {
	if len(types[field]) != 0 {
		if version == "V4" && value == nil {
			// TODO version === SignTypedDataVersion.V4 && value == null ? '0x0000000000000000000000000000000000000000000000000000000000000000' : keccak(encodeData(type, value, types, version)),
			return []string{"bytes32", "0x0000000000000000000000000000000000000000000000000000000000000000"}
		}
		// TODO
	}

	if field == "bytes" {
		v := value.([]byte)
		return []string{"bytes32", string(crypto.Keccak256(v))}
	}

	if field == "string" {
		v := value.(string)
		return []string{"bytes32", string(crypto.Keccak256([]byte(v)))}
	}
	return []string{}
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
		result += fmt.Sprintf("%s(%s)", dep, params)
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

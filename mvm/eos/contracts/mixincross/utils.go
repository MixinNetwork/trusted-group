package main

import (
	"github.com/uuosio/chain"
	"github.com/uuosio/chain/hex"
)

//table signers
type Signer struct {
	account    chain.Name //primary : t.account.N
	public_key chain.PublicKey
}

func VerifySignatures(data []byte, signatures []chain.Signature) bool {
	digest := chain.Sha256(data)
	signerDB := NewSignerDB(MTG_XIN, MTG_XIN)
	signers := make([]*Signer, 0, 10)
	it := signerDB.Lowerbound(0)
	for it.IsOk() {
		item, _ := signerDB.GetByIterator(it)
		signers = append(signers, item)
		it, _ = signerDB.Next(it)
	}

	threshold := len(signers)/3*2 + 1
	validSignatures := 0

	verfiedSignatures := make([]*chain.Signature, 0, len(signers))

	for i := 0; i < len(signatures); i++ {
		signature := signatures[i]
		CheckDuplicatedSignature(verfiedSignatures, &signature)
		verfiedSignatures = append(verfiedSignatures, &signature)

		pub_key := chain.RecoverKey(digest, &signature)
		for _, signer := range signers {
			if signer.public_key == *pub_key {
				validSignatures += 1
				break
			}
		}
		if validSignatures >= threshold {
			return true
		}
	}
	check(false, "Not enough valid signatures")
	return false
}

func CheckDuplicatedSignature(signatures []*chain.Signature, signature *chain.Signature) {
	for _, sig := range signatures {
		if *sig == *signature {
			check(false, "duplicated signature")
		}
	}
}

func check(b bool, msg string) {
	chain.Check(b, msg)
}

func Uint128ToString(uint128 chain.Uint128) string {
	return hex.EncodeToString(uint128[:])
}

func GetClientId(memo string) (*chain.Uint128, bool) {
	if len(memo) != 36 {
		return nil, false
	}

	client_id := make([]byte, 0, 32)
	for _, c := range memo {
		if c != '-' {
			client_id = append(client_id, byte(c))
		}
	}

	if len(client_id) != 32 {
		return nil, false
	}

	h, err := hex.DecodeString(string(client_id))
	if err != nil {
		return nil, false
	}
	out := new(chain.Uint128)
	copy(out[:], h)
	return out, true
}

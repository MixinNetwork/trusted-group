package main

import (
	"github.com/uuosio/chain"
)

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
		CheckDumplicatedSignature(verfiedSignatures, &signature)
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

func CheckDumplicatedSignature(signatures []*chain.Signature, signature *chain.Signature) {
	for _, sig := range signatures {
		if *sig == *signature {
			check(false, "dumplicated signature")
		}
	}
}

func check(b bool, msg string) {
	chain.Check(b, msg)
}

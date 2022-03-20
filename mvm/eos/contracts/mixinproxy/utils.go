package main

import (
	"github.com/uuosio/chain"
	"github.com/uuosio/chain/hex"
)

const (
	alphabet = "abcdefghijklmnopqrstuvwxyz12345"
)

//table signers ignore
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
		item := signerDB.GetByIterator(it)
		signers = append(signers, item)
		it, _ = signerDB.Next(it)
	}

	threshold := len(signers)*2/3 + 1
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

func assert(b bool, msg string) {
	chain.Assert(b, msg)
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

func GetAccountNameFromId(accountId uint64) chain.Name {
	strName := []byte{'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a', 'm', 'v', 'm'}
	for i := 0; i < 9; i++ {
		j := accountId % 31
		strName[i] = alphabet[j]
		accountId /= 31
		if accountId == 0 {
			break
		}
	}

	for i, j := 0, 9-1; i < j; i, j = i+1, j-1 {
		strName[i], strName[j] = strName[j], strName[i]
	}

	name := chain.NewName(string(strName))
	if !DEBUG {
		check(!chain.IsAccount(name), "account name already exists")
	}
	return name
}

func CreateNewAccount(creator chain.Name, ownerAccount chain.Name, newAccount chain.Name) {
	if DEBUG {
		if chain.IsAccount(newAccount) {
			return
		}
	}
	account := NewAccount{}
	account.Creator = creator
	account.Name = newAccount

	account.Owner.Threshold = 1
	account.Owner.Accounts = []PermissionLevelWeight{
		PermissionLevelWeight{
			PermissionLevel{
				Actor:      ownerAccount,
				Permission: chain.NewName("active"),
			},
			1,
		},
	}

	account.Active.Threshold = 1
	account.Active.Accounts = []PermissionLevelWeight{
		PermissionLevelWeight{
			PermissionLevel{
				Actor:      creator,
				Permission: chain.NewName("active"),
			},
			1,
		},
		PermissionLevelWeight{
			PermissionLevel{
				Actor:      creator,
				Permission: chain.NewName("multisig"),
			},
			1,
		},
	}

	chain.NewAction(
		&chain.PermissionLevel{Actor: creator, Permission: chain.ActiveName},
		chain.EosioContractName,
		chain.NewName("newaccount"),
		&account,
	).Send()

	// Payer    chain.Name
	// Receiver chain.Name
	// Quant    chain.Asset

	// buyRam := BuyRam{m.self, newAccountName, paid}
	chain.NewAction(
		&chain.PermissionLevel{Actor: creator, Permission: chain.ActiveName},
		chain.EosioContractName,
		chain.NewName("buyrambytes"),
		creator,
		newAccount,
		RAM_BYTES,
	).Send()
}

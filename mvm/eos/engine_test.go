package eos

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/learnforpractice/goeoslib/chain"
	"github.com/learnforpractice/goeoslib/crypto/secp256k1"
)

func TestBuildEventTx(t *testing.T) {
	// Process   string
	// Asset     string
	// Members   []string // need to do user mask per process
	// Threshold int
	// Amount    common.Integer
	// Extra     []byte
	// Timestamp uint64
	// Nonce     uint64
	// Signature []byte
	event := &encoding.Event{
		Process:   "49b00892-6954-4826-aaec-371ca165558a",
		Asset:     "49b00892-6954-4826-aaec-371ca165558a",
		Members:   []string{"49b00892-6954-4826-aaec-371ca165558a"},
		Threshold: 1,
		Amount:    common.NewInteger(1),
		Extra:     []byte("000000f8ce8880544eea5070c36d8549b45879219220695b97233d1b280c3c4a"),
		Timestamp: 1,
		Nonce:     0,
		Signature: []byte("signature"),
	}
	tx, err := BuildEventTransaction("mixin", "publisher", "hello", event, "0d38a4099ad044cf4fa7874752a55806f9f50b43953ffc760bc82c1f1fce65c8")
	if err != nil {
		panic(err)
	}
	t.Logf("%x", tx.Pack())
}

func TestUnpackTx(t *testing.T) {
	a := "b646a461f8004eea50700000000001305d67594ed759960000c60ad15b533201305d67594ed7599600000000a8ed32321810428a97721aa36a49b0089269544826aaec371ca165558a00"
	b := "2041a461f8004eea5070000000000100000000000000000000c60ad15b533201000000000000000000000000a8ed32321810428a97721aa36a49b0089269544826aaec371ca165558a00"
	_a, err := hex.DecodeString(a)
	if err != nil {
		panic(err)
	}
	_b, err := hex.DecodeString(b)
	if err != nil {
		panic(err)
	}

	t1 := chain.Transaction{}
	t1.Unpack(_a)
	if bytes.Compare(t1.Pack(), _a) != 0 {
		panic("pack after unpack went wrong...")
	}

	for _, a := range t1.Actions {
		t.Logf("a.Account: %s", a.Account.String())
		t.Logf("a.Name: %s", a.Name.String())
	}

	t2 := chain.Transaction{}
	t2.Unpack(_b)

	b1, _ := json.Marshal(t1)
	b2, _ := json.Marshal(t2)
	t.Logf("%s", string(b1))
	t.Logf("%s", string(b2))
}

func TestVerifyTx(t *testing.T) {
	a := "0566a461f8004eea50700000000001305d67594ed759960000c60ad15b533201305d67594ed7599600000000a8ed32321810428a97721aa36a49b0089269544826aaec371ca165558a00"
	_a, err := hex.DecodeString(a)
	if err != nil {
		panic(err)
	}
	t1 := chain.Transaction{}
	t1.Unpack(_a)
	chainId, err := chain.NewBytes32FromHex("8a34ec7df1b8cd06ff4a8abbaa7cc50300823350cadc59ab296cb00d104d2b8f")
	if err != nil {
		panic(err)
	}
	signature, err := secp256k1.NewSignatureFromBase58("SIG_K1_KdztqLe6fdwt2HHt4WpGmnkNu6SUvT2ezXMTPE2DqpUmGz286roz9oy693WajD9CvnSAueU9BKcay61dxPrdvkNzMphxXg")
	if err != nil {
		panic(err)
	}
	digest := t1.Id(chainId)
	pub, err := secp256k1.Recover(digest[:], signature)
	if err != nil {
		panic(err)
	}
	t.Logf("%s", pub.StringEOS())
}

func TestTx(t *testing.T) {
	chain.GetWallet().Import("test", "5Jbb4wuwz8MAzTB9FJNmrVYGXo4ABb7wqPVoWGcZ6x8V2FwNeDo")

	api := chain.NewChainApi("http://127.0.0.1:9000")

	expiration := uint32(time.Now().UnixNano()/1e9 + 3*60)
	tx := chain.NewTransaction(expiration)
	tx.SetReferenceBlock("000000f8ce8880544eea5070c36d8549b45879219220695b97233d1b280c3c4a")

	var action *chain.Action
	action = chain.NewAction(
		chain.PermissionLevel{Actor: chain.NewName("helloworld11"), Permission: chain.NewName("active")},
		chain.NewName("helloworld11"),
		chain.NewName("hello"),
	)
	tx.AddAction(action)
	r, err := api.PushAction(action)
	if err != nil {
		panic(err)
	}
	r2, _ := json.MarshalIndent(r, "", "  ")
	t.Logf("%s", r2)
}

func TestMultisig(t *testing.T) {
	api := chain.NewChainApi("http://127.0.0.1:9000")

	chain.GetWallet().Import("test", "5Jbb4wuwz8MAzTB9FJNmrVYGXo4ABb7wqPVoWGcZ6x8V2FwNeDo")
	keys := []string{"5JpXLb1tqxJB3Xtzd584xTdqKAzBnQ4TkqfEtT5QPotuv7Yt2bX",
		"5J2VCfZgiB6g86NfBmJE73yGRpvZXqS2UR7ZDGjt8XU2LPRddiY",
		"5J3hATRW5G6GT3pkSuyxrPw9KTU4hmbwX9WKddUSLwMcwQSLYex",
		"5K2Up4bcBo6BgDpfgccNmpgRE33nrvKsxfMgAauTMzHAqyN8SxM",
	}
	privs := []*secp256k1.PrivateKey{}
	for _, key := range keys {
		priv, err := secp256k1.NewPrivateKeyFromBase58(key)
		if err != nil {
			panic(err)
		}
		privs = append(privs, priv)
	}

	mixincontract := "mtgxinmtgxin"
	action := chain.NewAction(
		chain.PermissionLevel{Actor: chain.NewName(mixincontract), Permission: chain.NewName("active")},
		chain.NewName(mixincontract),
		chain.NewName("addprocess"),
		chain.NewName("helloworld11"),
		&chain.Uint128{},
	)
	expiration := uint32(time.Now().UnixNano()/1e9 + 3*60)
	tx := chain.NewTransaction(expiration)
	tx.SetReferenceBlock("000000f8ce8880544eea5070c36d8549b45879219220695b97233d1b280c3c4a")
	tx.AddAction(action)

	chainId, err := chain.NewBytes32FromHex("8a34ec7df1b8cd06ff4a8abbaa7cc50300823350cadc59ab296cb00d104d2b8f")
	if err != nil {
		panic(err)
	}

	signatures := []string{}
	for _, priv := range privs[:3] {
		sig, err := tx.Sign(priv, chainId)
		if err != nil {
			panic(err)
		}
		signatures = append(signatures, sig.String())
	}

	r, err := api.PushTransaction(tx, signatures, false)
	if err != nil {
		panic(err)
	}
	r2, _ := json.MarshalIndent(r, "", "  ")
	t.Logf("%s", r2)
}

func TestMultisigWithBuildEventTx(t *testing.T) {
	api := chain.NewChainApi("http://127.0.0.1:9000")

	chain.GetWallet().Import("test", "5Jbb4wuwz8MAzTB9FJNmrVYGXo4ABb7wqPVoWGcZ6x8V2FwNeDo")
	keys := []string{"5JpXLb1tqxJB3Xtzd584xTdqKAzBnQ4TkqfEtT5QPotuv7Yt2bX",
		"5J2VCfZgiB6g86NfBmJE73yGRpvZXqS2UR7ZDGjt8XU2LPRddiY",
		"5J3hATRW5G6GT3pkSuyxrPw9KTU4hmbwX9WKddUSLwMcwQSLYex",
		"5K2Up4bcBo6BgDpfgccNmpgRE33nrvKsxfMgAauTMzHAqyN8SxM",
	}
	privs := []*secp256k1.PrivateKey{}
	for _, key := range keys {
		priv, err := secp256k1.NewPrivateKeyFromBase58(key)
		if err != nil {
			panic(err)
		}
		privs = append(privs, priv)
	}

	event := &encoding.Event{
		Process:   "49b00892-6954-4826-aaec-371ca165558b",
		Asset:     "49b00892-6954-4826-aaec-371ca165558a",
		Members:   []string{"49b00892-6954-4826-aaec-371ca165558a"},
		Threshold: 1,
		Amount:    common.NewInteger(1),
		Extra:     []byte("000000f8ce8880544eea5070c36d8549b45879219220695b97233d1b280c3c4a"),
		Timestamp: uint64(time.Now().UnixNano()),
		Nonce:     0,
		Signature: []byte("signature"),
	}
	tx, err := BuildEventTransaction("mtgxinmtgxin", "mtgpublisher", "helloworld12", event, "0d38a4099ad044cf4fa7874752a55806f9f50b43953ffc760bc82c1f1fce65c8")
	t.Logf("++++tx.Pack() %x\n", tx.Pack())

	chainId, err := chain.NewBytes32FromHex("8a34ec7df1b8cd06ff4a8abbaa7cc50300823350cadc59ab296cb00d104d2b8f")
	if err != nil {
		panic(err)
	}

	signatures := []string{}
	for _, priv := range privs[:3] {
		sig, err := tx.Sign(priv, chainId)
		if err != nil {
			panic(err)
		}
		signatures = append(signatures, sig.String())
	}

	r, err := api.PushTransaction(tx, signatures, false)
	if err != nil {
		panic(err)
	}
	r2, _ := json.MarshalIndent(r, "", "  ")
	t.Logf("%s", r2)
}

func TestMultisigWithRawTx(t *testing.T) {
	api := chain.NewChainApi("http://127.0.0.1:9000")

	chain.GetWallet().Import("test", "5Jbb4wuwz8MAzTB9FJNmrVYGXo4ABb7wqPVoWGcZ6x8V2FwNeDo")
	keys := []string{"5JpXLb1tqxJB3Xtzd584xTdqKAzBnQ4TkqfEtT5QPotuv7Yt2bX",
		"5J2VCfZgiB6g86NfBmJE73yGRpvZXqS2UR7ZDGjt8XU2LPRddiY",
		"5J3hATRW5G6GT3pkSuyxrPw9KTU4hmbwX9WKddUSLwMcwQSLYex",
		"5K2Up4bcBo6BgDpfgccNmpgRE33nrvKsxfMgAauTMzHAqyN8SxM",
	}
	privs := []*secp256k1.PrivateKey{}
	for _, key := range keys {
		priv, err := secp256k1.NewPrivateKeyFromBase58(key)
		if err != nil {
			panic(err)
		}
		privs = append(privs, priv)
	}

	rawTx, _ := hex.DecodeString("b763a461f8004eea50700000000001305d67594ed759960000c60ad15b533201305d67594ed7599600000000a8ed32321810428a97721aa36a49b0089269544826aaec371ca165558a00")
	tx := &chain.Transaction{}
	tx.Unpack(rawTx)

	chainId, err := chain.NewBytes32FromHex("8a34ec7df1b8cd06ff4a8abbaa7cc50300823350cadc59ab296cb00d104d2b8f")
	if err != nil {
		panic(err)
	}

	signatures := []string{}
	for _, priv := range privs[:4] {
		sig, err := tx.Sign(priv, chainId)
		if err != nil {
			panic(err)
		}
		signatures = append(signatures, sig.String())
	}

	r, err := api.PushTransaction(tx, signatures, false)
	if err != nil {
		panic(err)
	}
	r2, _ := json.MarshalIndent(r, "", "  ")
	t.Logf("%s", r2)
}

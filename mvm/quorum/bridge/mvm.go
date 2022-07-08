package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

func (u *User) bindAndPass(ctx context.Context, p *Proxy, snapshotId, addr, asset string, amount decimal.Decimal) error {
	trace := mixin.UniqueConversationID(snapshotId, "BIND||PASS")
	extra := u.buildExtra(p, addr, asset, amount)
	op := &encoding.Operation{
		Purpose: encoding.OperationPurposeGroupEvent,
		Process: MVMRegistryId,
		Extra:   extra,
	}
	input := mixin.TransferInput{
		AssetID: asset,
		Amount:  amount,
		TraceID: trace,
	}
	input.OpponentMultisig.Receivers = MVMMembers
	input.OpponentMultisig.Threshold = uint8(MVMThreshold)
	input.Memo = base64.RawURLEncoding.EncodeToString(op.Encode())
	return u.send(ctx, &input)
}

func (u *User) buildExtra(p *Proxy, addr, asset string, amt decimal.Decimal) []byte {
	bind, pass := "81bac14f", "0ed1db9f"
	contract := strings.ToLower(MVMBridgeContract[2:])
	addr = "000000000000000000000000" + strings.ToLower(addr[2:])
	first := fmt.Sprintf("%04x", len(bind+addr)/2)
	amount := convertToMVMHex(amt)
	second := fmt.Sprintf("%04x", len(pass+addr+amount)/2)
	if u.checkBind(p) {
		extra := "0001" + contract + second + pass
		extra = extra + convertToMVMAddress(asset) + amount
		b, _ := hex.DecodeString(extra)
		if len(b) != 92 {
			panic(extra)
		}
		return b
	}

	extra := "0002" + contract + first + bind + addr
	extra = extra + contract + second + pass
	extra = extra + convertToMVMAddress(asset) + amount
	b, _ := hex.DecodeString(extra)
	if len(b) != 150 {
		panic(extra)
	}
	return p.buildHash(b)
}

func (p *Proxy) buildHash(b []byte) []byte {
	pid, err := uuid.FromString(MVMRegistryId)
	if err != nil {
		panic(err)
	}
	extra := pid.Bytes()

	cb, err := hex.DecodeString(MVMStorageContract[2:])
	if err != nil {
		panic(err)
	}
	extra = append(extra, cb...)

	k := new(big.Int).SetBytes(crypto.Keccak256(b))
	kbuf := make([]byte, 32)
	k.FillBytes(kbuf)
	extra = append(extra, kbuf...)

	if len(extra) != 68 {
		panic(hex.EncodeToString(extra))
	}

	o, err := p.storage.Read(nil, k)
	if err != nil {
		panic(err)
	}
	if bytes.Compare(o, b) == 0 {
		return extra
	}
	_, err = p.storage.Write(p.signer, k, b)
	if err != nil {
		panic(err)
	}
	return extra
}

func encodeActionAsExtra(pub []byte, a *Action) (string, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return "", err
	}

	hr := uuid.FromStringOrNil(MVMRegistryId).Bytes()
	ha, err := hex.DecodeString(MVMStorageContract[2:])
	if err != nil {
		panic(err)
	}
	key := SharedKey(pub)
	cipher := aesEncryptCBC(key[:], b)
	buf := append(pub, cipher...)
	hk := crypto.Keccak256(buf)

	extra := append(hr, ha...)
	extra = append(extra, hk...)
	extra = append(extra, b...)
	return hex.EncodeToString(extra), nil
}

func (u *User) getContract(p *Proxy) (common.Address, error) {
	uid, err := uuid.FromString(u.UserID)
	if err != nil {
		panic(err)
	}
	kb := []byte{0x0, 0x1}
	kb = append(kb, uid.Bytes()...)
	kb = append(kb, 0x0, 0x1)
	k := new(big.Int).SetBytes(crypto.Keccak256(kb))
	ua, err := p.registry.Contracts(nil, k)
	if err != nil {
		panic(err)
	}
	return ua, nil
}

func (u *User) checkBind(p *Proxy) bool {
	ua, err := u.getContract(p)
	if err != nil {
		panic(err)
	}
	if ua.String() == "0x0000000000000000000000000000000000000000" {
		return false
	}
	ba, err := p.bridge.Bridges(nil, ua)
	if err != nil {
		panic(err)
	}
	return ba.String() != "0x0000000000000000000000000000000000000000"
}

func convertToMVMHex(amount decimal.Decimal) string {
	buf := make([]byte, 32)
	amount = amount.Mul(decimal.NewFromInt(100000000))
	amount.BigInt().FillBytes(buf)
	return hex.EncodeToString(buf)
}

func convertToMVMAddress(asset string) string {
	aid, err := uuid.FromString(asset)
	if err != nil {
		panic(err)
	}
	return "000000000000000000000000" + "00000000" + hex.EncodeToString(aid.Bytes())
}

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/MixinNetwork/trusted-group/mvm/encoding"
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
	extra := "0002" + contract + first + bind + addr
	amount := convertToMVMHex(amt)
	second := fmt.Sprintf("%04x", len(pass+addr+amount)/2)
	extra = extra + contract + second + pass
	extra = extra + convertToMVMAddress(asset) + amount
	b, _ := hex.DecodeString(extra)
	if len(b) != 150 {
		panic(extra)
	}
	k := new(big.Int).SetBytes(crypto.Keccak256(b))
	o, err := p.proc.Read(nil, k)
	if err != nil {
		panic(err)
	}
	if bytes.Compare(o, b) != 0 {
		return p.buildHash(k, b)
	}
	extra = "0001" + contract + second + pass
	extra = extra + convertToMVMAddress(asset) + amount
	b, _ = hex.DecodeString(extra)
	if len(b) != 92 {
		panic(extra)
	}
	return b
}

func (p *Proxy) buildHash(k *big.Int, b []byte) []byte {
	_, err := p.proc.Write(p.signer, k, b)
	if err != nil {
		panic(err)
	}
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

	kbuf := make([]byte, 32)
	k.FillBytes(kbuf)
	extra = append(extra, kbuf...)

	if len(extra) != 68 {
		panic(hex.EncodeToString(extra))
	}
	return extra
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

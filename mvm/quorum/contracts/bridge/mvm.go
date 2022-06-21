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

func (p *Proxy) bindAndPass(snapshotId, addr, asset string, amount decimal.Decimal) error {
	ctx := context.Background()

	trace := mixin.UniqueConversationID(snapshotId, "BIND||PASS")
	extra := p.buildExtra(addr, asset, amount)
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
	_, err := p.Transaction(ctx, &input, ProxyPIN)
	return err
}

func (p *Proxy) buildExtra(addr, asset string, amt decimal.Decimal) []byte {
	bind, pass := "81bac14f", "0ed1db9f"
	contract := strings.ToLower(MVMBridgeContract[2:])
	addr = "000000000000000000000000" + strings.ToLower(addr[2:])
	first := fmt.Sprintf("%04x", len(bind+addr)/2)
	extra := "0002" + contract + first + bind + addr
	amount := convertToMVMHex(amt)
	second := fmt.Sprintf("%04x", len(pass+addr+amount)/2)
	extra = extra + contract + second + pass + addr + amount
	b, _ := hex.DecodeString(extra)
	if len(b) != 148 {
		panic(extra)
	}
	k := new(big.Int).SetBytes(crypto.Keccak256(b))
	o, err := p.Read(nil, k)
	if err != nil {
		panic(err)
	}
	if bytes.Compare(o, b) != 0 {
		return p.buildHash(k, b)
	}
	extra = "0001" + contract + first + bind + addr
	b, _ = hex.DecodeString(extra)
	if len(b) != 92 {
		panic(extra)
	}
	return b
}

func (p *Proxy) buildHash(k *big.Int, b []byte) []byte {
	_, err := p.Write(p.signer, k, b)
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

	if len(extra) != 92 {
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

package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"

	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

func (p *Proxy) bindAndPass(snapshotId, addr, asset string, amount decimal.Decimal) error {
	ctx := context.Background()

	trace := mixin.UniqueConversationID(snapshotId, "BIND||PASS")
	extra, _ := hex.DecodeString("bind-addr-pass-asset-amount")
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

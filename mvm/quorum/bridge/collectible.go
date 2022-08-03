package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

const (
	nfoAssetId = "1700941284a95f31b25ec8c546008f208f88eee4419ccdcdbe6e3195e60128ca" // FIXME
)

func (u *User) bindAndPassCollectible(ctx context.Context, p *Proxy, out *mixin.CollectibleOutput, addr string) error {
	traceId := mixin.UniqueConversationID(out.OutputID, "BIND||PASS")
	extra := u.buildCollectibleExtra(p, addr, out.TokenID)
	op := &encoding.Operation{
		Purpose: encoding.OperationPurposeGroupEvent,
		Process: MVMRegistryId,
		Extra:   extra,
	}
	extra = []byte(base64.RawURLEncoding.EncodeToString(op.Encode()))

	raw, err := p.buildRawCollectibleTransaction(ctx, out, extra, traceId)
	if err != nil {
		return err
	}
	req, err := p.CreateCollectibleRequest(ctx, "SIGN", hex.EncodeToString(raw.PayloadMarshal()))
	if err != nil {
		return err
	}
	req, err = p.SignCollectibleRequest(ctx, req.RequestID, u.PIN)
	if err != nil {
		return err
	}
	hash, err := p.SendRawTransaction(ctx, req.RawTransaction)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) buildCollectibleExtra(p *Proxy, addr, token string) []byte {
	bind, pass := "81bac14f", "82c4b3b2"
	contract := strings.ToLower(MVMBridgeContract[2:])
	addr = "000000000000000000000000" + strings.ToLower(addr[2:])
	first := fmt.Sprintf("%04x", len(bind+addr)/2)
	second := fmt.Sprintf("%04x", len(pass+addr)/2)
	if u.checkCollectibleBind(p) {
		extra := "0001" + contract + second + pass
		extra = extra + convertToMVMAddress(token)
		b, _ := hex.DecodeString(extra)
		if len(b) != 60 {
			panic(extra)
		}
		return b
	}

	extra := "0002" + contract + first + bind + addr
	extra = extra + contract + second + pass
	extra = extra + convertToMVMAddress(token)
	b, _ := hex.DecodeString(extra)
	if len(b) != 118 {
		panic(extra)
	}
	return p.buildHash(b)
}

func (u *User) checkCollectibleBind(p *Proxy) bool {
	ua, err := u.getContract(p)
	if err != nil {
		panic(err)
	}
	if ua.String() == "0x0000000000000000000000000000000000000000" {
		return false
	}
	ba, err := p.mirror.Bridges(nil, ua)
	if err != nil {
		panic(err)
	}
	return ba.String() != "0x0000000000000000000000000000000000000000"
}

func (p *Proxy) buildRawCollectibleTransaction(ctx context.Context, utxo *mixin.CollectibleOutput, extra []byte, traceId string) (*common.VersionedTransaction, error) {
	assetId, err := crypto.HashFromString(nfoAssetId)
	if err != nil {
		panic(err)
	}
	ver := common.NewTransaction(assetId)
	ver.Extra = mtg.BuildExtraNFO(extra)

	if utxo.Amount.Cmp(decimal.NewFromInt(1)) != 0 {
		panic(utxo.OutputID)
	}
	ver.AddInput(crypto.Hash(utxo.TransactionHash), utxo.OutputIndex)

	keys, err := p.BatchReadGhostKeys(ctx, []*mixin.GhostInput{{
		Receivers: MVMMembers,
		Index:     0,
		Hint:      traceId,
	}})
	if err != nil {
		return nil, err
	}

	out := keys[0].DumpOutput(uint8(MVMThreshold), utxo.Amount)
	ver.Outputs = append(ver.Outputs, newCommonOutput(out))
	return ver.AsLatestVersion(), nil
}

func newCommonOutput(out *mixin.Output) *common.Output {
	cout := &common.Output{
		Type:   common.OutputTypeScript,
		Amount: common.NewIntegerFromString(out.Amount.String()),
		Script: common.Script(out.Script),
		Mask:   crypto.Key(out.Mask),
	}
	for _, k := range out.Keys {
		ck := crypto.Key(k)
		cout.Keys = append(cout.Keys, &ck)
	}
	return cout
}

func (p *Proxy) readNetworkCollectibleOutputs(ctx context.Context, members []string, threshold uint8, offset time.Time, limit int) ([]*mixin.CollectibleOutput, error) {
	params := make(map[string]string)
	if !offset.IsZero() {
		params["offset"] = offset.UTC().Format(time.RFC3339Nano)
	}
	if limit > 0 {
		params["limit"] = fmt.Sprint(limit)
	}
	if threshold < 1 || int(threshold) > len(members) {
		return nil, fmt.Errorf("invalid members %v %d", members, threshold)
	}
	params["members"] = mixin.HashMembers(members)
	params["threshold"] = fmt.Sprint(threshold)
	params["state"] = "unspent"

	var outputs []*mixin.CollectibleOutput
	err := p.Get(ctx, "/collectibles/outputs", params, &outputs)
	if err != nil {
		return nil, err
	}
	return outputs, nil
}

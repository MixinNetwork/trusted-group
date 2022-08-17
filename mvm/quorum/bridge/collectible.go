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
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/shopspring/decimal"
)

const (
	nfoAssetId = "1700941284a95f31b25ec8c546008f208f88eee4419ccdcdbe6e3195e60128ca" // FIXME
)

type CollectibleOutput struct {
	mixin.CollectibleOutput
	Memo string `json:"memo"`
}

func (p *Proxy) loopCollectibleOutputs(ctx context.Context, store *Storage) error {
	ckpt, err := store.readCollectiblesCheckpoint(ctx)
	if err != nil {
		return err
	}
	outputs, err := p.readNetworkCollectibleOutputs(ctx, ckpt, 500)
	logger.Verbosef("Proxy.loopCollectibleOutputs(%s) => %d %v", ckpt, len(outputs), err)
	if err != nil {
		return err
	}

	for _, o := range outputs {
		ckpt = o.CreatedAt
		if o.UserID == "" {
			continue
		}
		if len(o.Receivers) != 1 || o.ReceiversThreshold != 1 {
			continue
		}
		if o.Receivers[0] != o.UserID {
			continue
		}
		if o.State != "unspent" {
			continue
		}
		logger.Verbosef("Proxy.loopCollectibleOutputs(%s) => %d %v => %v", ckpt, len(outputs), err, *o)
		err = store.writeCollectibleOutput(o)
		if err != nil {
			return err
		}
	}

	err = store.writeCollectiblesCheckpoint(ctx, ckpt)
	if err != nil {
		return err
	}
	if len(outputs) < 500 {
		time.Sleep(time.Second * 2)
	}
	return nil
}

func (p *Proxy) processCollectibleOutputs(ctx context.Context, store *Storage) {
	outputs, err := store.listCollectibleOutputs(100)
	if err != nil {
		panic(err)
	}

	for _, o := range outputs {
		if o.State != "unspent" {
			continue
		}
		user, err := store.readUserById(o.UserID)
		if err != nil {
			panic(err)
		}
		if user == nil {
			continue
		}
		err = p.processCollectibleOutputForUser(ctx, store, o, user)
		if err != nil {
			panic(err)
		}
	}

	err = store.deleteCollectibleOutputs(outputs)
	if err != nil {
		panic(err)
	}
	if len(outputs) < 100 {
		time.Sleep(1 * time.Second)
	}
}

func (p *Proxy) processCollectibleRawTransactions(ctx context.Context, store *Storage) {
	raws, err := store.listCollectibleRawTransactions(100)
	if err != nil {
		panic(err)
	}

	for key, raw := range raws {
		hash, err := p.SendRawTransaction(ctx, raw)
		if err != nil {
			panic(err)
		}
		if hash.String() != key {
			panic(raw)
		}
		tx, err := p.GetRawTransaction(ctx, *hash)
		if err != nil {
			panic(err)
		}
		if tx == nil || tx.Snapshot == nil {
			continue
		}
		err = store.deleteCollectibleRawTransaction(raw)
		if err != nil {
			panic(err)
		}
	}

	if len(raws) < 100 {
		time.Sleep(1 * time.Second)
	}
}

func (p *Proxy) processCollectibleOutputForUser(ctx context.Context, store *Storage, o *CollectibleOutput, user *User) error {
	act, err := p.decodeAction(user, o.Memo, "", true)
	if err != nil {
		return err
	}
	if act != nil && user.handleCollectible(ctx, store, o, act) == nil {
		return nil
	}
	return user.bindAndPassCollectible(ctx, p, o)
}

func (u *User) handleCollectible(ctx context.Context, store *Storage, o *CollectibleOutput, act *Action) error {
	logger.Verbosef("User.handleCollectible(%v, %v)", *o, *act)
	if act.Destination != "" {
		panic(act.Destination)
	}

	traceId := mixin.UniqueConversationID(o.OutputID, "HANDLE||TRANSFER")
	extra := []byte(base64.RawURLEncoding.EncodeToString([]byte(act.Extra)))

	return u.sendRawCollectibleTransaction(ctx, o, act.Receivers, uint8(act.Threshold), extra, traceId)
}

func (u *User) bindAndPassCollectible(ctx context.Context, p *Proxy, out *CollectibleOutput) error {
	logger.Verbosef("User.bindAndPassCollectible(%v)", *out)
	traceId := mixin.UniqueConversationID(out.OutputID, "BIND||PASS")
	extra := u.buildCollectibleExtra(p, u.FullName, out.TokenID)
	op := &encoding.Operation{
		Purpose: encoding.OperationPurposeGroupEvent,
		Process: MVMRegistryId,
		Extra:   extra,
	}
	extra = []byte(base64.RawURLEncoding.EncodeToString(op.Encode()))

	return u.sendRawCollectibleTransaction(ctx, out, MVMMembers, uint8(MVMThreshold), extra, traceId)
}

func (u *User) buildCollectibleExtra(p *Proxy, addr, token string) []byte {
	bind, pass := "81bac14f", "82c4b3b2"
	contract := strings.ToLower(MVMMirrorContract[2:])
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

func (u *User) sendRawCollectibleTransaction(ctx context.Context, utxo *CollectibleOutput, receivers []string, threshold uint8, extra []byte, traceId string) error {
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

	uc, err := mixin.NewFromKeystore(u.Key)
	if err != nil {
		panic(err)
	}
	keys, err := uc.BatchReadGhostKeys(ctx, []*mixin.GhostInput{{
		Receivers: receivers,
		Index:     0,
		Hint:      traceId,
	}})
	if err != nil {
		return err
	}

	out := keys[0].DumpOutput(threshold, utxo.Amount)
	ver.Outputs = append(ver.Outputs, newCommonOutput(out))
	req, err := uc.CreateCollectibleRequest(ctx, "SIGN", hex.EncodeToString(ver.AsLatestVersion().PayloadMarshal()))
	if err != nil {
		return err
	}
	req, err = uc.SignCollectibleRequest(ctx, req.RequestID, u.PIN)
	if err != nil {
		return err
	}
	err = store.writeCollectibleRawTransaction(req.RawTransaction)
	if err != nil {
		panic(err)
	}
	return nil
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

func (p *Proxy) readNetworkCollectibleOutputs(ctx context.Context, offset time.Time, limit int) ([]*CollectibleOutput, error) {
	params := make(map[string]string)
	if !offset.IsZero() {
		params["offset"] = offset.UTC().Format(time.RFC3339Nano)
	}
	if limit > 0 {
		params["limit"] = fmt.Sprint(limit)
	}

	var outputs []*CollectibleOutput
	err := p.Get(ctx, "/network/collectibles/outputs", params, &outputs)
	if err != nil {
		return nil, err
	}
	return outputs, nil
}

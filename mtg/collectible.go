package mtg

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

const (
	CollectibleMetaTokenId  = "2f8aa18a-3cb8-31d5-95bc-5a4f2e25dc2f"
	CollectibleMixinAssetId = "1700941284a95f31b25ec8c546008f208f88eee4419ccdcdbe6e3195e60128ca"
)

type CollectibleOutput struct {
	Type               string
	UserId             string
	OutputId           string
	TokenId            string
	TransactionHash    crypto.Hash
	OutputIndex        int
	Amount             decimal.Decimal
	SendersThreshold   int64
	Senders            []string
	ReceiversThreshold int64
	Receivers          []string
	Memo               string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	SignedBy           string
	SignedTx           string
	State              int
}

type CollectibleTransaction struct {
	TraceId   string
	State     int
	Receivers []string
	Threshold int
	Amount    string
	NFO       []byte
	Raw       []byte
	Hash      crypto.Hash
	UpdatedAt time.Time
	TokenId   string
}

func (grp *Group) BuildCollectibleMintTransaction(ctx context.Context, receivers []string, threshold int, nfo []byte) error {
	traceId := nfoTraceId(nfo)
	return grp.buildCollectibleTransaction(ctx, receivers, threshold, nfo, "", traceId)
}

func (grp *Group) BuildCollectibleTransferTransaction(ctx context.Context, receivers []string, threshold int, memo string, tokenId, traceId string) error {
	if uuid.FromStringOrNil(tokenId).String() != tokenId {
		return fmt.Errorf("invalid collectible token id %s", tokenId)
	}
	extra := EncodeMixinExtra("", traceId, memo)
	nfo := BuildExtraNFO([]byte(extra))
	if len(nfo) > common.ExtraSizeGeneralLimit {
		panic(memo)
	}
	return grp.buildCollectibleTransaction(ctx, receivers, threshold, nfo, tokenId, traceId)
}

func (grp *Group) buildCollectibleTransaction(ctx context.Context, receivers []string, threshold int, nfo []byte, tokenId, traceId string) error {
	if threshold <= 0 || threshold > len(receivers) {
		return fmt.Errorf("invalid receivers threshold %d/%d", threshold, len(receivers))
	}

	nfm, err := DecodeNFOMemo(nfo)
	if err != nil {
		return fmt.Errorf("invalid nfo data %x %v", nfo, err)
	}
	if nfm.WillMint() && tokenId != "" {
		return fmt.Errorf("invalid nfo and token combination %x %s", nfo, tokenId)
	}
	if !nfm.WillMint() && tokenId == "" {
		return fmt.Errorf("invalid nfo and token combination %x %s", nfo, tokenId)
	}

	if uuid.FromStringOrNil(traceId).String() != traceId {
		return fmt.Errorf("invalid collectible trace id %s", traceId)
	}

	old, err := grp.store.ReadCollectibleTransaction(traceId)
	if err != nil || old != nil {
		return err
	}
	tx := &CollectibleTransaction{
		TraceId:   traceId,
		State:     TransactionStateInitial,
		Receivers: receivers,
		Threshold: threshold,
		Amount:    "1",
		NFO:       nfo,
		UpdatedAt: grp.clock.Now(),
		TokenId:   tokenId,
	}
	return grp.store.WriteCollectibleTransaction(tx.TraceId, tx)
}

func (out *CollectibleOutput) StateName() string {
	switch out.State {
	case OutputStateUnspent:
		return mixin.UTXOStateUnspent
	case OutputStateSigned:
		return mixin.UTXOStateSigned
	case OutputStateSpent:
		return mixin.UTXOStateSpent
	}
	panic(out.State)
}

func (o *CollectibleOutput) Unified() *UnifiedOutput {
	return &UnifiedOutput{
		Type:                      OutputTypeCollectible,
		UserId:                    o.UserId,
		TransactionHash:           o.TransactionHash,
		OutputIndex:               o.OutputIndex,
		Amount:                    o.Amount,
		Memo:                      o.Memo,
		CreatedAt:                 o.CreatedAt,
		UpdatedAt:                 o.UpdatedAt,
		SignedBy:                  o.SignedBy,
		SignedTx:                  o.SignedTx,
		State:                     o.StateName(),
		UnifiedOutputId:           o.OutputId,
		UnifiedTokenId:            o.TokenId,
		UnifiedSenders:            o.Senders,
		UnifiedSendersThreshold:   o.SendersThreshold,
		UnifiedReceivers:          o.Receivers,
		UnifiedReceiversThreshold: o.ReceiversThreshold,
	}
}

func (grp *Group) signCollectibleTransaction(ctx context.Context, tx *CollectibleTransaction) ([]byte, error) {
	outputs, err := grp.store.ListCollectibleOutputsForTransaction(tx.TraceId)
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		if tx.TokenId == "" {
			outputs, err = grp.store.ListCollectibleOutputsForToken(mixin.UTXOStateUnspent, CollectibleMetaTokenId, 1)
		} else {
			outputs, err = grp.store.ListCollectibleOutputsForToken(mixin.UTXOStateUnspent, tx.TokenId, 1)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(outputs) == 0 {
		return nil, fmt.Errorf("empty outputs %s", tx.Amount)
	}

	ver, err := grp.buildRawCollectibleTransaction(ctx, tx, outputs)
	if err != nil {
		return nil, err
	}
	if ver.AggregatedSignature != nil || len(ver.SignaturesMap) > 0 {
		return ver.Marshal(), nil
	}

	raw := hex.EncodeToString(ver.Marshal())
	req, err := grp.mixin.CreateCollectibleRequest(ctx, mixin.MultisigActionSign, raw)
	if err != nil {
		return nil, err
	}

	req, err = grp.mixin.SignCollectibleRequest(ctx, req.RequestID, grp.pin)
	if err != nil {
		return nil, err
	}

	for _, out := range outputs {
		out.State = OutputStateSigned
		out.SignedBy = ver.PayloadHash().String()
		out.SignedTx = req.RawTransaction
	}
	err = grp.store.WriteCollectibleOutputs(outputs, tx.TraceId)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(req.RawTransaction)
}

func (grp *Group) buildRawCollectibleTransaction(ctx context.Context, tx *CollectibleTransaction, outputs []*CollectibleOutput) (*common.VersionedTransaction, error) {
	old, _ := decodeCollectibleTransactionWithExtra(outputs[0].SignedTx)
	if old != nil {
		return old, nil
	}

	if tx.Amount != "1" {
		panic(tx.Amount)
	}
	assetId, err := crypto.HashFromString(CollectibleMixinAssetId)
	if err != nil {
		panic(err)
	}
	ver := common.NewTransactionV2(assetId)
	ver.Extra = tx.NFO

	var total common.Integer
	for _, out := range outputs {
		total = total.Add(common.NewIntegerFromString(out.Amount.String()))
		ver.AddInput(out.TransactionHash, out.OutputIndex)
	}
	if total.Cmp(common.NewIntegerFromString(tx.Amount)) < 0 {
		return nil, fmt.Errorf("insufficient %s %s", total, tx.Amount)
	}

	keys, err := grp.mixin.BatchReadGhostKeys(ctx, []*mixin.GhostInput{{
		Receivers: tx.Receivers,
		Index:     0,
		Hint:      tx.TraceId,
	}, {
		Receivers: grp.members,
		Index:     1,
		Hint:      tx.TraceId,
	}})
	if err != nil {
		return nil, err
	}

	amount, err := decimal.NewFromString(tx.Amount)
	if err != nil {
		return nil, err
	}
	out := keys[0].DumpOutput(uint8(tx.Threshold), amount)
	ver.Outputs = append(ver.Outputs, newCommonOutput(out))

	if diff := total.Sub(common.NewIntegerFromString(tx.Amount)); diff.Sign() > 0 {
		amount, err := decimal.NewFromString(diff.String())
		if err != nil {
			return nil, err
		}
		out := keys[1].DumpOutput(uint8(grp.threshold), amount)
		ver.Outputs = append(ver.Outputs, newCommonOutput(out))
	}

	return ver.AsVersioned(), nil
}

func decodeCollectibleTransactionWithExtra(s string) (*common.VersionedTransaction, *mixinExtraPack) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil, nil
	}
	tx, err := common.UnmarshalVersionedTransaction(raw)
	if err != nil {
		return nil, nil
	}
	nfm, err := DecodeNFOMemo(tx.Extra)
	if err != nil {
		panic(tx.PayloadHash().String())
	}
	p := DecodeMixinExtra(string(nfm.Extra))
	if p == nil {
		return nil, nil
	}
	return tx, p
}

func nfoTraceId(nfo []byte) string {
	nid := crypto.NewHash(nfo).String()
	return mixin.UniqueConversationID(nid, nid)
}

type cr struct {
	RequestID      string `json:"request_id"`
	RawTransaction string `json:"raw_transaction"`
}

package chain

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/MixinNetwork/trusted-group/mvm/eos/secp256k1"
)

type TransactionExtension struct {
	Type uint16
	Data []byte
}

func (a *TransactionExtension) Size() int {
	return 2 + 5 + len(a.Data)
}

func (t *TransactionExtension) Pack() []byte {
	enc := NewEncoder(2 + 5 + len(t.Data))
	enc.Pack(t.Type)
	enc.Pack(t.Data)
	return enc.GetBytes()
}

func (t *TransactionExtension) Unpack(data []byte) (int, error) {
	var err error
	dec := NewDecoder(data)
	t.Type, err = dec.UnpackUint16()
	if err != nil {
		return 0, err
	}

	t.Data, err = dec.UnpackBytes()
	if err != nil {
		return 0, err
	}
	return dec.Pos(), nil
}

type Transaction struct {
	// time_point_sec  expiration;
	// uint16_t        ref_block_num;
	// uint32_t        ref_block_prefix;
	// unsigned_int    max_net_usage_words = 0UL; /// number of 8 byte words this transaction can serialize into after compressions
	// uint8_t         max_cpu_usage_ms = 0UL; /// number of CPU usage units to bill transaction for
	// unsigned_int    delay_sec = 0UL; /// number of seconds to delay transaction, default: 0
	Expiration     TimePointSec `json:"expiration"`
	RefBlockNum    uint16       `json:"ref_block_num"`
	RefBlockPrefix uint32       `json:"ref_block_prefix"`
	//[VLQ or Base-128 encoding](https://en.wikipedia.org/wiki/Variable-length_quantity)
	//unsigned_int vaint (eosio.cdt/libraries/eosiolib/core/eosio/varint.hpp)
	MaxNetUsageWords   VarUint32               `json:"max_net_usage_words"`
	MaxCpuUsageMs      uint8                   `json:"max_cpu_usage_ms"`
	DelaySec           VarUint32               `json:"delay_sec"`
	ContextFreeActions []*Action               `json:"context_free_actions"`
	Actions            []*Action               `json:"actions"`
	Extention          []*TransactionExtension `json:"transaction_extensions"`
}

type PackedTransaction struct {
	Signatures    []string `json:"signatures"`
	Compression   string   `json:"compression"`
	PackedContext Bytes    `json:"packed_context_free_data"`
	PackedTx      Bytes    `json:"packed_trx"`
}

func NewTransaction(expiration uint32) *Transaction {
	t := &Transaction{}
	t.Expiration = TimePointSec{uint32(expiration)}
	// t.RefBlockNum = uint16(taposBlockNum)
	// t.RefBlockPrefix = uint32(taposBlockPrefix)
	t.MaxNetUsageWords = VarUint32(0)
	t.MaxCpuUsageMs = uint8(0)
	// t.DelaySec = uint32(delaySec)
	t.ContextFreeActions = []*Action{}
	t.Actions = []*Action{}
	t.Extention = []*TransactionExtension{}

	return t
}

func GetRefBlockNum(refBlock []byte) uint32 {
	return uint32(refBlock[0])<<24 | uint32(refBlock[1])<<16 | uint32(refBlock[2])<<8 | uint32(refBlock[3])
}

func GetRefBlockPrefix(refBlock []byte) uint32 {
	return uint32(refBlock[11])<<24 | uint32(refBlock[10])<<16 | uint32(refBlock[9])<<8 | uint32(refBlock[8])
}

func (t *Transaction) SetReferenceBlock(refBlock string) error {
	id, err := hex.DecodeString(refBlock)
	if err != nil {
		return newError(err)
	}
	t.RefBlockNum = uint16(GetRefBlockNum(id))
	t.RefBlockPrefix = GetRefBlockPrefix(id)
	return nil
}

func (t *Transaction) AddAction(a *Action) {
	t.Actions = append(t.Actions, a)
}

func (t *Transaction) Pack() []byte {
	initSize := 4 + 2 + 4 + 5 + 1 + 5

	initSize += 5 // Max varint size
	for _, action := range t.ContextFreeActions {
		initSize += action.Size()
	}

	initSize += 5 // Max varint size
	for _, action := range t.Actions {
		initSize += action.Size()
	}

	initSize += 5 // Max varint size
	for _, extention := range t.Extention {
		initSize += extention.Size()
	}
	enc := NewEncoder(initSize)
	enc.Pack(t.Expiration.UTCSeconds)
	enc.Pack(t.RefBlockNum)
	enc.Pack(t.RefBlockPrefix)
	enc.PackVarUint32(uint32(t.MaxNetUsageWords))
	enc.PackUint8(t.MaxCpuUsageMs)
	enc.PackVarUint32(uint32(t.DelaySec))

	enc.PackLength(len(t.ContextFreeActions))
	for _, action := range t.ContextFreeActions {
		enc.Pack(action)
	}

	enc.PackLength(len(t.Actions))
	for _, action := range t.Actions {
		enc.Pack(action)
	}

	enc.PackLength(len(t.Extention))
	for _, extention := range t.Extention {
		enc.Pack(extention)
	}
	return enc.GetBytes()
}

func (t *Transaction) Unpack(data []byte) (int, error) {
	var err error

	dec := NewDecoder(data)
	t.Expiration.UTCSeconds, err = dec.UnpackUint32()
	if err != nil {
		return 0, err
	}

	t.RefBlockNum, err = dec.UnpackUint16()
	if err != nil {
		return 0, err
	}

	t.RefBlockPrefix, err = dec.UnpackUint32()
	if err != nil {
		return 0, err
	}

	t.MaxNetUsageWords, err = dec.UnpackVarUint32()
	if err != nil {
		return 0, err
	}

	t.MaxCpuUsageMs, err = dec.UnpackUint8()
	if err != nil {
		return 0, err
	}

	t.DelaySec, err = dec.UnpackVarUint32()
	if err != nil {
		return 0, err
	}

	contextFreeActionLength, err := dec.UnpackVarUint32()
	if err != nil {
		return 0, err
	}

	t.ContextFreeActions = make([]*Action, contextFreeActionLength)
	for i := 0; i < int(contextFreeActionLength); i++ {
		action := &Action{}
		_, err := dec.Unpack(action)
		if err != nil {
			return 0, err
		}
		t.ContextFreeActions[i] = action
	}

	actionLength, err := dec.UnpackVarUint32()
	if err != nil {
		return 0, err
	}

	t.Actions = make([]*Action, actionLength)
	for i := 0; i < int(actionLength); i++ {
		action := &Action{}
		_, err := dec.Unpack(action)
		if err != nil {
			return 0, err
		}
		t.Actions[i] = action
	}

	extentionLength, err := dec.UnpackVarUint32()
	if err != nil {
		return 0, err
	}
	t.Extention = make([]*TransactionExtension, extentionLength)
	for i := 0; i < int(extentionLength); i++ {
		extention := &TransactionExtension{}
		extention.Type, err = dec.UnpackUint16()
		if err != nil {
			return 0, err
		}

		extention.Data, err = dec.UnpackBytes()
		if err != nil {
			return 0, err
		}
		t.Extention[i] = extention
	}
	return dec.Pos(), nil
}

func (t *Transaction) Id(chainId *Bytes32) *Bytes32 {
	hash := sha256.New()
	hash.Write(chainId[:])
	hash.Write(t.Pack())
	//TODO: hash context_free_data
	cfdHash := [32]byte{}
	hash.Write(cfdHash[:])
	digest := hash.Sum(nil)
	return NewBytes32(digest)
}

func (t *Transaction) Sign(privKey *secp256k1.PrivateKey, chainId *Bytes32) (*secp256k1.Signature, error) {
	digest := t.Id(chainId)
	sign, err := secp256k1.Sign(digest[:], privKey)
	if err != nil {
		return nil, err
	}
	return sign, nil
}

func (t *Transaction) SignWithPublicKey(pubKey string, chainId *Bytes32) (*secp256k1.Signature, error) {
	priv, err := GetWallet().GetPrivateKey(pubKey)
	if err != nil {
		return nil, err
	}
	return t.Sign(priv, chainId)
}

func NewPackedTransaction(tx *Transaction, signatures []string, compress bool) *PackedTransaction {
	packed := &PackedTransaction{}
	packed.Compression = "none"
	packed.PackedTx = tx.Pack()
	if compress {
		packed.Compression = "zlib"
		//TODO: compress PackedTx with zlib
		var b bytes.Buffer
		w := zlib.NewWriter(&b)
		w.Write(packed.PackedTx[:])
		w.Close()
		packed.PackedTx = b.Bytes()
	} else {
		packed.Compression = "none"
	}

	if signatures == nil {
		packed.Signatures = []string{}
	} else {
		packed.AddSignatures(signatures)
	}
	return packed
}

func (t *PackedTransaction) AddSignatures(signatures []string) {
	sort.Slice(signatures, func(i, j int) bool {
		return signatures[i] < signatures[j]
	})
	t.Signatures = append(t.Signatures, signatures...)
}

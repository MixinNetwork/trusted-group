package mtg

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid"
)

func writeUuid(enc *common.Encoder, id string) {
	uid := uuid.FromStringOrNil(id)
	enc.WriteInt(16)
	enc.Write(uid.Bytes())
}

func writeString(enc *common.Encoder, str string) {
	data := []byte(str)
	enc.WriteInt(len(data))
	enc.Write(data)
}

func writeBool(enc *common.Encoder, f bool) {
	data := 0
	if f {
		data = 1
	}
	enc.WriteInt(data)
}

func writeReferences(enc *common.Encoder, refs []crypto.Hash) error {
	rl := len(refs)
	enc.WriteInt(rl)
	for _, r := range refs {
		if !r.HasValue() {
			panic(fmt.Errorf("invalid ref %s", r.String()))
		}
		enc.Write(r[:])
	}
	return nil
}

func (tx *Transaction) serialize() (string, error) {
	enc := common.NewEncoder()
	writeUuid(enc, tx.TraceId)
	writeUuid(enc, tx.AppId)
	writeUuid(enc, tx.OpponentAppId)
	enc.WriteInt(tx.State)
	writeUuid(enc, tx.AssetId)
	writeString(enc, strings.Join(tx.Receivers, ","))
	enc.WriteInt(tx.Threshold)
	writeString(enc, tx.Amount)
	writeString(enc, tx.Memo)
	enc.WriteUint64(tx.Sequence)
	writeBool(enc, tx.compaction)
	writeBool(enc, tx.storage)
	writeReferences(enc, tx.references)
	writeUuid(enc, tx.storageTraceId)
	return base64.RawURLEncoding.EncodeToString(enc.Bytes()), nil
}

func readUuid(dec *common.Decoder) (string, error) {
	data, err := dec.ReadBytes()
	if err != nil {
		return "", err
	}
	id, err := uuid.FromBytes(data)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func readString(dec *common.Decoder) (string, error) {
	data, err := dec.ReadBytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func readBool(dec *common.Decoder) (bool, error) {
	f, err := dec.ReadInt()
	if err != nil {
		return false, err
	}
	return f == 1, nil
}

func readReferences(dec *common.Decoder) ([]crypto.Hash, error) {
	rl, err := dec.ReadInt()
	if err != nil {
		return nil, err
	}
	var refs []crypto.Hash
	for ; rl > 0; rl -= 1 {
		var r crypto.Hash
		err := dec.Read(r[:])
		if err != nil {
			return nil, err
		}
		refs = append(refs, r)
	}
	return refs, nil
}

func deserialize(data string) (*Transaction, error) {
	rb, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	dec := common.NewDecoder(rb)

	traceId, err := readUuid(dec)
	if err != nil {
		return nil, err
	}
	appId, err := readUuid(dec)
	if err != nil {
		return nil, err
	}
	opponentAppId, err := readUuid(dec)
	if err != nil {
		return nil, err
	}
	state, err := dec.ReadInt()
	if err != nil {
		return nil, err
	}
	assetId, err := readUuid(dec)
	if err != nil {
		return nil, err
	}
	receivers, err := readString(dec)
	if err != nil {
		return nil, err
	}
	treshold, err := dec.ReadInt()
	if err != nil {
		return nil, err
	}
	amount, err := readString(dec)
	if err != nil {
		return nil, err
	}
	memo, err := readString(dec)
	if err != nil {
		return nil, err
	}
	sequence, err := dec.ReadUint64()
	if err != nil {
		return nil, err
	}
	compaction, err := readBool(dec)
	if err != nil {
		return nil, err
	}
	storage, err := readBool(dec)
	if err != nil {
		return nil, err
	}
	refs, err := readReferences(dec)
	if err != nil {
		return nil, err
	}
	storageAssetId, err := readUuid(dec)
	if err != nil {
		return nil, err
	}

	tx := &Transaction{
		TraceId:       traceId,
		AppId:         appId,
		OpponentAppId: opponentAppId,
		State:         state,
		AssetId:       assetId,
		Receivers:     strings.Split(receivers, ","),
		Threshold:     treshold,
		Amount:        amount,
		Memo:          memo,
		Sequence:      sequence,
		compaction:    compaction,
		storage:       storage,
		references:    refs,
	}
	if storageAssetId != uuid.Nil.String() {
		tx.storageTraceId = storageAssetId
	}

	return tx, nil
}

func SerializeTransactions(txs []*Transaction) (string, error) {
	var data []string
	for _, tx := range txs {
		etx, err := tx.serialize()
		if err != nil {
			return "", err
		}
		data = append(data, etx)
	}
	return strings.Join(data, ","), nil
}

func DeserializeTransactions(txs string) ([]*Transaction, error) {
	items := strings.Split(txs, ",")
	var transactions []*Transaction
	for _, item := range items {
		tx, err := deserialize(item)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

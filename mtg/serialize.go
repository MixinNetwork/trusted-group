package mtg

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
)

var (
	null  = []byte{0x00, 0x00}
	magic = []byte{0x77, 0x77}
)

func writeByte(enc *common.Encoder, b int) {
	if b > 200 {
		panic(b)
	}
	err := enc.WriteByte(byte(b))
	if err != nil {
		panic(err)
	}
}

func writeUuid(enc *common.Encoder, id string) {
	uid := uuid.FromStringOrNil(id)
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
	writeByte(enc, data)
}

func writeReferences(enc *common.Encoder, refs []crypto.Hash) {
	writeByte(enc, len(refs))
	for _, r := range refs {
		if !r.HasValue() {
			panic(fmt.Errorf("invalid ref %s", r.String()))
		}
		enc.Write(r[:])
	}
}

func writeConsumed(enc *common.Encoder, consumed []*UnifiedOutput, consumedIds []string) {
	if len(consumed) > 0 && len(consumed) != len(consumedIds) {
		panic(len(consumedIds))
	}
	writeByte(enc, len(consumedIds))
	for _, id := range consumedIds {
		writeUuid(enc, id)
	}
}

func (tx *Transaction) Serialize() []byte {
	enc := common.NewEncoder()
	writeUuid(enc, tx.TraceId)
	writeUuid(enc, tx.AppId)
	writeUuid(enc, tx.OpponentAppId)
	writeUuid(enc, tx.ActionId)
	writeByte(enc, tx.State)
	writeUuid(enc, tx.AssetId)
	writeString(enc, tx.Amount)
	writeString(enc, tx.Memo)
	enc.WriteUint64(tx.Sequence)
	writeBool(enc, tx.compaction)
	writeBool(enc, tx.storage)
	writeReferences(enc, tx.references)
	writeUuid(enc, tx.storageTraceId)
	writeConsumed(enc, tx.consumed, tx.consumedIds)
	if tx.IsWithdrawal() {
		enc.Write(magic)
		writeString(enc, tx.Destination.String)
		writeString(enc, tx.Tag.String)
	} else {
		enc.Write(null)
		writeString(enc, strings.Join(tx.Receivers, ","))
		writeByte(enc, tx.Threshold)
	}
	return enc.Bytes()
}

func readUuid(dec *common.Decoder) (string, error) {
	b := make([]byte, 16)
	err := dec.Read(b)
	if err != nil {
		return "", err
	}
	id, err := uuid.FromBytes(b)
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
	f, err := dec.ReadByte()
	if err != nil {
		return false, err
	}
	return f == 1, nil
}

func readReferences(dec *common.Decoder) ([]crypto.Hash, error) {
	rl, err := dec.ReadByte()
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

func readConsumed(dec *common.Decoder) ([]string, error) {
	cl, err := dec.ReadByte()
	if err != nil {
		return nil, err
	}
	var outputs []string
	for ; cl > 0; cl -= 1 {
		oid, err := readUuid(dec)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, oid)
	}
	return outputs, nil
}

func Deserialize(rb []byte) (*Transaction, error) {
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
	actionId, err := readUuid(dec)
	if err != nil {
		return nil, err
	}
	state, err := dec.ReadByte()
	if err != nil {
		return nil, err
	}
	assetId, err := readUuid(dec)
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
	storageTraceId, err := readUuid(dec)
	if err != nil {
		return nil, err
	}
	ids, err := readConsumed(dec)
	if err != nil {
		return nil, err
	}
	tx := &Transaction{
		TraceId:       traceId,
		AppId:         appId,
		OpponentAppId: opponentAppId,
		ActionId:      actionId,
		State:         int(state),
		AssetId:       assetId,
		Amount:        amount,
		Memo:          memo,
		Sequence:      sequence,
		compaction:    compaction,
		storage:       storage,
		references:    refs,
		consumedIds:   ids,
	}
	if storageTraceId != uuid.Nil.String() {
		tx.storageTraceId = storageTraceId
	}

	magic, err := dec.ReadMagic()
	if err != nil {
		return nil, err
	}
	if magic {
		destination, err := readString(dec)
		if err != nil {
			return nil, err
		}
		tag, err := readString(dec)
		if err != nil {
			return nil, err
		}
		tx.Destination = sql.NullString{Valid: true, String: destination}
		tx.Tag = sql.NullString{Valid: true, String: tag}
	} else {
		receivers, err := readString(dec)
		if err != nil {
			return nil, err
		}
		threshold, err := dec.ReadByte()
		if err != nil {
			return nil, err
		}
		tx.Receivers = SplitIds(receivers)
		tx.Threshold = int(threshold)
	}
	return tx, nil
}

func SerializeTransactions(txs []*Transaction) []byte {
	enc := common.NewEncoder()
	writeByte(enc, len(txs))
	for _, tx := range txs {
		b := tx.Serialize()
		enc.WriteInt(len(b))
		enc.Write(b)
	}
	return enc.Bytes()
}

func DeserializeTransactions(tb []byte) ([]*Transaction, error) {
	dec := common.NewDecoder(tb)
	count, err := dec.ReadByte()
	if err != nil || count == 0 {
		return nil, err
	}
	txs := make([]*Transaction, count)
	for i := 0; i < int(count); i++ {
		b, err := dec.ReadBytes()
		if err != nil {
			return nil, err
		}
		tx, err := Deserialize(b)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}
	return txs, nil
}

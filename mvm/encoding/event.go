package encoding

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/gofrs/uuid"
)

//
// MTG => VM
// nonce || asset || amount || memo || members || threshold || sig
//
// VM => MTG
// nonce || asset || amount || memo || members || threshold
//
type Event struct {
	Process   string
	Asset     string
	Members   []string
	Threshold int
	Amount    common.Integer
	Memo      string
	Nonce     uint64
	Signature []byte
}

func (e *Event) Encode() []byte {
	enc := common.NewEncoder()
	enc.WriteUint64(e.Nonce)
	writeUUID(enc, e.Asset)
	enc.WriteInteger(e.Amount)
	writeString(enc, e.Memo)

	if len(e.Members) > 64 {
		panic(len(e.Members))
	}
	enc.WriteInt(len(e.Members))
	for _, m := range e.Members {
		writeUUID(enc, m)
	}
	if e.Threshold > len(e.Members) {
		panic(e.Threshold)
	}
	enc.WriteInt(e.Threshold)

	if len(e.Signature) > 128 {
		panic(e.Signature)
	}
	enc.WriteInt(len(e.Signature))
	enc.Write(e.Signature)

	return enc.Bytes()
}

func DecodeEvent(b []byte) (*Event, error) {
	dec := common.NewDecoder(b)
	nonce, err := dec.ReadUint64()
	if err != nil {
		return nil, err
	}
	asset, err := readUUID(dec)
	if err != nil {
		return nil, err
	}
	amount, err := dec.ReadInteger()
	if err != nil {
		return nil, err
	}
	memo, err := dec.ReadBytes()
	if err != nil {
		return nil, err
	}

	ml, err := dec.ReadInt()
	if err != nil {
		return nil, err
	}
	members := make([]string, ml)
	for i := 0; i < ml; i++ {
		m, err := readUUID(dec)
		if err != nil {
			return nil, err
		}
		members[i] = m
	}
	threshold, err := dec.ReadInt()
	if err != nil {
		return nil, err
	}
	sig, err := dec.ReadBytes()
	if err != nil {
		return nil, err
	}

	return &Event{
		Asset:     asset,
		Members:   members,
		Threshold: threshold,
		Amount:    amount,
		Memo:      string(memo),
		Nonce:     nonce,
		Signature: sig,
	}, nil
}

func writeUUID(enc *common.Encoder, id string) {
	uid, err := uuid.FromString(id)
	if err != nil {
		panic(err)
	}
	enc.Write(uid.Bytes())
}

func writeString(enc *common.Encoder, s string) {
	if len(s) > 128 {
		panic(s)
	}
	enc.WriteInt(len(s))
	enc.Write([]byte(s))
}

func readUUID(dec *common.Decoder) (string, error) {
	var b [16]byte
	err := dec.Read(b[:])
	if err != nil {
		return "", err
	}
	id, err := uuid.FromBytes(b[:])
	return id.String(), err
}

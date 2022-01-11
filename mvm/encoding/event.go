package encoding

import (
	"fmt"

	"github.com/MixinNetwork/mixin/common"
	"github.com/gofrs/uuid"
)

//
// MTG => VM
// process || nonce || asset || amount || extra || timestamp || members || threshold || sig
//
// VM => MTG
// process || nonce || asset || amount || extra || timestamp || members || threshold
//
type Event struct {
	Process   string
	Asset     string
	Members   []string // need to do user mask per process
	Threshold int
	Amount    common.Integer
	Extra     []byte
	Timestamp uint64
	Nonce     uint64
	Signature []byte
}

func (e *Event) ID() string {
	return fmt.Sprintf("%s:%16x", e.Process, e.Nonce)
}

func (e *Event) Encode() []byte {
	enc := common.NewEncoder()
	writeUUID(enc, e.Process)
	enc.WriteUint64(e.Nonce)
	writeUUID(enc, e.Asset)
	enc.WriteInteger(e.Amount)
	writeBytes(enc, e.Extra)
	enc.WriteUint64(e.Timestamp)

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
	writeBytes(enc, e.Signature)

	return enc.Bytes()
}

func DecodeEvent(b []byte) (*Event, error) {
	dec := common.NewDecoder(b)
	process, err := readUUID(dec)
	if err != nil {
		return nil, err
	}
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
	extra, err := dec.ReadBytes()
	if err != nil {
		return nil, err
	}
	timestamp, err := dec.ReadUint64()
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
		Process:   process,
		Asset:     asset,
		Members:   members,
		Threshold: threshold,
		Amount:    amount,
		Extra:     extra,
		Timestamp: timestamp,
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

func writeBytes(enc *common.Encoder, b []byte) {
	if len(b) > 65*21 { //max 21 signers
		panic(b)
	}
	enc.WriteInt(len(b))
	enc.Write(b)
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

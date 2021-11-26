package eos

import (
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/gofrs/uuid"
)

func uuidToBytes(id string) []byte {
	uid, err := uuid.FromString(id)
	if err != nil {
		panic(err)
	}
	return uid.Bytes()
}

func bytesToUUID(b []byte) string {
	id, err := uuid.FromBytes(b[:])
	if err != nil {
		panic(err)
	}
	return id.String()
}

func reverseBytes(b []byte) []byte {
	r := make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		r[i] = b[len(b)-1-i]
	}
	return r
}

func convertTxRequestToEvent(req *TxLog) *encoding.Event {
	amount := common.Integer{}
	amount.UnmarshalMsgpack(reverseBytes(req.amount[:]))
	members := make([]string, len(req.members))
	for i := range req.members {
		members[i] = bytesToUUID(req.members[i][:])
	}
	return &encoding.Event{
		Process:   bytesToUUID(req.process[:]),
		Asset:     bytesToUUID(req.asset[:]),
		Members:   members,
		Threshold: int(req.threshold),
		Amount:    amount,
		Extra:     req.extra,
		Timestamp: req.timestamp,
		Nonce:     req.nonce,
	}
}

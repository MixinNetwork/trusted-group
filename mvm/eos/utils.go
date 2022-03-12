package eos

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/drand/kyber/share"
	"github.com/gofrs/uuid"
	"github.com/learnforpractice/goeoslib/chain"
	"github.com/learnforpractice/goeoslib/crypto/secp256k1"
)

const (
	TX_EXPIRATION = 15
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

func convertTxRequestToEvent(req *TxRequest) *encoding.Event {
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

func convertTxLogToEvent(req *TxLog) *encoding.Event {
	amount := common.Integer{}
	amount.UnmarshalMsgpack(reverseBytes(req.amount[:]))
	members := make([]string, len(req.members))
	for i := range req.members {
		members[i] = bytesToUUID(req.members[i][:])
	}
	extra := req.extra
	if len(extra) > 128 {
		extra = extra[:128]
	}
	return &encoding.Event{
		Process:   bytesToUUID(req.process[:]),
		Asset:     bytesToUUID(req.asset[:]),
		Members:   members,
		Threshold: int(req.threshold),
		Amount:    amount,
		Extra:     extra,
		Timestamp: req.timestamp,
		Nonce:     req.nonce,
	}
}

func genPrivateKey(share *share.PriShare) *secp256k1.PrivateKey {
	h := sha256.New()
	_, _ = share.V.MarshalTo(h)
	_ = binary.Write(h, binary.LittleEndian, share.I)
	key := secp256k1.NewPrivateKey(h.Sum(nil))
	return key
}

func BuildEventTransaction(mixincontract string, eventPublisher string, address string, event *encoding.Event, refBlockId string) (*chain.Transaction, error) {
	expiration := uint32(time.Now().Unix() + TX_EXPIRATION)
	tx := chain.NewTransaction(expiration)

	if len(refBlockId) != 64 {
		return nil, errors.New("Invalid reference block")
	}
	tx.SetReferenceBlock(refBlockId)

	var action *chain.Action
	if event.Nonce == 0 { //add process event
		logger.Verbosef("add process event %s", event.Process)
		addprocess := NewAddProcess(address, event.Process, event.Signature)
		action = chain.NewAction(
			&chain.PermissionLevel{Actor: chain.NewName(eventPublisher), Permission: chain.NewName("active")},
			chain.NewName(mixincontract),
			chain.NewName("addprocess"),
			addprocess,
		)
	} else {
		txEvent, err := convertEventToTxEvent(event)
		if err != nil {
			return nil, err
		}
		action = chain.NewAction(
			&chain.PermissionLevel{Actor: chain.NewName(eventPublisher), Permission: chain.NewName("active")},
			chain.NewName(address),
			chain.NewName("onevent"),
			txEvent,
		)
	}
	tx.Actions = append(tx.Actions, action)
	return tx, nil
}

func BuildErrorEventTransaction(eventPublisher string, address string, event *encoding.Event, refBlockId string, reason string) (*chain.Transaction, error) {
	expiration := uint32(time.Now().Unix() + TX_EXPIRATION)
	tx := chain.NewTransaction(expiration)

	if len(refBlockId) != 64 {
		return nil, errors.New("Invalid reference block")
	}
	tx.SetReferenceBlock(refBlockId)

	var action *chain.Action

	txEvent, err := convertEventToTxEvent(event)
	if err != nil {
		return nil, err
	}

	action = chain.NewAction(
		&chain.PermissionLevel{Actor: chain.NewName(eventPublisher), Permission: chain.NewName("active")},
		chain.NewName(address),
		chain.NewName("onerrorevent"),
		txEvent,
		reason,
	)
	tx.Actions = append(tx.Actions, action)
	return tx, nil
}

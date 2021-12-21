package eos

import (
	"crypto/sha256"

	"github.com/learnforpractice/goeoslib/chain"
	"github.com/learnforpractice/goeoslib/crypto/secp256k1"
)

type Signature struct {
	typ  uint8
	data [65]byte
}

type TxEvent struct {
	nonce      uint64
	process    chain.Uint128
	asset      chain.Uint128
	members    []chain.Uint128
	threshold  int32
	amount     chain.Uint128
	extra      []byte
	timestamp  uint64
	signatures []secp256k1.Signature
}

func (t *TxEvent) Pack() []byte {
	enc := chain.NewEncoder(t.Size())
	enc.PackUint64(t.nonce)
	enc.WriteBytes(t.process[:])
	enc.WriteBytes(t.asset[:])
	{
		enc.PackLength(len(t.members))
		for i := range t.members {
			enc.Pack(&t.members[i])
		}
	}

	enc.PackInt32(t.threshold)
	enc.WriteBytes(t.amount[:])
	enc.PackBytes(t.extra)
	enc.PackUint64(t.timestamp)

	enc.WriteUint8(uint8(len(t.signatures)))
	for i := range t.signatures {
		enc.WriteUint8(uint8(0)) //type
		enc.WriteBytes(t.signatures[i].Data[:])
	}
	return enc.GetBytes()
}

func (t *TxEvent) PackWithoutSignatures() []byte {
	enc := chain.NewEncoder(t.Size())
	enc.PackUint64(t.nonce)
	enc.WriteBytes(t.process[:])
	enc.WriteBytes(t.asset[:])
	{
		enc.PackLength(len(t.members))
		for i := range t.members {
			enc.Pack(&t.members[i])
		}
	}

	enc.PackInt32(t.threshold)
	enc.WriteBytes(t.amount[:])
	enc.PackBytes(t.extra)
	enc.PackUint64(t.timestamp)
	return enc.GetBytes()
}

func (t *TxEvent) Digest() *chain.Bytes32 {
	data := t.PackWithoutSignatures()
	hash := sha256.New()
	hash.Write(data)
	digest := hash.Sum(nil)
	return chain.NewBytes32(digest)
}

func (t *TxEvent) Sign(priv *secp256k1.PrivateKey) (*secp256k1.Signature, error) {
	digest := t.Digest()
	sig, err := priv.Sign(digest[:])
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func (t *TxEvent) Unpack(data []byte) (int, error) {
	var err error
	dec := chain.NewDecoder(data)
	t.nonce, err = dec.UnpackUint64()
	if err != nil {
		return 0, err
	}

	err = dec.UnpackI(&t.process)
	if err != nil {
		return 0, err
	}

	err = dec.UnpackI(&t.asset)
	if err != nil {
		return 0, err
	}

	{
		length, err := dec.UnpackLength()
		if err != nil {
			return 0, err
		}
		t.members = make([]chain.Uint128, length)
		for i := 0; i < length; i++ {
			_, err = dec.Unpack(&t.members[i])
			if err != nil {
				return 0, err
			}
		}
	}

	t.threshold, err = dec.UnpackInt32()
	if err != nil {
		return 0, err
	}

	err = dec.UnpackI(&t.amount)
	if err != nil {
		return 0, err
	}

	{
		length, err := dec.UnpackLength()
		if err != nil {
			return 0, err
		}
		t.extra = make([]byte, length)
		for i := 0; i < length; i++ {
			t.extra[i], err = dec.UnpackUint8()
			if err != nil {
				return 0, err
			}
		}
	}

	t.timestamp, err = dec.UnpackUint64()
	if err != nil {
		return 0, err
	}

	size, err := dec.UnpackUint8()
	if err != nil {
		return 0, err
	}
	t.signatures = make([]secp256k1.Signature, 0, size)
	for i := 0; i < int(size); i++ {
		signature := secp256k1.Signature{}
		err := dec.Read(signature.Data[:])
		if err != nil {
			return 0, err
		}
		t.signatures = append(t.signatures, signature)
	}

	return dec.Pos(), nil
}

func (t *TxEvent) Size() int {
	size := 0
	size += 8  //nonce
	size += 16 //process
	size += 16 //asset
	size += chain.PackedVarUint32Length(uint32(len(t.members)))
	for i := range t.members {
		size += chain.PackedVarUint32Length(uint32(len(t.members[i]))) + len(t.members[i])
	}
	size += 4  //threshold
	size += 16 //amount
	size += chain.PackedVarUint32Length(uint32(len(t.extra)))
	size += len(t.extra)
	size += 8 //timestamp
	size += chain.PackedVarUint32Length(uint32(len(t.signatures)))
	size += 66 * len(t.signatures)
	return size
}

type TxRequest struct {
	id        uint64 //primary : t.id
	nonce     uint64
	process   chain.Uint128
	asset     chain.Uint128
	members   []chain.Uint128 // need to do user mask per process
	threshold int32
	amount    chain.Uint128
	extra     []byte
	timestamp uint64
}

func (t *TxRequest) Pack() []byte {
	enc := chain.NewEncoder(t.Size())
	enc.PackUint64(t.id)
	enc.PackUint64(t.nonce)
	enc.WriteBytes(t.process[:])
	enc.WriteBytes(t.asset[:])
	{
		enc.PackLength(len(t.members))
		for i := range t.members {
			enc.Pack(&t.members[i])
		}
	}

	enc.PackInt32(t.threshold)
	enc.WriteBytes(t.amount[:])
	enc.PackBytes(t.extra)
	enc.PackUint64(t.timestamp)
	return enc.GetBytes()
}

func (t *TxRequest) Unpack(data []byte) (int, error) {
	var err error
	dec := chain.NewDecoder(data)
	t.id, err = dec.UnpackUint64()
	if err != nil {
		return 0, err
	}
	t.nonce, err = dec.UnpackUint64()
	if err != nil {
		return 0, err
	}

	err = dec.UnpackI(&t.process)
	if err != nil {
		return 0, err
	}

	err = dec.UnpackI(&t.asset)
	if err != nil {
		return 0, err
	}

	{
		length, err := dec.UnpackLength()
		if err != nil {
			return 0, err
		}

		t.members = make([]chain.Uint128, length)
		for i := 0; i < length; i++ {
			_, err = dec.Unpack(&t.members[i])
			if err != nil {
				return 0, err
			}
		}
	}

	t.threshold, err = dec.UnpackInt32()
	if err != nil {
		return 0, err
	}
	err = dec.UnpackI(&t.amount)
	if err != nil {
		return 0, err
	}
	{
		length, err := dec.UnpackLength()
		if err != nil {
			return 0, err
		}
		t.extra = make([]byte, length)
		for i := 0; i < length; i++ {
			t.extra[i], err = dec.UnpackUint8()
			if err != nil {
				return 0, err
			}
		}
	}

	t.timestamp, err = dec.UnpackUint64()
	if err != nil {
		return 0, err
	}
	return dec.Pos(), nil
}

func (t *TxRequest) Size() int {
	size := 0
	size += 8  //id
	size += 8  //nonce
	size += 16 //process
	size += 16 //asset
	size += chain.PackedVarUint32Length(uint32(len(t.members)))
	for i := range t.members {
		size += chain.PackedVarUint32Length(uint32(len(t.members[i]))) + len(t.members[i])
	}
	size += 4  //threshold
	size += 16 //amount
	size += chain.PackedVarUint32Length(uint32(len(t.extra)))
	size += len(t.extra)
	size += 8 //timestamp
	return size
}

type TxLog struct {
	id        uint64
	nonce     uint64
	contract  chain.Name
	process   chain.Uint128
	asset     chain.Uint128
	members   []chain.Uint128
	threshold int32
	amount    chain.Uint128
	extra     []byte
	timestamp uint64
}

func (t *TxLog) Pack() []byte {
	enc := chain.NewEncoder(t.Size())
	enc.PackUint64(t.id)
	enc.PackUint64(t.nonce)
	enc.PackUint64(t.contract.N)
	enc.WriteBytes(t.process[:])
	enc.WriteBytes(t.asset[:])
	{
		enc.PackLength(len(t.members))
		for i := range t.members {
			enc.WriteBytes(t.members[i][:])
		}
	}

	enc.PackInt32(t.threshold)
	enc.WriteBytes(t.amount[:])
	enc.PackBytes(t.extra)
	enc.PackUint64(t.timestamp)
	return enc.GetBytes()
}

func (t *TxLog) Unpack(data []byte) (int, error) {
	var err error

	dec := chain.NewDecoder(data)
	t.id, err = dec.UnpackUint64()
	if err != nil {
		return 0, err
	}

	t.nonce, err = dec.UnpackUint64()
	if err != nil {
		return 0, err
	}

	t.contract, err = dec.UnpackName()
	if err != nil {
		return 0, err
	}
	err = dec.UnpackI(&t.process)
	if err != nil {
		return 0, err
	}

	err = dec.UnpackI(&t.asset)
	if err != nil {
		return 0, err
	}

	{
		length, err := dec.UnpackLength()
		if err != nil {
			return 0, err
		}

		t.members = make([]chain.Uint128, length)
		for i := 0; i < length; i++ {
			err = dec.UnpackI(&t.members[i])
			if err != nil {
				return 0, err
			}
		}
	}

	t.threshold, err = dec.UnpackInt32()
	if err != nil {
		return 0, err
	}

	err = dec.UnpackI(&t.amount)
	if err != nil {
		return 0, err
	}

	t.extra, err = dec.UnpackBytes()
	if err != nil {
		return 0, err
	}

	t.timestamp, err = dec.UnpackUint64()
	if err != nil {
		return 0, err
	}

	return dec.Pos(), nil
}

func (t *TxLog) Size() int {
	size := 0
	size += 8  //id
	size += 8  //contract
	size += 16 //process
	size += 16 //asset
	size += chain.PackedVarUint32Length(uint32(len(t.members)))
	size += len(t.members) * 16
	size += 4  //threshold
	size += 16 //amount
	size += chain.PackedVarUint32Length(uint32(len(t.extra)))
	size += len(t.extra)
	size += 8 //timestamp
	return size
}

type AddProcess struct {
	address    chain.Name
	process    chain.Uint128
	signatures []secp256k1.Signature
}

func (t *AddProcess) Pack() []byte {
	enc := chain.NewEncoder(t.Size())
	enc.PackUint64(t.address.N)
	enc.WriteBytes(t.process[:])

	enc.WriteUint8(uint8(len(t.signatures)))
	for i := range t.signatures {
		enc.WriteUint8(uint8(0)) //type
		enc.WriteBytes(t.signatures[i].Data[:])
	}
	//enc.PackBytes(t.signature)
	return enc.GetBytes()
}

func (t *AddProcess) Unpack(data []byte) (int, error) {
	var err error
	dec := chain.NewDecoder(data)
	t.address, err = dec.UnpackName()
	if err != nil {
		return 0, err
	}

	err = dec.Read(t.process[:])
	if err != nil {
		return 0, err
	}
	return dec.Pos(), nil
}

func (t *AddProcess) PackWithoutSignatures() []byte {
	enc := chain.NewEncoder(8 + 16)
	enc.PackUint64(t.address.N)
	enc.WriteBytes(t.process[:])
	return enc.GetBytes()
}

func (t *AddProcess) Size() int {
	size := 0
	size += 8  //address
	size += 16 //process
	size += chain.PackedVarUint32Length(uint32(len(t.signatures)))
	size += 66 * len(t.signatures)
	return size
}

func (t *AddProcess) Digest() *chain.Bytes32 {
	data := t.PackWithoutSignatures()
	hash := sha256.New()
	hash.Write(data)
	digest := hash.Sum(nil)
	return chain.NewBytes32(digest)
}

func (t *AddProcess) Sign(priv *secp256k1.PrivateKey) (*secp256k1.Signature, error) {
	digest := t.Digest()
	sig, err := priv.Sign(digest[:])
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func NewAddProcess(address string, process string, signatures []byte) *AddProcess {
	addprocess := &AddProcess{}
	addprocess.address = chain.NewName(address)
	copy(addprocess.process[:], uuidToBytes(process))

	addprocess.signatures = make([]secp256k1.Signature, len(signatures)/65)
	for i := 0; i < len(signatures)/65; i++ {
		copy(addprocess.signatures[i].Data[:], signatures[i*65:i*65+65])
	}
	return addprocess
}

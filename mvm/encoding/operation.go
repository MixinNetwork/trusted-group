package encoding

import "github.com/MixinNetwork/mixin/common"

const (
	OperationPurposeUnknown       = 0
	OperationPurposeGroupEvent    = 1
	OperationPurposeAddProcess    = 11
	OperationPurposeCreditProcess = 12
)

type Operation struct {
	Purpose  int
	Process  string
	Platform string
	Address  string
	Extra    []byte
}

func (o *Operation) Encode() []byte {
	enc := common.NewEncoder()
	enc.WriteInt(o.Purpose)
	writeUUID(enc, o.Process)
	writeBytes(enc, []byte(o.Platform))
	writeBytes(enc, []byte(o.Address))
	writeBytes(enc, o.Extra)
	return enc.Bytes()
}

func DecodeOperation(b []byte) (*Operation, error) {
	dec := common.NewDecoder(b)
	purpose, err := dec.ReadInt()
	if err != nil {
		return nil, err
	}
	process, err := readUUID(dec)
	if err != nil {
		return nil, err
	}
	platform, err := dec.ReadBytes()
	if err != nil {
		return nil, err
	}
	address, err := dec.ReadBytes()
	if err != nil {
		return nil, err
	}
	extra, err := dec.ReadBytes()
	if err != nil {
		return nil, err
	}
	return &Operation{
		Purpose:  purpose,
		Process:  process,
		Platform: string(platform),
		Address:  string(address),
		Extra:    extra,
	}, nil
}

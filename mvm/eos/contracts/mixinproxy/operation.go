package main

import "github.com/uuosio/chain"

//packer ignore
type Operation struct {
	Purpose  int
	Process  []byte
	Platform string
	Address  string
	Extra    []byte
}

// func (o *Operation) Encode() []byte {
// 	enc := common.NewEncoder()
// 	enc.WriteInt(o.Purpose)
// 	writeUUID(enc, o.Process)
// 	writeBytes(enc, []byte(o.Platform))
// 	writeBytes(enc, []byte(o.Address))
// 	writeBytes(enc, o.Extra)
// 	return enc.Bytes()
// }

func readUint16(dec *chain.Decoder) uint16 {
	return (uint16(dec.ReadUint8()) << 8) + uint16(dec.ReadUint8())
}

func readBytes(dec *chain.Decoder) []byte {
	length := readUint16(dec)
	data := make([]byte, length)
	dec.Read(data)
	return data
}

func DecodeOperation(b []byte) *Operation {
	dec := chain.NewDecoder(b)
	op := &Operation{}

	op.Purpose = (int(dec.ReadUint8()) << 8) + int(dec.ReadUint8())
	op.Process = make([]byte, 16)
	dec.Read(op.Process[:])
	op.Platform = string(readBytes(dec))
	op.Address = string(readBytes(dec))
	op.Extra = readBytes(dec)
	return op
}

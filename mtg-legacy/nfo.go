package mtg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/gofrs/uuid/v5"
)

// MTG node boots should sync all the nfos from begining, and then it knows
// all the nfos state since the genisis. When MTG receives 0.001XIN payment,
// with the proper memo, and it should check that the nfo requested should
// not exceed the max, and should be by the same issuer.
//
// The MM API should never allow the memo of an nfo changes, unless from empty
// memo to a proper memo, or from a proper memo to a destroy memo. A destroy
// memo is only meant to be used by bridged nfo. A normal nfo transfer MUST
// just omit the memo.

const (
	NMPrefix  = "NFO"
	NMVersion = 0x00
)

var (
	NMDefaultCollectionKey = uuid.Nil.Bytes()

	NMDefaultChain uuid.UUID
	NMDefaultClass []byte
)

func init() {
	chain, err := uuid.FromString("43d61dcd-e413-450d-80b8-101d5e903357")
	if err != nil {
		panic(err)
	}
	class, err := hex.DecodeString("3c8c161a18ae2c8b14fda1216fff7da88c419b5d")
	if err != nil {
		panic(err)
	}
	NMDefaultChain = chain
	NMDefaultClass = class
}

// asset = hash(chain || class || collection || token)

type NFOMemo struct {
	Prefix  string
	Version byte

	Mask       uint64
	Chain      uuid.UUID // 16 bytes
	Class      []byte    // 64 bytes contract address
	Collection uuid.UUID // 16 bytes
	Token      []byte    // 64 bytes hash of content

	Extra []byte
}

func BuildMintNFO(collection string, token []byte, hash crypto.Hash) []byte {
	gid := uuid.FromStringOrNil(collection)
	nfo := NFOMemo{
		Prefix:     NMPrefix,
		Version:    NMVersion,
		Chain:      NMDefaultChain,
		Class:      NMDefaultClass,
		Collection: gid,
		Token:      token,
		Extra:      hash[:],
	}
	nfo.Mark([]int{0})
	return nfo.Encode()
}

func BuildExtraNFO(extra []byte) []byte {
	nfo := NFOMemo{
		Prefix:  NMPrefix,
		Version: NMVersion,
		Extra:   extra,
	}
	return nfo.Encode()
}

func (nm *NFOMemo) Mark(indexes []int) {
	for _, i := range indexes {
		if i >= 64 || i < 0 {
			panic(fmt.Errorf("invalid NFO memo index %d", i))
		}
		nm.Mask ^= (1 << uint64(i))
	}
}

func (nm *NFOMemo) Indexes() []int {
	keys := make([]int, 0)
	for i := uint64(0); i < 64; i++ {
		mask := uint64(1) << i
		if nm.Mask&mask == mask {
			keys = append(keys, int(i))
		}
	}
	return keys
}

func (nm *NFOMemo) WillMint() bool {
	return nm.Mask != 0
}

func (nm *NFOMemo) Encode() []byte {
	nw := new(nfoWriter)
	nw.write([]byte(nm.Prefix))
	nw.writeByte(nm.Version)
	if nm.Mask != 0 {
		nw.writeByte(1)
		nw.writeUint64(nm.Mask)
		nw.writeUUID(nm.Chain)
		nw.writeSlice(nm.Class)
		nw.writeSlice(nm.Collection.Bytes())
		nw.writeSlice(nm.Token)
		st := tokenBytesStrip(nm.Token)
		if bytes.Compare(nm.Token, st) != 0 {
			panic(hex.EncodeToString(nm.Token))
		}
	} else {
		nw.writeByte(0)
	}
	nw.writeSlice(nm.Extra)
	return nw.Bytes()
}

func DecodeNFOMemo(b []byte) (*NFOMemo, error) {
	if len(b) < 4 {
		return nil, fmt.Errorf("NFO length %d", len(b))
	}
	if string(b[:3]) != NMPrefix {
		return nil, fmt.Errorf("NFO prefix %v", b[:3])
	}
	if b[3] != NMVersion {
		return nil, fmt.Errorf("NFO version %v", b[3])
	}
	nr := &nfoReader{*bytes.NewReader(b[4:])}
	nm := &NFOMemo{
		Prefix:  NMPrefix,
		Version: NMVersion,
	}

	hint, err := nr.ReadByte()
	if err != nil {
		return nil, err
	}

	if hint == 1 {
		nm.Mask, err = nr.readUint64()
		if err != nil {
			return nil, err
		}
		if nm.Mask != 1 {
			return nil, fmt.Errorf("invalid mask %v", nm.Indexes())
		}
		nm.Chain, err = nr.readUUID()
		if err != nil {
			return nil, err
		}
		if nm.Chain != NMDefaultChain {
			return nil, fmt.Errorf("invalid chain %s", nm.Chain.String())
		}
		nm.Class, err = nr.readBytes()
		if err != nil {
			return nil, err
		}
		if bytes.Compare(nm.Class, NMDefaultClass) != 0 {
			return nil, fmt.Errorf("invalid class %s", hex.EncodeToString(nm.Class))
		}
		collection, err := nr.readBytes()
		if err != nil {
			return nil, err
		}
		nm.Collection, err = uuid.FromBytes(collection)
		if err != nil {
			return nil, err
		}
		nm.Token, err = nr.readBytes()
		if err != nil {
			return nil, err
		}
		st := tokenBytesStrip(nm.Token)
		if bytes.Compare(nm.Token, st) != 0 {
			return nil, fmt.Errorf("invalid token format %s", hex.EncodeToString(nm.Token))
		}
	}
	nm.Extra, err = nr.readBytes()
	if err != nil {
		return nil, err
	}

	return nm, nil
}

type nfoReader struct{ bytes.Reader }

func (nr *nfoReader) readUint64() (uint64, error) {
	var b [8]byte
	err := nr.read(b[:])
	if err != nil {
		return 0, err
	}
	d := binary.BigEndian.Uint64(b[:])
	return d, nil
}

func (nr *nfoReader) readUint32() (uint32, error) {
	var b [4]byte
	err := nr.read(b[:])
	if err != nil {
		return 0, err
	}
	d := binary.BigEndian.Uint32(b[:])
	return d, nil
}

func (nr *nfoReader) readUUID() (uuid.UUID, error) {
	var b [16]byte
	err := nr.read(b[:])
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.FromBytes(b[:])
}

func (nr *nfoReader) readBytes() ([]byte, error) {
	l, err := nr.ReadByte()
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return nil, nil
	}
	b := make([]byte, l)
	err = nr.read(b)
	return b, err
}

func (nr *nfoReader) read(b []byte) error {
	l, err := nr.Read(b)
	if err != nil {
		return err
	}
	if l != len(b) {
		return fmt.Errorf("data short %d %d", l, len(b))
	}
	return nil
}

type nfoWriter struct{ bytes.Buffer }

func (nw *nfoWriter) writeUUID(u uuid.UUID) {
	nw.write(u.Bytes())
}

func (nw *nfoWriter) writeSlice(b []byte) {
	l := len(b)
	if l >= 128 {
		panic(l)
	}
	nw.writeByte(byte(l))
	nw.write(b)
}

func (nw *nfoWriter) writeByte(b byte) {
	err := nw.WriteByte(b)
	if err != nil {
		panic(err)
	}
}

func (nw *nfoWriter) write(b []byte) {
	l, err := nw.Write(b)
	if err != nil {
		panic(err)
	}
	if l != len(b) {
		panic(b)
	}
}

func (nw *nfoWriter) writeUint64(d uint64) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, d)
	nw.write(b)
}

func (nw *nfoWriter) writeUint32(d uint32) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, d)
	nw.write(b)
}

func tokenBytesStrip(b []byte) []byte {
	b = new(big.Int).SetBytes(b).Bytes()
	if len(b) == 0 {
		return []byte{0}
	}
	return b
}

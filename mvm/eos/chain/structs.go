package chain

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type Bytes []byte

func (t Bytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString([]byte(t)))
}

func (a *Bytes) UnmarshalJSON(b []byte) error {
	bs, err := hex.DecodeString(strings.Trim(string(b), `"`))
	if err != nil {
		return err
	}
	*a = append((*a)[:0], bs...)
	return nil
}

type VarInt32 int32

func (t *VarInt32) Pack() []byte {
	return PackVarInt32(int32(*t))
}

func (t *VarInt32) Unpack(data []byte) (int, error) {
	v, n := UnpackVarInt32(data)
	*t = VarInt32(v)
	return n, nil
}

func (t *VarInt32) Size() int {
	return PackedVarInt32Length(int32(*t))
}

type VarUint32 uint32

func (t *VarUint32) Pack() []byte {
	return PackVarUint32(uint32(*t))
}

func (t *VarUint32) Unpack(data []byte) (int, error) {
	v, n := UnpackVarUint32(data)
	*t = VarUint32(v)
	return n, nil
}

func (t *VarUint32) Size() int {
	return PackedVarUint32Length(uint32(*t))
}

func (t *VarUint32) MarshalJSON() ([]byte, error) {
	return json.Marshal(uint32(*t))
}

type Int128 [16]byte

func (n *Int128) Pack() []byte {
	return n[:]
}

func (n *Int128) Unpack(data []byte) (int, error) {
	dec := NewDecoder(data)
	if err := dec.Read(n[:]); err != nil {
		return 0, err
	}
	return 16, nil
}

func (t *Int128) Size() int {
	return 16
}

type Uint128 [16]byte

func (n *Uint128) Pack() []byte {
	return n[:]
}

func (n *Uint128) Unpack(data []byte) (int, error) {
	dec := NewDecoder(data)
	if err := dec.Read(n[:]); err != nil {
		return 0, err
	}
	return 16, nil
}

func (t *Uint128) Size() int {
	return 16
}

func (n *Uint128) SetUint64(v uint64) {
	tmp := Uint128{}
	copy(n[:], tmp[:]) //memset
	binary.LittleEndian.PutUint64(n[:], v)
}

func (n *Uint128) Uint64() uint64 {
	return binary.LittleEndian.Uint64(n[:])
}

type Uint256 [32]uint8

func (n *Uint256) Pack() []byte {
	return n[:]
}

func (n *Uint256) Unpack(data []byte) (int, error) {
	dec := NewDecoder(data)
	if err := dec.Read(n[:]); err != nil {
		return 0, err
	}
	return 32, nil
}

func (t *Uint256) Size() int {
	return 32
}

func (n *Uint256) SetUint64(v uint64) {
	tmp := Uint256{}
	copy(n[:], tmp[:]) //memset
	binary.LittleEndian.PutUint64(n[:], v)
}

func (n *Uint256) Uint64() uint64 {
	return binary.LittleEndian.Uint64(n[:])
}

type Float128 [16]byte

func (n *Float128) Pack() []byte {
	return n[:]
}

func (n *Float128) Unpack(data []byte) (int, error) {
	dec := NewDecoder(data)
	if err := dec.Read(n[:]); err != nil {
		return 0, err
	}
	return 16, nil
}

func (t *Float128) Size() int {
	return 16
}

type TimePoint struct {
	Elapsed uint64
}

func (t *TimePoint) Pack() []byte {
	enc := NewEncoder(t.Size())
	enc.PackUint64(t.Elapsed)
	return enc.GetBytes()
}

func (t *TimePoint) Unpack(data []byte) (int, error) {
	dec := NewDecoder(data)
	dec.Unpack(&t.Elapsed)
	return 8, nil
}

func (t *TimePoint) Size() int {
	return 8
}

type TimePointSec struct {
	UTCSeconds uint32
}

func (t *TimePointSec) Pack() []byte {
	enc := NewEncoder(t.Size())
	enc.PackUint32(t.UTCSeconds)
	return enc.GetBytes()
}

func (t *TimePointSec) Unpack(data []byte) (int, error) {
	dec := NewDecoder(data)
	dec.Unpack(&t.UTCSeconds)
	return 4, nil
}

func (t *TimePointSec) Size() int {
	return 4
}

func (t TimePointSec) MarshalJSON() ([]byte, error) {
	s := time.Unix(int64(t.UTCSeconds), 0).UTC().Format("2006-01-02T15:04:05")
	return json.Marshal(s)
}

func (a *TimePointSec) UnmarshalJSON(b []byte) error {
	t, err := time.Parse("2006-01-02T15:04:05", strings.Trim(string(b), "\""))
	if err != nil {
		return newError(err)
	}
	a.UTCSeconds = uint32(t.Unix())
	return nil
}

type BlockTimestampType struct {
	Slot uint32
}

func (t *BlockTimestampType) Pack() []byte {
	enc := NewEncoder(t.Size())
	enc.PackUint32(t.Slot)
	return enc.GetBytes()
}

func (t *BlockTimestampType) Unpack(data []byte) (int, error) {
	dec := NewDecoder(data)
	dec.Unpack(&t.Slot)
	return 4, nil
}

func (t *BlockTimestampType) Size() int {
	return 4
}

type JsonObject map[string]interface{}

func NewJsonObjectFromBytes(data []byte) (JsonObject, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var x interface{}
	err := d.Decode(&x)
	if err != nil {
		return nil, err
	}
	return JsonObject(x.(map[string]interface{})), nil
}

func NewJsonObjectFromInterface(obj interface{}) (JsonObject, bool) {
	_obj, ok := obj.(map[string]interface{})
	if !ok {
		return nil, false
	}
	return JsonObject(_obj), true
}

//return string, []JsonObject, or map[string]JsonObject
func (b JsonObject) Get(keys ...interface{}) (interface{}, error) {
	if len(keys) == 0 {
		return nil, newErrorf("empty keys")
	}

	var value interface{}
	value = map[string]interface{}(b)

	for _, key := range keys {
		switch _key := key.(type) {
		case string:
			_value, ok := value.(map[string]interface{})
			if !ok {
				return nil, newErrorf("expect type:map[string]interface{}, got %T", value)
			}
			value, ok = _value[_key]
			if !ok {
				return nil, newErrorf("key %s not found", _key)
			}
		case int:
			_value, ok := value.([]interface{})
			if !ok {
				return nil, newErrorf("expect type:map[string]interface{}, got %T", value)
			}
			if _key < 0 || _key >= len(_value) {
				return nil, newErrorf("index out of range")
			}
			value = _value[_key]
		default:
			return nil, newErrorf("invalid key type: %T", key)
		}
	}
	return value, nil
}

func (b JsonObject) GetArray(keys ...interface{}) ([]interface{}, error) {
	v, err := b.Get(keys...)
	if err != nil {
		return nil, err
	}

	_v, ok := v.([]interface{})
	if !ok {
		return nil, newErrorf("value is not an array %T", v)
	}
	return _v, nil
}

func (b JsonObject) GetJsonObject(keys ...interface{}) (JsonObject, error) {
	v, err := b.Get(keys...)
	if err != nil {
		return nil, err
	}

	_v, ok := v.(map[string]interface{})
	if !ok {
		return nil, newErrorf("value is not an array %T", v)
	}
	return JsonObject(_v), nil
}

func (b JsonObject) GetString(keys ...interface{}) (string, error) {
	v, err := b.Get(keys...)
	if err != nil {
		return "", err
	}

	_v, ok := v.(string)
	if !ok {
		return "", newErrorf("value is not a string %T", v)
	}
	return _v, nil
}

func (b JsonObject) GetInt64(keys ...interface{}) (int64, error) {
	v, err := b.Get(keys...)
	if err != nil {
		return 0, err
	}

	if _v, ok := v.(json.Number); ok {
		return strconv.ParseInt(string(_v), 10, 64)
	} else {
		return 0, newErrorf("value is not a number %T", v)
	}
}

func (b JsonObject) GetUint64(keys ...interface{}) (uint64, error) {
	v, err := b.Get(keys...)
	if err != nil {
		return 0, err
	}

	if _v, ok := v.(json.Number); ok {
		return strconv.ParseUint(string(_v), 10, 64)
	} else {
		return 0, newErrorf("value is not a number %T", v)
	}
}

func (b JsonObject) GetTime(keys ...interface{}) (*time.Time, error) {
	v, err := b.GetString(keys...)
	if err != nil {
		return nil, err
	}

	t, err := time.Parse("2006-01-02T15:04:05", v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (b JsonObject) ToString() string {
	v, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}
	return string(v)
}

package mtg

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/vmihailenco/msgpack/v4"
)

func MsgpackMarshalPanic(val any) []byte {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf).UseCompactEncoding(true).SortMapKeys(true)
	err := enc.Encode(val)
	if err != nil {
		panic(fmt.Errorf("MsgpackMarshalPanic: %#v %s", val, err.Error()))
	}
	return buf.Bytes()
}

func MsgpackUnmarshal(data []byte, val any) error {
	err := msgpack.Unmarshal(data, val)
	if err == nil {
		return err
	}
	return fmt.Errorf("MsgpackUnmarshal: %s %s", hex.EncodeToString(data), err.Error())
}

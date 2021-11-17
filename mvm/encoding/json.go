package encoding

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func JSONMarshalPanic(val interface{}) []byte {
	b, err := json.Marshal(val)
	if err != nil {
		panic(fmt.Errorf("JSONMarshalPanic: %#v %s", val, err.Error()))
	}
	return b
}

func JSONUnmarshal(data []byte, val interface{}) error {
	err := json.Unmarshal(data, val)
	if err == nil {
		return err
	}
	return fmt.Errorf("JSONUnmarshal: %s %s", hex.EncodeToString(data), err.Error())
}

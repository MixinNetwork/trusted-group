package chain

import (
	"encoding/hex"
	"errors"
	"fmt"
)

func DecodeHash256(hash string) ([]byte, error) {
	_hash, err := hex.DecodeString(hash)
	if err != nil {
		return nil, newError(err)
	}
	if len(_hash) != 32 {
		return nil, newErrorf("invalid hash")
	}
	return _hash, nil
}

func newError(err error) error {
	return err
}

func newErrorf(format string, args ...interface{}) error {
	errMsg := fmt.Sprintf(format, args...)
	return errors.New(errMsg)
}

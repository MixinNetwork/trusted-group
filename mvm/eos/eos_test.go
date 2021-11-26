package eos

import (
	"encoding/binary"
	"log"
	"math/big"
	"testing"
)

func TestBigInt(t *testing.T) {
	b := [8]byte{}
	binary.BigEndian.PutUint64(b[:], uint64(100))
	a := big.NewInt(1)
	a.SetBytes(b[:])
	log.Println("++++a:", a)
}

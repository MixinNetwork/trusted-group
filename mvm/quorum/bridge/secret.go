package main

import (
	"crypto/ed25519"
	"encoding/hex"

	"filippo.io/edwards25519"
)

func PublicKeyToCurve25519(publicKey ed25519.PublicKey) ([]byte, error) {
	p, err := (&edwards25519.Point{}).SetBytes(publicKey[:])
	if err != nil {
		return nil, err
	}
	return p.BytesMontgomery(), nil
}

func CurvePublicKey(public string) []byte {
	buf, err := hex.DecodeString(public)
	if err != nil {
		panic(err)
	}
	curve25519Public, err := PublicKeyToCurve25519(ed25519.PublicKey(buf))
	if err != nil {
		panic(err)
	}
	return curve25519Public
}

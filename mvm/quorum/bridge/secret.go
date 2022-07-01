package main

import (
	"crypto/ed25519"
	"encoding/base64"

	"filippo.io/edwards25519"
)

func PublicKeyToCurve25519(publicKey ed25519.PublicKey) ([]byte, error) {
	p, err := (&edwards25519.Point{}).SetBytes(publicKey[:])
	if err != nil {
		return nil, err
	}
	return p.BytesMontgomery(), nil
}

func CurvePublicKey(public string) string {
	buf, err := base64.RawURLEncoding.DecodeString(public)
	if err != nil {
		panic(err)
	}
	curve25519Public, err := PublicKeyToCurve25519(ed25519.PublicKey(buf))
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(curve25519Public)
}

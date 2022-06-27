package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	"filippo.io/edwards25519"
)

func PublicKeyToCurve25519(curve25519Public *[32]byte, publicKey ed25519.PublicKey) error {
	A, err := edwards25519.NewIdentityPoint().SetBytes(publicKey[:32])
	if err != nil {
		return fmt.Errorf("Invalid public key %x", publicKey)
	}
	x := A.BytesMontgomery()
	copy(curve25519Public[:], x[:])
	return nil
}

func ServerCurvePublic(public string) string {
	buf, err := base64.RawURLEncoding.DecodeString(public)
	if err != nil {
		panic(err)
	}
	var curve25519Public [32]byte
	err = PublicKeyToCurve25519(&curve25519Public, ed25519.PublicKey(buf))
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(curve25519Public[:])
}

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"

	"filippo.io/edwards25519"
	"golang.org/x/crypto/curve25519"
)

func PublicKeyToCurve25519(publicKey ed25519.PublicKey) ([]byte, error) {
	p, err := (&edwards25519.Point{}).SetBytes(publicKey[:])
	if err != nil {
		return nil, err
	}
	return p.BytesMontgomery(), nil
}

func CurvePublicKey(public string) string {
	buf, err := hex.DecodeString(public)
	if err != nil {
		panic(err)
	}
	curve25519Public, err := PublicKeyToCurve25519(ed25519.PublicKey(buf))
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(curve25519Public)
}

func PrivateKeyToCurve25519(curve25519Private *[32]byte, seed []byte) {
	h := sha512.New()
	h.Write(seed)
	digest := h.Sum(nil)

	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	copy(curve25519Private[:], digest)
}

func SharedKey(public []byte) [32]byte {
	curve25519Public, err := PublicKeyToCurve25519(ed25519.PublicKey(public))
	if err != nil {
		panic(err)
	}
	seed, err := hex.DecodeString(ServerPrivate)
	if err != nil {
		panic(err)
	}

	var dst, priv, pub [32]byte
	PrivateKeyToCurve25519(&priv, seed)
	copy(pub[:], curve25519Public[:])
	curve25519.ScalarMult(&dst, &priv, &pub)
	return dst
}

func aesEncryptCBC(key, msg []byte) []byte {
	padding := aes.BlockSize - len(msg)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	msg = append(msg, padtext...)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(hex.EncodeToString(key))
	}
	ciphertext := make([]byte, aes.BlockSize+len(msg))
	iv := ciphertext[:aes.BlockSize]
	n, err := io.ReadFull(rand.Reader, iv)
	if n != aes.BlockSize || err != nil {
		panic(err)
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], msg)
	return ciphertext
}

func aesDecryptCBC(key, ciphertext []byte) ([]byte, error) {
	if cl := len(ciphertext); cl < aes.BlockSize || cl%aes.BlockSize != 0 {
		return nil, fmt.Errorf("AES cipher text invalid length %d", cl)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	iv := ciphertext[:aes.BlockSize]
	source := ciphertext[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(source, source)

	length := len(source)
	unpadding := int(source[length-1])
	if unpadding > length {
		return nil, fmt.Errorf("AES CBC padding invalid %d %d", unpadding, length)
	}
	return source[:length-unpadding], nil
}

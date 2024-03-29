package crypto

import (
	"encoding/hex"
	"fmt"

	"github.com/MixinNetwork/trusted-group/mvm/crypto/en256"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/drand/kyber"
	"github.com/drand/kyber/sign/bls"
)

const (
	KeyVersion = 'T'
)

func Sign(scalar kyber.Scalar, msg []byte) ([]byte, error) {
	scheme := bls.NewSchemeOnG1(en256.NewSuiteG2())
	return scheme.Sign(scalar, msg)
}

func Verify(pub kyber.Point, msg, sig []byte) error {
	scheme := bls.NewSchemeOnG1(en256.NewSuiteG2())
	return scheme.Verify(pub, msg, sig)
}

func PrivateKeyFromHex(s string) (kyber.Scalar, error) {
	seed, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	suite := en256.NewSuiteG2()
	scalar := suite.Scalar().SetBytes(seed)
	return scalar, nil
}

func PrivateKeyBytes(scalar kyber.Scalar) []byte {
	b, err := scalar.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return b
}

func PublicKey(scalar kyber.Scalar) kyber.Point {
	suite := en256.NewSuiteG2()
	return suite.Point().Mul(scalar, nil)
}

func PublicKeyString(point kyber.Point) string {
	b := PublicKeyBytes(point)
	return base58.CheckEncode(b, KeyVersion)
}

func PublicKeyBytes(point kyber.Point) []byte {
	b, err := point.MarshalBinary()
	if err != nil {
		panic(err)
	}
	return b
}

func PubKeyFromBytes(b []byte) (kyber.Point, error) {
	suite := en256.NewSuiteG2()
	point := suite.G2().Point()
	err := point.UnmarshalBinary(b)
	return point, err
}

func PubKeyFromBase58(s string) (kyber.Point, error) {
	b, ver, err := base58.CheckDecode(s)
	if err != nil {
		return nil, err
	}
	if ver != KeyVersion {
		return nil, fmt.Errorf("invalid version %d", ver)
	}
	return PubKeyFromBytes(b)
}

package secp256k1

/*
#cgo CFLAGS: -I./libsecp256k1-zkp
#cgo CFLAGS: -I./libsecp256k1-zkp/include
#cgo CFLAGS: -I./libsecp256k1-zkp/src/

//#define HAVE_CONFIG_H 1
#include <stdbool.h>

#ifdef __SIZEOF_INT128__
#  define HAVE___INT128
#  define USE_FIELD_5X52
#  define USE_SCALAR_4X64
#else
#  define USE_FIELD_10X26
#  define USE_SCALAR_8X32
#endif
#define USE_ENDOMORPHISM
#define USE_NUM_NONE
#define USE_FIELD_INV_BUILTIN
#define USE_SCALAR_INV_BUILTIN
#define NDEBUG

#define secp256k1_context_destroy __secp256k1_context_destroy
#define secp256k1_ec_seckey_verify __secp256k1_ec_seckey_verify
#define secp256k1_ecdsa_verify __secp256k1_ecdsa_verify
#define secp256k1_nonce_function_default __secp256k1_nonce_function_default
#define secp256k1_ecdsa_sign __secp256k1_ecdsa_sign
#define secp256k1_ec_privkey_tweak_mul __secp256k1_ec_privkey_tweak_mul
#define secp256k1_ec_pubkey_tweak_mul __secp256k1_ec_pubkey_tweak_mul
#define secp256k1_context_randomize __secp256k1_context_randomize
#define secp256k1_ec_pubkey_create __secp256k1_ec_pubkey_create
#define secp256k1_context_create __secp256k1_context_create
#define secp256k1_context_clone __secp256k1_context_clone
#define secp256k1_ec_privkey_tweak_add __secp256k1_ec_privkey_tweak_add
#define secp256k1_ec_pubkey_tweak_add __secp256k1_ec_pubkey_tweak_add
#define secp256k1_nonce_function_rfc6979 __secp256k1_nonce_function_rfc6979


#include "./libsecp256k1-zkp/src/secp256k1.c"

static secp256k1_context_t *gctx = NULL;
static void init_context() {
    gctx = secp256k1_context_create(SECP256K1_CONTEXT_SIGN | SECP256K1_CONTEXT_VERIFY | SECP256K1_CONTEXT_COMMIT | SECP256K1_CONTEXT_RANGEPROOF);
}

static void destroy_context() {
	secp256k1_context_destroy(gctx);
	gctx = NULL;
}

static int extended_nonce_function( unsigned char *nonce32, const unsigned char *msg32,
									const unsigned char *key32, unsigned int attempt,
									const void *data ) {
	unsigned int* extra = (unsigned int*) data;
	(*extra)++;
	return secp256k1_nonce_function_default( nonce32, msg32, key32, *extra, 0 );
}

static int is_canonical( const unsigned char* data ) {
	return !(data[1] & 0x80)
			&& !(data[1] == 0 && !(data[2] & 0x80))
			&& !(data[33] & 0x80)
			&& !(data[33] == 0 && !(data[34] & 0x80));
}

static int sign_compact( const unsigned char* digest, const unsigned char *seckey, size_t seckey_size, bool require_canonical, unsigned char* compact_signature, size_t compact_signature_len) {
	int ret = 0;
	int recid;
	unsigned int counter = 0;
	do
	{
		ret = secp256k1_ecdsa_sign_compact( gctx, digest, compact_signature + 1, seckey, extended_nonce_function, &counter, &recid );
		if (ret == 0)
		{
			return 0;
		}
	} while( require_canonical && !is_canonical( compact_signature ) );
	compact_signature[0] = 27 + 4 + recid;
	return 1;
}

static int secp256k1_recover( const unsigned char* signature, size_t signature_size, const unsigned char* digest, size_t digest_size, unsigned char* pub_key, size_t pub_key_size, bool check_canonical )
{
	int nV = signature[0];
	if (nV<27 || nV>=35) {
		//"unable to reconstruct public key from signature" );
		return 0;
	}

	if( check_canonical ) {
		if (!is_canonical( signature )) {
			//"signature is not canonical"
			return 0;
		}
	}

	unsigned int pk_len;
	int ret = secp256k1_ecdsa_recover_compact( gctx, digest, signature + 1, pub_key, (int*) &pk_len, 1, (signature[0] - 27) & 3 );
	if (ret == 0) {
		return 0;
	}
	return pk_len == pub_key_size;
}

static int secp256k1_get_public_key(const unsigned char* seckey, size_t seckey_size, unsigned char* pubkey, size_t pubkey_size)
{
	unsigned int pk_len;
	int ret = secp256k1_ec_pubkey_create( gctx, pubkey, (int*) &pk_len, seckey, 1 );
	if (ret == 0) {
		return 0;
	}

	if (pk_len != pubkey_size) {
		return 0;
	}
	return 1;
}
*/
import "C"

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"unsafe"

	"crypto/sha256"

	"github.com/akamensky/base58"
	"golang.org/x/crypto/ripemd160"
)

func SayHello() {
	println("Hello!!!!")
}

var gInit = false

func Init() bool {
	if !gInit {
		C.init_context()
		gInit = true
	}
	return true
}

var (
	_init = Init()
)

func Destroy() {
	if gInit {
		C.destroy_context()
		gInit = false
	}
}

type PublicKey struct {
	Data [33]byte
}

func (pk *PublicKey) Bytes() []byte {
	return pk.Data[:]
}

func (pk *PublicKey) StringEOS() string {
	hash := ripemd160.New()
	hash.Write(pk.Data[:])
	digest := hash.Sum(nil)

	pub := pk.Data[:]
	pub = append(pub, digest[:4]...)
	return "EOS" + base58.Encode(pub)
}

func (pk *PublicKey) String() string {
	hash := ripemd160.New()
	hash.Write(pk.Data[:])
	hash.Write([]byte("K1"))
	digest := hash.Sum(nil)

	pub := pk.Data[:]
	pub = append(pub, digest[:4]...)
	return "PUB_K1_" + base58.Encode(pub)
}

func NewPublicKeyFromBase58(strPub string) (*PublicKey, error) {
	if strings.HasPrefix(strPub, "EOS") {
		strPub = strPub[3:]
		pub, err := base58.Decode(strPub)
		if err != nil {
			return nil, err
		}

		if len(pub) != 37 {
			return nil, errors.New("Invalid public key length")
		}

		hash := ripemd160.New()
		hash.Write(pub[:33])
		digest := hash.Sum(nil)
		if !bytes.Equal(pub[33:], digest[:4]) {
			return nil, errors.New("Invalid public key")
		}

		_pub := &PublicKey{}
		copy(_pub.Data[:], pub[:])
		return _pub, nil
	} else if strings.HasPrefix(strPub, "PUB_K1_") {
		strPub = strPub[len("PUB_K1_"):]
		pub, err := base58.Decode(strPub)
		if err != nil {
			return nil, err
		}

		if len(pub) != 37 {
			return nil, errors.New("Invalid public key length")
		}

		hash := ripemd160.New()
		hash.Write(pub[:33])
		hash.Write([]byte("K1"))
		digest := hash.Sum(nil)
		if !bytes.Equal(pub[33:], digest[:4]) {
			return nil, errors.New("Invalid public key")
		}

		_pub := &PublicKey{}
		copy(_pub.Data[:], pub[:])
		return _pub, nil
	} else {
		return nil, errors.New("Invalid public key format")
	}
}

type PrivateKey struct {
	Data [32]byte
}

func NewPrivateKey(seed []byte) *PrivateKey {
	priv := &PrivateKey{}
	copy(priv.Data[:], seed)
	return priv
}

func NewPrivateKeyFromHex(strPriv string) (*PrivateKey, error) {
	seed, err := hex.DecodeString(strPriv)
	if err != nil {
		return nil, err
	}

	priv := &PrivateKey{}
	copy(priv.Data[:], seed)
	return priv, nil
}

func NewPrivateKeyFromBase58(strPriv string) (*PrivateKey, error) {
	seed, err := base58.Decode(strPriv)
	if err != nil {
		return nil, err
	}

	hash := sha256.New()
	hash.Write(seed[:33])
	digest := hash.Sum(nil)

	hash = sha256.New()
	hash.Write(digest)
	digest = hash.Sum(nil)

	if !bytes.Equal(seed[33:], digest[:4]) {
		return nil, errors.New("Invalid private key")
	}

	priv := &PrivateKey{}
	copy(priv.Data[:], seed[1:])
	return priv, nil
}

func (priv *PrivateKey) String() string {
	seed := []byte{0x80}
	seed = append(seed, priv.Data[:]...)
	hash := sha256.New()
	hash.Write(seed)
	digest := hash.Sum(nil)

	hash = sha256.New()
	hash.Write(digest)
	digest = hash.Sum(nil)

	seed = append(seed, digest[:4]...)
	return base58.Encode(seed)
}

func (priv *PrivateKey) GetPublicKey() *PublicKey {
	var pubkey [33]byte
	_seckey := (*C.uchar)(unsafe.Pointer(&priv.Data[0]))
	_pubkey := (*C.uchar)(unsafe.Pointer(&pubkey))
	ret := C.secp256k1_get_public_key(_seckey, 32, _pubkey, 33)
	if ret == 0 {
		panic("secp256k1_get_public_key failed")
		return nil
	}
	return &PublicKey{pubkey}
}

func (priv *PrivateKey) Sign(digest []byte) (*Signature, error) {
	if len(digest) != 32 {
		return nil, errors.New("Invalid digest length")
	}

	signature := make([]byte, 65)
	_digest := (*C.uchar)(unsafe.Pointer(&digest[0]))
	_seckey := (*C.uchar)(unsafe.Pointer(&priv.Data[0]))
	_signature := (*C.uchar)(unsafe.Pointer(&signature[0]))
	ret := C.sign_compact(_digest, _seckey, 32, (C.bool)(true), _signature, 65)
	if ret == 0 {
		return nil, errors.New("sign failed")
	}
	return NewSignature(signature), nil
}

type Signature struct {
	Data [65]byte
}

func NewSignature(sig []byte) *Signature {
	s := &Signature{}
	copy(s.Data[:], sig)
	return s
}

func NewSignatureFromBase58(sig string) (*Signature, error) {
	if !strings.HasPrefix(sig, "SIG_K1_") {
		return nil, errors.New("Invalid signature")
	}
	_sig := sig[len("SIG_K1_"):]
	__sig, err := base58.Decode(_sig)
	if err != nil {
		return nil, err
	}
	if len(__sig) != 65+4 {
		return nil, errors.New("Invalid signature length")
	}

	hash := ripemd160.New()
	hash.Write(__sig[:65])
	hash.Write([]byte("K1"))
	digest := hash.Sum(nil)

	if !bytes.Equal(__sig[65:], digest[:4]) {
		return nil, errors.New("Invalid signature: checksum mismatch")
	}

	s := &Signature{}
	copy(s.Data[:], __sig)
	return s, nil
}

func (sig *Signature) Bytes() []byte {
	return sig.Data[:]
}

func (sig *Signature) String() string {
	buf := make([]byte, 0, len(sig.Data)+4)
	buf = append(buf, sig.Data[:]...)

	hash := ripemd160.New()
	hash.Write(sig.Data[:])
	hash.Write([]byte("K1"))
	digest := hash.Sum(nil)
	buf = append(buf, digest[:4]...)

	return "SIG_K1_" + base58.Encode(buf[:])
}

func Sign(digest []byte, seckey *PrivateKey) (*Signature, error) {
	if len(digest) != 32 {
		return nil, errors.New("Invalid digest length")
	}
	return seckey.Sign(digest)
}

//digest is 32 bytes
//signature is 65 bytes
//pubkey is 33 bytes
func Recover(digest []byte, signature *Signature) (*PublicKey, error) {
	_digest := (*C.uchar)(unsafe.Pointer(&digest[0]))
	_signature := (*C.uchar)(unsafe.Pointer(&signature.Data[0]))

	var pubkey_recovered [33]byte
	_pubkey_recovered := (*C.uchar)(unsafe.Pointer(&pubkey_recovered[0]))
	ret := C.secp256k1_recover(_signature, 65, _digest, 32, _pubkey_recovered, 33, (C.bool)(true))
	if ret == 0 {
		return nil, errors.New("recover failed")
	}
	return &PublicKey{pubkey_recovered}, nil
}

func GetPublicKey(seckey *PrivateKey) (*PublicKey, error) {
	if len(seckey.Data) != 32 {
		return nil, errors.New("seckey must be 32 bytes")
	}

	var pubkey [33]byte
	_seckey := (*C.uchar)(unsafe.Pointer(&seckey.Data[0]))
	_pubkey := (*C.uchar)(unsafe.Pointer(&pubkey))
	ret := C.secp256k1_get_public_key(_seckey, 32, _pubkey, 33)
	if ret == 0 {
		return nil, errors.New("get public key failed")
	}
	return &PublicKey{pubkey}, nil
}

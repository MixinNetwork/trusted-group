package chain

import (
	"github.com/MixinNetwork/trusted-group/mvm/eos/secp256k1"
)

type Wallet struct {
	keys map[string]*secp256k1.PrivateKey
}

var gWallet *Wallet

func GetWallet() *Wallet {
	if gWallet == nil {
		gWallet = &Wallet{}
		gWallet.keys = make(map[string]*secp256k1.PrivateKey)
	}
	return gWallet
}

func (w *Wallet) Import(name string, strPriv string) error {
	priv, err := secp256k1.NewPrivateKeyFromBase58(strPriv)
	if err != nil {
		return newError(err)
	}

	pub := priv.GetPublicKey()
	w.keys[pub.StringEOS()] = priv
	return nil
}

//GetPublicKeys
func (w *Wallet) GetPublicKeys() []string {
	keys := make([]string, 0, len(w.keys))
	for k := range w.keys {
		keys = append(keys, k)
	}
	return keys
}

func (w *Wallet) GetPrivateKey(pubKey string) (*secp256k1.PrivateKey, error) {
	priv, ok := w.keys[pubKey]
	if !ok {
		return nil, newErrorf("not found")
	}
	return priv, nil
}

func (w *Wallet) Sign(digest *Bytes32, pubKey string) (*secp256k1.Signature, error) {
	pub, err := secp256k1.NewPublicKeyFromBase58(pubKey)
	if err != nil {
		return nil, newError(err)
	}

	priv, ok := w.keys[pub.StringEOS()]
	if !ok {
		return nil, newErrorf("not found")
	}
	sig, err := priv.Sign(digest[:])
	if err != nil {
		return nil, newError(err)
	}
	return sig, nil
}

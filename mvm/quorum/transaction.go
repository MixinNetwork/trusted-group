package quorum

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/shopspring/decimal"
)

func (e *Engine) signContractNotifierDepositTransaction(pub string, key string, amount decimal.Decimal, nonce uint64) (string, string) {
	return e.signTransaction(pub, key, amount, nil, nonce)
}

func (e *Engine) signGroupEventTransaction(contract string, evt *encoding.Event, notifier string) (string, string) {
	data := EventMethod + fmt.Sprintf("%064x", 0x20)
	data = data + fmt.Sprintf("%064x", len(evt.Encode()))
	data = data + hex.EncodeToString(evt.Encode())
	for p := len(evt.Encode()) % 32; p > 0 && p < 32; p++ {
		data = data + "00"
	}
	db, err := hex.DecodeString(data[2:])
	if err != nil {
		panic(err)
	}
	return e.signTransaction(contract, notifier, decimal.Zero, db, evt.Nonce)
}

func (e *Engine) signTransaction(to string, key string, amount decimal.Decimal, data []byte, nonce uint64) (string, string) {
	ecdsaPriv, err := crypto.HexToECDSA(key)
	if err != nil {
		panic(err)
	}

	cb, err := hex.DecodeString(to[2:])
	if err != nil {
		panic(err)
	}
	var address common.Address
	copy(address[:], cb)

	gasPrice := big.NewInt(GasPrice)
	amt := amount.Mul(decimal.New(1, etherPrecision)).BigInt()
	tx := types.NewTransaction(nonce, address, amt, GasLimit, gasPrice, data)
	params := params.MainnetChainConfig
	params.ChainID = big.NewInt(ChainID)
	signer := types.MakeSigner(params, params.EIP155Block)
	tx, err = types.SignTx(tx, signer, ecdsaPriv)
	if err != nil {
		panic(err)
	}

	rb, err := tx.MarshalBinary()
	if err != nil {
		panic(err)
	}
	id := tx.Hash().Hex()
	return id, "0x" + fmt.Sprintf("%x", rb)
}

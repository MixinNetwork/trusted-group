package quorum

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/trusted-group/mvm/encoding"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func (e *Engine) SignGroupEventTransaction(notifier string, contract string, evt *encoding.Event) {
	ecdsaPriv, err := crypto.HexToECDSA(notifier)
	if err != nil {
		panic(err)
	}

	cb, err := hex.DecodeString(contract[2:])
	if err != nil {
		panic(err)
	}
	var address common.Address
	copy(address[:], cb)

	data := EventMethod + fmt.Sprintf("%064x", 0x20)
	data = data + fmt.Sprintf("%064x", len(evt.Encode()))
	data = data + hex.EncodeToString(evt.Encode())
	for p := len(evt.Encode()) % 32; p > 0 && p < 32; p++ {
		data = data + "00"
	}
	db, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}

	gasPrice := new(big.Int).SetUint64(GasPrice)
	tx := types.NewTransaction(evt.Nonce, address, new(big.Int), GasLimit, gasPrice, db)
	signer := types.MakeSigner(params.MainnetChainConfig, params.MainnetChainConfig.LondonBlock)
	tx, err = types.SignTx(tx, signer, ecdsaPriv)
	if err != nil {
		panic(err)
	}

	txId := tx.Hash().Hex()
	rb, err := tx.MarshalBinary()
	if err != nil {
		panic(err)
	}
	raw := fmt.Sprintf("%x", rb)
	logger.Println(txId, raw)
}

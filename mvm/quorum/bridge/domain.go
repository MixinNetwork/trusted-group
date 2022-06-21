package main

import (
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/domains/akash"
	"github.com/MixinNetwork/mixin/domains/algorand"
	"github.com/MixinNetwork/mixin/domains/arweave"
	"github.com/MixinNetwork/mixin/domains/avalanche"
	"github.com/MixinNetwork/mixin/domains/bch"
	"github.com/MixinNetwork/mixin/domains/binance"
	"github.com/MixinNetwork/mixin/domains/bitcoin"
	"github.com/MixinNetwork/mixin/domains/bsv"
	"github.com/MixinNetwork/mixin/domains/cosmos"
	"github.com/MixinNetwork/mixin/domains/dash"
	"github.com/MixinNetwork/mixin/domains/decred"
	"github.com/MixinNetwork/mixin/domains/dfinity"
	"github.com/MixinNetwork/mixin/domains/dogecoin"
	"github.com/MixinNetwork/mixin/domains/eos"
	"github.com/MixinNetwork/mixin/domains/etc"
	"github.com/MixinNetwork/mixin/domains/ethereum"
	"github.com/MixinNetwork/mixin/domains/filecoin"
	"github.com/MixinNetwork/mixin/domains/handshake"
	"github.com/MixinNetwork/mixin/domains/horizen"
	"github.com/MixinNetwork/mixin/domains/kusama"
	"github.com/MixinNetwork/mixin/domains/litecoin"
	"github.com/MixinNetwork/mixin/domains/mobilecoin"
	"github.com/MixinNetwork/mixin/domains/monero"
	"github.com/MixinNetwork/mixin/domains/namecoin"
	"github.com/MixinNetwork/mixin/domains/near"
	"github.com/MixinNetwork/mixin/domains/nervos"
	"github.com/MixinNetwork/mixin/domains/polkadot"
	"github.com/MixinNetwork/mixin/domains/polygon"
	"github.com/MixinNetwork/mixin/domains/ravencoin"
	"github.com/MixinNetwork/mixin/domains/ripple"
	"github.com/MixinNetwork/mixin/domains/siacoin"
	"github.com/MixinNetwork/mixin/domains/solana"
	"github.com/MixinNetwork/mixin/domains/stellar"
	"github.com/MixinNetwork/mixin/domains/tezos"
	"github.com/MixinNetwork/mixin/domains/tron"
	"github.com/MixinNetwork/mixin/domains/zcash"
)

func verifyDestination(chainId crypto.Hash, addr string) error {
	switch chainId {
	case ethereum.EthereumChainId:
		return ethereum.VerifyAddress(addr)
	case etc.EthereumClassicChainId:
		return etc.VerifyAddress(addr)
	case bitcoin.BitcoinChainId:
		return bitcoin.VerifyAddress(addr)
	case monero.MoneroChainId:
		return monero.VerifyAddress(addr)
	case zcash.ZcashChainId:
		return zcash.VerifyAddress(addr)
	case horizen.HorizenChainId:
		return horizen.VerifyAddress(addr)
	case litecoin.LitecoinChainId:
		return litecoin.VerifyAddress(addr)
	case dogecoin.DogecoinChainId:
		return dogecoin.VerifyAddress(addr)
	case ravencoin.RavencoinChainId:
		return ravencoin.VerifyAddress(addr)
	case namecoin.NamecoinChainId:
		return namecoin.VerifyAddress(addr)
	case dash.DashChainId:
		return dash.VerifyAddress(addr)
	case decred.DecredChainId:
		return decred.VerifyAddress(addr)
	case bch.BitcoinCashChainId:
		return bch.VerifyAddress(addr)
	case bsv.BitcoinSVChainId:
		return bsv.VerifyAddress(addr)
	case handshake.HandshakenChainId:
		return handshake.VerifyAddress(addr)
	case nervos.NervosChainId:
		return nervos.VerifyAddress(addr)
	case siacoin.SiacoinChainId:
		return siacoin.VerifyAddress(addr)
	case filecoin.FilecoinChainId:
		return filecoin.VerifyAddress(addr)
	case solana.SolanaChainId:
		return solana.VerifyAddress(addr)
	case near.NearChainId:
		return near.VerifyAddress(addr)
	case polkadot.PolkadotChainId:
		return polkadot.VerifyAddress(addr)
	case kusama.KusamaChainId:
		return kusama.VerifyAddress(addr)
	case ripple.RippleChainId:
		return ripple.VerifyAddress(addr)
	case stellar.StellarChainId:
		return stellar.VerifyAddress(addr)
	case tezos.TezosChainId:
		return tezos.VerifyAddress(addr)
	case eos.EOSChainId:
		return eos.VerifyAddress(addr)
	case tron.TronChainId:
		return tron.VerifyAddress(addr)
	case mobilecoin.MobileCoinChainId:
		return mobilecoin.VerifyAddress(addr)
	case cosmos.CosmosChainId:
		return cosmos.VerifyAddress(addr)
	case avalanche.AvalancheChainId:
		return avalanche.VerifyAddress(addr)
	case binance.BinanceChainId:
		return binance.VerifyAddress(addr)
	case akash.AkashChainId:
		return akash.VerifyAddress(addr)
	case arweave.ArweaveChainId:
		return arweave.VerifyAddress(addr)
	case dfinity.DfinityChainId:
		return dfinity.VerifyAddress(addr)
	case algorand.AlgorandChainId:
		return algorand.VerifyAddress(addr)
	case polygon.PolygonChainId:
		return polygon.VerifyAddress(addr)
	}
	panic(chainId)
}

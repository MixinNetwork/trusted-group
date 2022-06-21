# Bridge Proxy

This is a simple bridge service that is able to transfer all assets between MVM and different blockchains.


## Setup

At first, you need an Ethereum compatible wallet, e.g. MetaMask, then visit https://scan.mvm.dev to add Mixin Virtual Machine network to the wallet.

Get your account address, e.g. `0x914DFf811EF12267e1b644d9cb9B65743B98B131`, and register it to the bridge with following API

```
curl -H 'Content-Type: application/json' https://bridge.mvm.dev/users \
  -d '{"public_key": "0x914DFf811EF12267e1b644d9cb9B65743B98B131"}'
```


## Bridge Asset

After you registered your account address to the proxy, you got a Mixin Network API user, with which you can get all its deposit addresses for different blockchains.

Besides deposit money through those different blockchain addresses, another surprising thing is you can transfer in any assets from your Mixin Network wallets instantly.


## Access MTG

It's easy to make 4swap or other MTG apps compatible with the bridged Ethereum compatible wallets, e.g. MetaMask. Let's say swap some BTC to MOB, it's as easy as sign a `transferWithExtra` transaction to the `BTC` contract on MVM.

The first param for `transferWithExtra` is the contract address obtained from the setup phase, and the second one is the amount of BTC to swap, note that all the bridged assets has an 8 decimals precision.

The last param is the encoded data of MTG receivers and extra data, read the `encodeActionToExtra` code in action.go to get the details. And the proxy also provide an API `POST /extra` to make the test faster.

# Setup MVM Network for Eos Mainnet

## Install EOSIO software

EOSIO software can be obtained from `https://github.com/EOSIO/eos`

## Generate an Eos Private Key for signing MTG events

Each mvm node should generate an Eos private key with the following command

```
cleos create key --file key.txt
```

## Preparing two Eos Accounts

One for deploying mtg.xin contract, one for publishing MTG events.
In this example, it will use `mtgxinmtgxin` account to deploy mtg.xin contract, and `mtgpublisher` to publish MTG event. The `owner` permission and the `active` permission must be multi-signed with public keys created above.

## Setup an Eos node

1. Download an EOS Mannnet v5 snapshot from `https://snapshots.eosnation.io`
2. Extract snapshot file. Take `snapshot-2021-12-05-04-eos-v5-0219028238.bin.zst` as downloaded file for example.

```bash
zstd -d snapshot-2021-12-05-04-eos-v5-0219028238.bin.zst
```

3. Obtain an Eos node config file from `https://github.com/learnforpractice/mvm-eos-config` and copy it to `config-dir` directory

4. Start an Eos node with the snapshot 

```
nodeos --data-dir data-dir --config-dir config-dir --filter-on="mtgxinmtgxin:ontxlog:mtgxinmtgxin" --plugin eosio::history_api_plugin --snapshot snapshot-2021-12-05-04-eos-v5-0219028238.bin
```

For the next time you start the node, just run the following command:

```bash
nodeos --data-dir data-dir --config-dir config-dir --filter-on="mtgxinmtgxin:ontxlog:mtgxinmtgxin" --plugin eosio::history_api_plugin 
```

## Config Eos in MVM config file

```
[eos]
store = "./test/eos"
key = ""
public-keys = [
  'EOS6tmi4GpKzHqPYHE11vfacHtYfaz4rSX1x3efN4MPJcjZktCjwt',
  'EOS7Rf9DaUHqFHcCR5N2KVsGZRv1ApGamZNrSQNgQ8AY2rd5GRCGT',
  'EOS5UV91Dhe9p2AjgSnWbLxx5N9UMfnqGc9GPDW35MDTegobcgCQN',
  'EOS5vKpMYTByN6Mv28NZDQVnu29PPwrLGFCTcUs8oq6WsMmysLVtZ'
]

rpc-get-state = "http://127.0.0.1:8888"
chain-id = "aca376f206b8fc25a6ed44dbdc66547c36c6c33e3a119ffbeaef943642f0e906"
mixin-contract="mtgxinmtgxin"
mtg-publisher="mtgpublisher"

mtg-executor="mtgexecutor1"
mtg-executor-key=""

rpc-push = "http://127.0.0.1:8888"
publisher=true

```

Explanation:

1. `store` specifies the directory for storing eos engine data.
2. `key` specifies the key for signing events.
3. `public-keys` contains all the public keys of mtg.xin contract `active` permission which is multi-signed by MVM nodes.
4. `rpc-get-state` is the Eos Mainnet node url for getting on chain data
5. `chain-id` specifies the chain id of the Eos mainnet.
6. `mixin-contract` specifies the account which runs the contract at `contracts/mtg.xin`.
7. `mtg-publisher` specifies the account to publish mtg events.
8. `mtg-executor` specifies the account to execute mtg events.
9. `mtg-executor-key` specifies the `mtg-executor` private key for signing event transactions.
10. `rpc-push` specifies the mainnet RPC url to broadcast event transactions
11. `publisher` indicates whether the MVM node should broadcast signed MTG event transactions to Eos network. It's ok to specify multiple publisher in a MVM network.

## Deploying mtg.xin Contract

mtg.xin contract should be deployed to `mtgxinmtgxin` account. mvm nodes can use `eosio.msig` for deploying mtg.xin contract.

## Publishing an Eos Smart Contract to MVM
1. First deploy your Eos MVM contract. An example can be found in `contracts/dappdemo` directory.
2. Publish contract to MVM network with the following command:

```
./mvm publish -p eos -m mvmconfig.toml -k bot.json -a mvmtest12345 -e helloworld
```

Replace `mvmtest12345` with your contract account

1. `mvmconfig.toml` is a MVM config file that you can obtain from a MVM node. The content is like below:

```
[mtg.genesis]
members = [
  "e07c06fa-084c-4ce1-b14a-66a9cb147b9e",
  "e0148fc6-0e10-470e-8127-166e0829c839",
  "18a62033-8845-455f-bcde-0e205ef4da44",
  "49b00892-6954-4826-aaec-371ca165558a"
]
threshold = 3
timestamp = 1639648035217658880
```

2. `bot.json` is the mixin bot configure file that was fetched from the mixin developers dashboard.
3. `-e` specifies the memo
4. `-a` specifies the account which Smart Contract deployed on.

## Interacting with the Smart Contract in the MVM network

```
./mvm invoke -m mvmconfig.toml -k bot.json -p 49b00892-6954-4826-aaec-371ca165558a -a 0.145 -e helloworld
```

1. `-p` specifies the mixin bot client id which has been bound to an Eos account.
2. `-e `specifies the memo.


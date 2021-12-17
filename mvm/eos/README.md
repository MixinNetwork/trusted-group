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
zstd -d snapshot-2021-12-05-04-eos-v5-0219028238.bin.zst snapshot-2021-12-05-04-eos-v5-0219028238.bin
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
store = "./test/eos1"
rpc = "http://127.0.0.1:9000"
key = "5JpXLb1tqxJB3Xtzd584xTdqKAzBnQ4TkqfEtT5QPotuv7Yt2bX"
public_keys = [
   "EOS7rVXEPKJsYbjW3KZpaLBnpTx7U3XTdKedEWfPpj2mF9jTdcsG5",
   "EOS8JEHuDKeLgKfhCQT1rwHSdB1fuDd2ngZDequcvWbF9a92VKubc",
   "EOS6Bi6Ndo1PxiEuvt1osP267xyjo8HEbLgVzBrX7Ewruf7dbUiRp",
   "EOS8N2voiByjmCZeTwQpjHjvwpg8Gnbv3wFTqgcg7GdKk1UvYekzX"
]

chain_id = "aca376f206b8fc25a6ed44dbdc66547c36c6c33e3a119ffbeaef943642f0e906"
mixin_contract="mtgxinmtgxin"
mtg_publisher="mtgpublisher"
publisher=true
```

Explanation:

1. `store` specifies the directory for storing eos engine data
2. `rpc` is the Eos Mainnet node url
3. `public_keys` contains all the public keys of mtg.xin contract `active` permission which is multi-signed by MVM nodes.
4. `chain_id` specifies the chain id of the Eos mainnet.
5. `mixin_contract` specifies the account which will run the contract at `contracts/mtg.xin`.
6. `mtg_publisher` specifies the account for publishing mtg events.
7. `publisher` indicates whether the MVM node should broadcast signed MTG event transactions to Eos network. It's ok to specify multiple publisher in a MVM network.

## Deploying mtg.xin Contract

mtg.xin contract should be deployed to `mtgxinmtgxin` account. mvm nodes can use `eosio.msig` for deploying mtg.xin contract.

## Publishing an Eos Smart Contract to MVM
1. First deploy your Eos MVM contract. An example can be found in `contracts/dappdemo` directory.
2. Publish contract to MVM network with the following command:

```
./mvm publish -p eos -m mvmconfig.toml -k bot.json -a mvmtest12345 -e 00000016434f895e3a242657
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
3. `-e` specifies the first 24 hex characters of referenced block id which was used to build an Eos transaction. The referenced block id can be fetched from get_info RPC API
4. `-a` specifies the account which Smart Contract deployed on.

## Interacting with the Smart Contract in the MVM network

```
./mvm invoke -m mvmconfig.toml -k bot.json -p 49b00892-6954-4826-aaec-371ca165558a -a 0.145 -e 00000016434f895e3a242657
```

1. `-p` specifies the mixin bot client id which has been bound to an Eos account.

2. Just like publish command, `-e `specifies the first 24 hex characters of referenced block id.

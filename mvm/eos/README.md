# Setup a Local MVM Testnet for Eos

## Installing nodeos from `https://github.com/EOSIO/eos`

## Installing pyeoskit for Launching a Testnet

```
pip3 install pyeoskit
```

## Config Eos in MVM config file

Below is an Example:

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

chain_id = "8a34ec7df1b8cd06ff4a8abbaa7cc50300823350cadc59ab296cb00d104d2b8f"
mixin_contract="mtgxinmtgxin"
publisher=true
```

Explanation:

1. `store` specify the directory for store eos engine data
2. `rpc` is the Eos testnet node url
3. `public_keys` contains all the public keys of mixin contract action permission which is multisigned by MVM nodes.
4. `chain_id` specify the chain id of Eos network or Eos testnet.
5. `mixin_contract` specify the mixin account which will run contract at `contracts/mtg.xin`.
5. `publisher` specify the Eos transaction publisher


## Starting an Eos testnet

```python
from pyeoskit import testnet
test = testnet.Testnet(extra='--filter-on="mtgxinmtgxin:ontxlog:mtgxinmtgxin" --plugin=eosio::db_size_api_plugin', show_log=True)
```

Pay attention to `--filter-on="mtgxinmtgxin:ontxlog:mtgxinmtgxin"`, this filter is  critical to make Eos engine of MVM works. Currently Eos engine leverage history-plugin to fetch action trace history. It's practical since the size of each record of action history pretty is small. Eos engine will add support for state-history plugin which require heavy Server resources.

## Deploying mtg.xin Contract to Testnet

For deploying mtg.xin contract, just run the following command

```
cd contracts
./deploy.sh
``` 

Alongside deploying mtg.xin contract to mtgxinmtgxin account, deploy.sh will also deploy dappdemo contract to `helloworld11` account

## Publishing Eos Smart Contracts to MVM
1. First deploy your Eos contract for MVM. For an example, please refer to `contracts/dappdemo` directory.
2. Publish contract to MVM network with the following command:

```
./mvm publish -p eos -m test1.toml -k bot.json -a helloworld11 -e 00000016434f895e3a242657
```

1. `test1.toml` is the MVM config file
2. `bot.json` is the mixin bot configure file which was fetched from mixin developers dashboard.
3. `-e` specify the first 24 hex charactors of referenced block id which used to build an Eos transaction. The referenced block id can be fetched from get_info rpc api
4. `-a` specify the account which Smart Contract deployed on.

There's a `publish.py` script file in `scripts` directory which can be modified accordingly for easing the work.

## Interacting with Smart Contract with MVM network

```
./mvm invoke -m test1.toml -k test4.json -p 49b00892-6954-4826-aaec-371ca165558a -a 0.145 -e 00000016434f895e3a242657
```

1. `-p` specify the mixin bot client id which has been binded to an Eos account.

2. Just like publish command, `-e `specify the first 24 hex charactors of referenced block id.

There's a `invoke.py` script file in `scripts` directory which can be modified accordingly for easing the work.

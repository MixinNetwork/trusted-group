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


## Launching an Eos testnet

Launch python interactive console with `python3 -i` command.

Import the following scripts:
```python
from pyeoskit import testnet
test = testnet.Testnet(host='127.0.0.1', extra='--filter-on="mtgxinmtgxin:ontxlog:mtgxinmtgxin" --plugin=eosio::db_size_api_plugin', show_log=False)
test.start()
```

That will print the nodeos command like this, save it somewhere, so next time you can start the testnet without python. 

```
nodeos --verbose-http-errors  --http-max-response-time-ms 100 --p2p-listen-endpoint 127.0.0.1:9100 --data-dir ./.eos-testnet/dd --config-dir ./.eos-testnet/cd --filter-on="mtgxinmtgxin:ontxlog:mtgxinmtgxin" --plugin=eosio::db_size_api_plugin -p eosio --plugin eosio::producer_plugin --plugin eosio::chain_api_plugin --plugin eosio::producer_api_plugin --plugin eosio::history_api_plugin -e --resource-monitor-space-threshold 99 --http-server-address 127.0.0.1:9000 --contracts-console --access-control-allow-origin="*"
```

You may need to change `host` to a public address for other nodes to connect to this node.

To stop Eos testnet, run:
```
test.stop()
```

Pay attention to `--filter-on="mtgxinmtgxin:ontxlog:mtgxinmtgxin"`, this filter is  critical to make Eos engine of MVM works. Currently Eos engine leverage history-plugin to fetch action trace history. It's practical since the size of each record of action history is pretty small. Eos engine will add support for `state-history-plugin` which requires heavy server resources.


## Connecting to a Testnet
```
nodeos --verbose-http-errors  --http-max-response-time-ms 100 --p2p-listen-endpoint 127.0.0.1:9101 --p2p-peer-address 127.0.0.1:9100 --data-dir ./.eos-testnet/dd --config-dir ./.eos-testnet/cd --filter-on="mtgxinmtgxin:ontxlog:mtgxinmtgxin" --plugin=eosio::db_size_api_plugin --plugin eosio::producer_plugin --plugin eosio::chain_api_plugin --plugin eosio::producer_api_plugin --plugin eosio::history_api_plugin --resource-monitor-space-threshold 99 --http-server-address 127.0.0.1:9001 --contracts-console --access-control-allow-origin="*"
```

The following arguments need to modify accordingly.
1. `--p2p-listen-endpoint` specify address for listen to incomming p2p connections. 
2. `--p2p-peer-address` specify p2p address for connecting to, in this example, it's `127.0.0.1:9100` which is specified in launch Eos testnet command
3. `--http-server-address` specify rpc address for Eos engine to connecting to.


## Deploying mtg.xin Contract to Testnet

For deploying mtg.xin contract, just run the following command

```
cd contracts
./deploy.sh
``` 

Alongside deploying mtg.xin contract to `mtgxinmtgxin` account, `deploy.sh` will also deploy dappdemo contract to `helloworld11` account

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

## Interacting with Smart Contract in MVM network

```
./mvm invoke -m test1.toml -k test4.json -p 49b00892-6954-4826-aaec-371ca165558a -a 0.145 -e 00000016434f895e3a242657
```

1. `-p` specify the mixin bot client id which has been binded to an Eos account.

2. Just like publish command, `-e `specify the first 24 hex charactors of referenced block id.

There's a `invoke.py` script file in `scripts` directory which can be modified accordingly for easing the work.

import os
import sys
import json
import time
from inspect import currentframe, getframeinfo

test_dir = os.path.dirname(__file__)
sys.path.append(os.path.join(test_dir, '..'))

from uuosio import log
from uuosio.chaintester import ChainTester

logger = log.get_logger(__name__)

tester = ChainTester()

main_account = 'mtgxinmtgxin'
token_account = 'mtgtoken1234'

owner_key = 'EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV'
active_key = 'EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV'
tester.create_account('eosio', 'mvm.token', owner_key, active_key, 64*1024, 1.0, 1.0)
tester.create_account('eosio', main_account, owner_key, active_key, 1024*1024, 1.0, 1.0)
tester.create_account('eosio', token_account, owner_key, active_key, 1024*1024, 1.0, 1.0)

args = {
    "account": main_account,
    "permission": "active",
    "parent": "owner",
    "auth": {
        "threshold": 1,
        "keys": [
            {
                "key": "EOS6AjF6hvF7GSuSd4sCgfPKq5uWaXvGM2aQtEUCwmEHygQaqxBSV",
                "weight": 1
            },
        ],
        "accounts": [
            {
                "permission":
                {
                    "actor":main_account,
                    "permission": "eosio.code"
                },
                "weight":1
            }
        ],
        "waits": []
    }
}

tester.push_action('eosio', 'updateauth', args, {main_account:'active'})

def init():
    with open('dappdemo.wasm', 'rb') as f:
        code = f.read()
    with open('dappdemo.abi', 'rb') as f:
        abi = f.read()
    tester.deploy_contract(main_account, code, abi, 0)

init()

def test_mtg_tx():
    event = {
        'nonce': 1,
        'process': '0x' + '11' * 16,
        'asset': '0x' + '22' * 16,
        'members': ['aa', 'bb'],
        'threshold': 1,
        'amount': '0x' + '33' * 16,
        'extra': '00' * 32,
        'timestamp': int(time.time()),
        'signature': '00' * 64
    }
    tx_event = {
        'event': event
    }

    r = tester.push_action(main_account, 'onevent', tx_event, {main_account: 'active'})
    tester.produce_block()
    r = tester.get_table_rows(True, main_account, main_account, 'txevents', '', '', 10)
    logger.info(r)

    # assert len(r['rows']) == 1
    r = tester.get_table_rows(False, main_account, main_account, 'txrequests', '', '', 10)
    logger.info(r)

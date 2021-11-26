
import os
import sys
import time
try:
	from pyeoskit import eosapi, wallet
except:
	print('pyeoskit not found, please install it with "pip install pyeoskit"')
	sys.exit(-1)

from pyeoskit import log
from pyeoskit.exceptions import ChainException
logger = log.get_logger(__name__)

# modify your test account here
main_account = 'helloworld11'
# modify your test account private key here
wallet.import_key('test', '5JRYimgLBrRLCBAcjHUWCYRv3asNedTYYzVgmiU4q2ZVxMBiJXL')
wallet.import_key('test', '5Jbb4wuwz8MAzTB9FJNmrVYGXo4ABb7wqPVoWGcZ6x8V2FwNeDo')

# modify test node here
eosapi.set_node('http://127.0.0.1:9000')

info = eosapi.get_account(main_account)
# logger.info(info)

owner_key = 'EOS7sPDxfw5yx5SZgQcVb57zS1XeSWLNpQKhaGjjy2qe61BrAQ49o'
active_key = 'EOS7sPDxfw5yx5SZgQcVb57zS1XeSWLNpQKhaGjjy2qe61BrAQ49o'
try:
    eosapi.create_account(main_account, 'mtgxinmtgxin', owner_key, active_key, 1024*1024, 1.0, 10000.0)
except Exception as e:
    pass
    # logger.error(e)
pub_key = 'EOS7sPDxfw5yx5SZgQcVb57zS1XeSWLNpQKhaGjjy2qe61BrAQ49o'
# pub_key = 'EOS6AjF6hvF7GSuSd4sCgfPKq5uWaXvGM2aQtEUCwmEHygQaqxBSV'

# from pyeoskit import utils
# utils.dbw('helloworld11', 'mtgxinmtgxin', 10.0, 1000.0)

def deploy_contract(account, path, pub_key):
    args = {
        "account": account,
        "permission": "active",
        "parent": "owner",
        "auth": {
            "threshold": 1,
            "keys": [
                {
                    "key": pub_key,
                    "weight": 1
                },
            ],
            "accounts": [
                {
                    "permission":
                    {
                        "actor":account,
                        "permission": "eosio.code"
                    },
                    "weight":1
                }
            ],
            "waits": []
        }
    }

    eosapi.push_action('eosio', 'updateauth', args, {account:'active'})

    with open(f'{path}.wasm', 'rb') as f:
        code = f.read()
    with open(f'{path}.abi', 'rb') as f:
        abi = f.read()
    if account == 'mtgxinmtgxin':
        abi = b''

    try:
        eosapi.deploy_contract(account, code, abi, vm_type=0)
    except ChainException as e:
        if not e.json['error']['details'][0]['message'] == 'contract is already running this version of code':
            raise e

deploy_contract('mtgxinmtgxin', './mtg.xin/mtg.xin', pub_key)
deploy_contract('helloworld11', './dappdemo/dappdemo', pub_key)


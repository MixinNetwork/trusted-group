
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
main_account = 'mtgxinmtgxin'
# modify your test account private key here
wallet.import_key('test', '5JRYimgLBrRLCBAcjHUWCYRv3asNedTYYzVgmiU4q2ZVxMBiJXL')
wallet.import_key('test', '5Jbb4wuwz8MAzTB9FJNmrVYGXo4ABb7wqPVoWGcZ6x8V2FwNeDo')

# modify test node here
eosapi.set_node('http://127.0.0.1:9000')

info = eosapi.get_account(main_account)
# logger.info(info)

owner_key = 'EOS6AjF6hvF7GSuSd4sCgfPKq5uWaXvGM2aQtEUCwmEHygQaqxBSV'
active_key = 'EOS6AjF6hvF7GSuSd4sCgfPKq5uWaXvGM2aQtEUCwmEHygQaqxBSV'
try:
    eosapi.create_account(main_account, 'mtgxinmtgxin', owner_key, active_key, 1024*1024, 1.0, 10.0)
except Exception as e:
    pass
    # logger.error(e)
pub_key = 'EOS7sPDxfw5yx5SZgQcVb57zS1XeSWLNpQKhaGjjy2qe61BrAQ49o'
# pub_key = 'EOS6AjF6hvF7GSuSd4sCgfPKq5uWaXvGM2aQtEUCwmEHygQaqxBSV'
args = {
    "account": main_account,
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
                    "actor":main_account,
                    "permission": "eosio.code"
                },
                "weight":1
            }
        ],
        "waits": []
    }
}

eosapi.push_action('eosio', 'updateauth', args, {main_account:'active'})


with open('test.wasm', 'rb') as f:
    code = f.read()
with open('test.abi', 'rb') as f:
    abi = f.read()

try:
    eosapi.deploy_contract(main_account, code, abi, vm_type=0)
except ChainException as e:
    if not e.json['error']['details'][0]['message'] == 'contract is already running this version of code':
        raise e

r = eosapi.push_action(main_account, 'sayhello', b'', {main_account:'active'})
logger.info(r)

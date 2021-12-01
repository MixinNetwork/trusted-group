
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
logger.info(info)

with open('dappdemo.wasm', 'rb') as f:
    code = f.read()
with open('dappdemo.abi', 'rb') as f:
    abi = f.read()

try:
    eosapi.deploy_contract(main_account, code, abi, vm_type=0)
except ChainException as e:
    if not e.json['error']['details'][0]['message'] == 'contract is already running this version of code':
        raise e

event = {
    'nonce': 2,
    'process': '0x' + '11' * 16,
    'asset': '0x' + '22' * 16,
    'members': ['aa', 'bb'],
    'threshold': 1,
    'amount': '0x' + int.to_bytes(int(1e9), 16, 'little').hex(),
    'extra': '00' * 32,
    'timestamp': int(time.time()),
    'signature': '00' * 64
}

tx_event = {
    'event': event
}

try:
    r = eosapi.push_action(main_account, 'onevent', tx_event, {main_account: 'active'})
except Exception as e:
    logger.info(e)

r = eosapi.get_table_rows(True, main_account, main_account, 'txevents', '', '', 10)
logger.info(r)

r = eosapi.get_table_rows(True, main_account, main_account, 'txrequests', '', '', 10)
logger.info(r)


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

with open('dappdemo.wasm', 'rb') as f:
    code = f.read()
with open('dappdemo.abi', 'rb') as f:
    abi = f.read()

try:
    eosapi.deploy_contract(main_account, code, abi, vm_type=0)
except ChainException as e:
    if not e.json['error']['details'][0]['message'] == 'contract is already running this version of code':
        raise e

r = eosapi.get_table_rows(True, main_account, main_account, 'txrequests', '', '', 10)
logger.info(r)
rows = r['rows']
if not rows:
    sys.exit(0)

args = {
    'lastFinishedRequest': rows[-1]['id']
}

logger.info(args)

try:
    r = eosapi.push_action(main_account, 'clearreqs', args, {main_account: 'active'})
    logger.info(r['processed']['action_traces'][0]['console'])
except Exception as e:
    logger.info(e)


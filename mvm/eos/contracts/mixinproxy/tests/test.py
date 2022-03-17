import os
import sys
import json
import time
import uuid
import hashlib
from inspect import currentframe, getframeinfo

test_dir = os.path.dirname(__file__)
sys.path.append(os.path.join(test_dir, '..'))

from ipyeos import log
from ipyeos.chaintester import ChainTester

from pyeoskit import eosapi

logger = log.get_logger(__name__)

MTG_XIN_CONTRACT = 'mtgxinmtgxin'
MTG_PUBLISHER = 'mtgpublisher'

def get_line_number():
    cf = currentframe()
    return cf.f_back.f_lineno

def print_except(tx):
    if 'processed' in tx:
        tx = tx['processed']
    for trace in tx['action_traces']:
        logger.info(trace['console'])
        logger.info(json.dumps(trace['except'], indent=4))

def uuid2uint128(uuid_str):
    process = uuid.UUID(uuid_str)
    process = int.from_bytes(process.bytes, 'little')
    return '0x' + process.to_bytes(16, 'big').hex()

def print_console(tx):
    cf = currentframe()
    filename = getframeinfo(cf).filename
    num = cf.f_back.f_lineno

    if 'processed' in tx:
        tx = tx['processed']

    for trace in tx['action_traces']:
        # logger.info(trace['console'])
        print(f'{num}:action_traces:%s'%(trace['console'], ))
        if not 'inline_traces' in trace:
            continue
        for inline_trace in trace['inline_traces']:
            print(f'{num}:inline_traces:%s'%(inline_trace['console'], ))


class Test(object):
    @classmethod
    def setup_class(cls):
        cls.init()

    @classmethod
    def teardown_class(cls):
        cls.chain.free()

    def setup_method(self, method):
        pass

    def teardown_method(self, method):
        self.chain.produce_block()

    @classmethod
    def init(cls):
        cls.test_keys = []
        for i in range(4):
            key = eosapi.create_key()
            cls.test_keys.append(key)

        cls.chain = ChainTester()
        owner_key = 'EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV'
        active_key = 'EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV'
        accounts = [
            MTG_XIN_CONTRACT,
            'mtgpublisher',
            'mtgsigner111',
            'mtgsigner112',
            'mtgsigner113',
            'mtgsigner114',
            'mixincrossss',
            'mixinwtokens',
        ]

        for account in accounts:
            cls.chain.create_account('eosio', account, owner_key, active_key, 10*1024*1024, 10.0, 10.0)
        cls.chain.produce_block()

        cls.chain.transfer('hello', 'mixincrossss', 1000.0000, 'hello')


        cls.update_auth(MTG_XIN_CONTRACT, 'EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV')
        cls.update_auth('mixincrossss', 'EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV')
        cls.update_auth('mixinwtokens', 'EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV', 'mixincrossss')

        pub_key = 'EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV'
        account = 'mixincrossss'
        args = {
            "account": account,
            "permission": "multisig",
            "parent": "owner",
            "auth": {
                "threshold": 1,
                "keys": [
                    {
                        "key": pub_key,
                        "weight": 1
                    },
                ],
                "accounts": [],
                "waits": []
            }
        }

        cls.chain.push_action('eosio', 'updateauth', args, {account:'owner'})

        with open(os.path.join(test_dir, 'mtg.xin.wasm'), 'rb') as f:
            code = f.read()
        with open(os.path.join(test_dir, 'mtg.xin.abi'), 'r') as f:
            abi = f.read()
        cls.chain.deploy_contract(MTG_XIN_CONTRACT, code, abi, 0)

        with open('mixinproxy.wasm', 'rb') as f:
            code = f.read()
        with open('mixinproxy.abi', 'r') as f:
            abi = f.read()
        cls.chain.deploy_contract('mixincrossss', code, abi, 0)

        with open(os.path.join(test_dir, 'token.wasm'), 'rb') as f:
            code = f.read()
        with open(os.path.join(test_dir, 'token.abi'), 'r') as f:
            abi = f.read()
        cls.chain.deploy_contract('mixinwtokens', code, abi, 0)
        
        signers = [
                'mtgsigner111',
                'mtgsigner112',
                'mtgsigner113',
                'mtgsigner114',
        ]
        _signers = []
        for i in range(len(signers)):
            signer = {
                'account': signers[i],
                'public_key': cls.test_keys[i]['public'],
            }
            _signers.append(signer)

        args = dict(
            signers = _signers
        )
        r = cls.chain.push_action(MTG_XIN_CONTRACT, 'setup', args, {MTG_XIN_CONTRACT: 'active'})
        print_console(r)
        # rows = cls.chain.get_table_rows(True, MTG_XIN_CONTRACT, MTG_XIN_CONTRACT, 'signers', '', '', 10)
        # logger.info(rows)
        client_id = 'e0148fc6-0e10-470e-8127-166e0829c839'
        process = uuid2uint128(client_id)
        args = {
            'contract': 'mixincrossss',
            'process': process,
            'signatures': [],
        }

        packed_add_process = cls.chain.pack_args(MTG_XIN_CONTRACT, 'addprocess', args)
        packed_add_process = packed_add_process[:-1]
        digest = hashlib.sha256(packed_add_process).hexdigest()
        signatures = []
        for key in cls.test_keys:
            priv = key['private']
            signature = eosapi.sign_digest(digest, priv)
            signatures.append(signature)
        args['signatures'] = signatures

        r = cls.chain.push_action(MTG_XIN_CONTRACT, 'addprocess', args, {MTG_XIN_CONTRACT: 'active'})

        r = cls.chain.push_action('mixincrossss', 'initialize', b'', {'mixincrossss': 'active'})
        
        asset_id = uuid2uint128('43d61dcd-e413-450d-80b8-101d5e903357')
        args = {
            'symbol': '8,METH',
            'asset_id': asset_id, #ETH
        }
        r = cls.chain.push_action('mixincrossss', 'addasset', args, {'mixincrossss': 'active'})

        asset_id = uuid2uint128('6cfe566e-4aad-470b-8c9a-2fd35b49c68d')
        args = {
            'symbol': '8,MEOS',
            'asset_id': asset_id, #EOS
        }
        r = cls.chain.push_action('mixincrossss', 'addasset', args, {'mixincrossss': 'active'})

        args = {
            'fee': '0.10000000 MEOS',
        }
        r = cls.chain.push_action('mixincrossss', 'setaccfee', args, {'mixincrossss': 'active'})

    @classmethod
    def update_auth(cls, account, pub_key, code_account = None):
        if not code_account:
            code_account = account
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
                            "actor": code_account,
                            "permission": "eosio.code"
                        },
                        "weight":1
                    }
                ],
                "waits": []
            }
        }

        cls.chain.push_action('eosio', 'updateauth', args, {account:'active'})

    def get_balance(self, account):
        params = dict(
            json=True,
            code='mixinwtokens',
            scope=account,
            table='accounts',
            lower_bound='',
            upper_bound='',
            limit=10,
        )
        try:
            ret = self.chain.api.get_table_rows(params)
            balance = ret['rows'][0]['balance'].split(' ')[0]
            return round(float(balance) * 10000) / 10000
        except Exception as e:
            logger.info(e)
            return 0.0

    def sign_event(self, tx_event):
        packed_tx_event = self.chain.pack_args('mixincrossss', 'onevent', tx_event)
        logger.info("+++packed_tx_event: %s", packed_tx_event.hex())

        packed_tx_event = packed_tx_event[:-1]
        digest = hashlib.sha256(packed_tx_event).hexdigest()
        signatures = []
        for key in self.test_keys:
            priv = key['private']
            signature = eosapi.sign_digest(digest, priv)
            signatures.append(signature)
        tx_event['event']['signatures'] = signatures

    def test_event(self):
        client_id = 'e0148fc6-0e10-470e-8127-166e0829c839'
        asset_id = uuid2uint128('6cfe566e-4aad-470b-8c9a-2fd35b49c68d')

        process = uuid2uint128(client_id)
        logger.info("++++++process %s", process)
        event = {
            'nonce': 1,
            'process': process,
            'asset': asset_id, #EOS
            'members': ['0x' + '11' * 16],
            'threshold': 1,
            'amount': '0x' + int.to_bytes(int(1e8), 16, 'big').hex(),
            'extra': b'hello'.hex(),
            'timestamp': int(time.time()*1e9),
            'signatures': []
        }

        tx_event = {
            'event': event
        }
        self.sign_event(tx_event)
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        print_console(r)
        logger.info('++++%s', r['elapsed'])
        self.chain.produce_block()

        args = {
            'from': 'aaaaaaaaamvm',
            'to': 'eosio',
            'quantity': '0.10000000 MEOS',
            'memo': 'hello'
        }
        self.chain.pack_args('mixinwtokens', 'transfer', args)
        extra = ''
        event = {
            'nonce': 2,
            'process': process,
            'asset': asset_id, #EOS
            'members': ['0x' + '11' * 16],
            'threshold': 1,
            'amount': '0x' + int.to_bytes(int(1e8), 16, 'big').hex(),
            'extra': extra,
            'timestamp': int(time.time()*1e9),
            'signatures': []
        }

        tx_event = {
            'event': event
        }
        self.sign_event(tx_event)
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})

        ret = self.chain.get_table_rows(True, 'mixincrossss', 'mixincrossss', 'createaccfee', '','', 10)
        logger.info("+++++ret %s", ret)
        logger.info("+++++balance %s", self.get_balance('aaaaaaaaamvm'))

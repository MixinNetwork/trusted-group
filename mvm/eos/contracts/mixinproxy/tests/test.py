import os
import io
import sys
import json
import time
import uuid
import hashlib
from datetime import datetime, timedelta
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

def uuid2bytes(uuid_str):
    return uuid.UUID(uuid_str).bytes

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
        pass

    @classmethod
    def teardown_class(cls):
        pass

    def setup_method(self, method):
        self.init()

    def teardown_method(self, method):
        self.chain.free()

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
        cls.update_auth('mixinwtokens', 'EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV', ['mixincrossss', 'mixinwtokens'])

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
        process_id_str = 'e0148fc6-0e10-470e-8127-166e0829c839'
        process = uuid2uint128(process_id_str)
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
    def update_auth(cls, account, pub_key, code_accounts = None):
        if not code_accounts:
            code_accounts = [account]
        account_permissions = []
        for account in code_accounts:
            perm = {
                "permission":
                {
                    "actor": account,
                    "permission": "eosio.code"
                },
                "weight":1
            }
            account_permissions.append(perm)

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
                "accounts": account_permissions,
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
            if len(ret['rows']) == 0:
                return 0.0
            balance = ret['rows'][0]['balance'].split(' ')[0]
            return round(float(balance) * 10000) / 10000
        except Exception as e:
            logger.info(e)
            return 0.0

    def sign_event(self, tx_event):
        origin_extra = tx_event['origin_extra']
        tx_event['origin_extra'] = ''
        packed_tx_event = self.chain.pack_args('mixincrossss', 'onevent', tx_event)
        packed_tx_event = packed_tx_event[:-2]
        tx_event['origin_extra'] = origin_extra
        digest = hashlib.sha256(packed_tx_event).hexdigest()
        signatures = []
        for key in self.test_keys:
            priv = key['private']
            signature = eosapi.sign_digest(digest, priv)
            signatures.append(signature)
        tx_event['event']['signatures'] = signatures

    def build_extra(self, process, address = 'mixincrossss', extra=b''):
        def write_bytes(buf, data):
            buf.write(int.to_bytes(len(data), 2, 'big'))
            buf.write(data)

        if isinstance(extra, str):
            extra = extra.encode()
        print(process)
        process = uuid.UUID(process).bytes #test2 mixincross
        buf = io.BytesIO()
        buf.write(int.to_bytes(1, 2, 'big')) #Purpose

        buf.write(process) #Process

        write_bytes(buf, b'eos') #Platform
        write_bytes(buf, address.encode()) #Address
        write_bytes(buf, extra) #Extra
        value = buf.getvalue()
        assert len(value) < 256
        return value

    def build_event(self, asset_id, nonce, amount, extra, origin_extra, timestamp): 
        process_id_str = 'e0148fc6-0e10-470e-8127-166e0829c839'
        process = uuid2uint128(process_id_str)
        asset_id = uuid2uint128(asset_id)
        event = {
            'nonce': nonce,
            'process': process,
            'asset': asset_id, #EOS
            'members': ['0x' + '11' * 16],
            'threshold': 1,
            'amount': '0x' + int.to_bytes(int(amount), 16, 'big').hex(),
            'extra': extra.hex(),
            'timestamp': timestamp,
            'signatures': []
        }
        tx_event = {
            'event': event,
            'origin_extra': origin_extra.hex()
        }
        self.sign_event(tx_event)
        return tx_event

    def current_time(self):
        info = self.chain.api.get_info()
        head_block_time = info['head_block_time']
        return datetime.strptime(head_block_time, "%Y-%m-%dT%H:%M:%S.%f")

    def current_timestamp(self):
        info = self.chain.api.get_info()
        head_block_time = info['head_block_time']
        head_block_time = datetime.strptime(head_block_time, "%Y-%m-%dT%H:%M:%S.%f")

        delta = head_block_time - datetime(1970, 1, 1)
        return int(delta.total_seconds() * 1e9)

    def test_event(self):
        process_id_str = 'e0148fc6-0e10-470e-8127-166e0829c839'
        asset_id = '6cfe566e-4aad-470b-8c9a-2fd35b49c68d'
        tx_event = self.build_event(asset_id, 1, 1e8, b'hello', b'', int(time.time()*1e9))
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        print_console(r)
        logger.info('++++%s', r['elapsed'])
        self.chain.produce_block()

        tx_event = self.build_event(asset_id, 2, 1e8, b'', b'', int(time.time()*1e9))
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})

        ret = self.chain.get_table_rows(True, 'mixincrossss', 'mixincrossss', 'createaccfee', '','', 10)
        logger.info("+++++ret %s", ret)
        assert self.get_balance('aaaaaaaaamvm') == 1.0

        # def build_extra(process, amount, address = 'mixincrossss', extra=b''):
        args = {
            'from': 'aaaaaaaaamvm',
            'to': 'eosio',
            'quantity': '0.10000000 MEOS',
            'memo': 'hello'
        }
        args = self.chain.pack_args('mixinwtokens', 'transfer', args)
        extra = b'\x00' + int.to_bytes(self.chain.s2n('mixinwtokens'), 8, 'little') + \
            int.to_bytes(self.chain.s2n('transfer'), 8, 'little') + args
        originExtra = self.build_extra(process_id_str, 'mixincrossss', extra)
        hash = hashlib.sha256(originExtra).digest()
        extra = b'\x01' + hash + 'https://a.com'.encode()
        tx_event = self.build_event(asset_id, 3, 1e8, extra, b'', int(time.time()*1e9))

        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        args = {
            'executor': MTG_PUBLISHER, 
            'nonce': 3,
            'origin_extra': originExtra.hex()
        }
        r = self.chain.push_action('mixincrossss', 'execpending', args, {MTG_PUBLISHER: 'active'})
        self.chain.produce_block()
        assert self.get_balance('aaaaaaaaamvm') == 1.9

        delay_seconds = 3*60
        start_time = self.current_time()

        # next_block_time = start_time + timedelta(seconds=delay_seconds)
        # self.chain.produce_block(next_block_time)
        # self.chain.produce_block() #finalize the previous block

        delta = start_time - datetime(1970, 1, 1)
        timestamp = int((delta.total_seconds() - delay_seconds) * 1e9)
        # timestamp = int(time.time()*1e9)
        tx_event = self.build_event(asset_id, 4, 1e8, originExtra, b'', timestamp)
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        self.chain.produce_block()
        assert self.get_balance('aaaaaaaaamvm') == 1.9

    def test_expiration(self):
        process_id_str = 'e0148fc6-0e10-470e-8127-166e0829c839'
        asset_id = '6cfe566e-4aad-470b-8c9a-2fd35b49c68d'

        tx_event = self.build_event(asset_id, 1, 1e8, b'hello', b'', int(time.time()*1e9))
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
        args = self.chain.pack_args('mixinwtokens', 'transfer', args)
        extra = b'\x00' + int.to_bytes(self.chain.s2n('mixinwtokens'), 8, 'little') + \
            int.to_bytes(self.chain.s2n('transfer'), 8, 'little') + args
        # extra = self.build_extra(process_id_str, 'mixincrossss', extra)
        tx_event = self.build_event(asset_id, 2, 1e8, extra, b'', self.current_timestamp())
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        assert self.get_balance('aaaaaaaaamvm') == 0.9
        self.chain.produce_block()

        tx_event = self.build_event(asset_id, 3, 1e8, extra, b'', self.current_timestamp() - int(3*60*1e9))
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        assert self.get_balance('aaaaaaaaamvm') == 0.9
        self.chain.produce_block()


        tx_event = self.build_event(asset_id, 4, 1e8, extra, b'', self.current_timestamp() - int(3*60*1e9))
        tx_event['reason'] = 'test'
        r = self.chain.push_action('mixincrossss', 'onerrorevent', tx_event, {MTG_PUBLISHER: 'active'})
        assert self.get_balance('aaaaaaaaamvm') == 0.9
        self.chain.produce_block()

        ret = self.chain.get_table_rows(True, 'mixincrossss', 'mixincrossss', 'errorevents', '', '', True)
        assert len(ret['rows']) == 0

        tx_event = self.build_event(asset_id, 5, 1e8, extra, b'', self.current_timestamp())
        tx_event['reason'] = 'test'
        r = self.chain.push_action('mixincrossss', 'onerrorevent', tx_event, {MTG_PUBLISHER: 'active'})
        assert self.get_balance('aaaaaaaaamvm') == 0.9
        self.chain.produce_block()

        ret = self.chain.get_table_rows(True, 'mixincrossss', 'mixincrossss', 'errorevents', '', '', True)
        # logger.info('%s', ret)
        assert len(ret['rows']) == 1

        r = self.chain.push_action('mixincrossss', 'exec', {'executor': MTG_PUBLISHER}, {MTG_PUBLISHER: 'active'})
        self.chain.produce_block()
        ret = self.chain.get_table_rows(True, 'mixincrossss', 'mixincrossss', 'errorevents', '', '', True)
        logger.info('%s', ret)
        assert len(ret['rows']) == 0

    def test_pending(self):
        process_id_str = 'e0148fc6-0e10-470e-8127-166e0829c839'
        asset_id = '6cfe566e-4aad-470b-8c9a-2fd35b49c68d'
        tx_event = self.build_event(asset_id, 1, 1e8, b'hello', b'', int(time.time()*1e9))
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        print_console(r)
        logger.info('++++%s', r['elapsed'])
        self.chain.produce_block()

        # def build_extra(process, amount, address = 'mixincrossss', extra=b''):
        args = {
            'from': 'aaaaaaaaamvm',
            'to': 'eosio',
            'quantity': '0.10000000 MEOS',
            'memo': 'hello'
        }
        args = self.chain.pack_args('mixinwtokens', 'transfer', args)
        extra = b'\x00' + int.to_bytes(self.chain.s2n('mixinwtokens'), 8, 'little') + \
            int.to_bytes(self.chain.s2n('transfer'), 8, 'little') + args
        originExtra = self.build_extra(process_id_str, 'mixincrossss', extra)
        hash = hashlib.sha256(originExtra).digest()
        extra = b'\x01' + hash + 'https://a.com'.encode()
        tx_event = self.build_event(asset_id, 3, 1e8, extra, b'', int(time.time()*1e9))

        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        args = {
            'executor': MTG_PUBLISHER, 
            'nonce': 3,
            'origin_extra': originExtra.hex()
        }
        r = self.chain.push_action('mixincrossss', 'execpending', args, {MTG_PUBLISHER: 'active'})
        self.chain.produce_block()
        assert self.get_balance('aaaaaaaaamvm') == 0.9

    def test_transfer(self):
        process_id_str = 'e0148fc6-0e10-470e-8127-166e0829c839'
        asset_id = '6cfe566e-4aad-470b-8c9a-2fd35b49c68d'
        tx_event = self.build_event(asset_id, 1, 1e8, b'hello', b'', int(time.time()*1e9))
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        print_console(r)
        logger.info('++++%s', r['elapsed'])
        self.chain.produce_block()


        args = {
            'from': 'aaaaaaaaamvm',
            'to': 'hello',
            'quantity': '0.10000000 MEOS',
            'memo': 'hello'
        }
        args = self.chain.pack_args('mixinwtokens', 'transfer', args)
        extra = b'\x00' + int.to_bytes(self.chain.s2n('mixinwtokens'), 8, 'little') + \
            int.to_bytes(self.chain.s2n('transfer'), 8, 'little') + args

        tx_event = self.build_event(asset_id, 2, 1e8, extra, b'', int(time.time()*1e9))
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})

        ret = self.chain.get_table_rows(True, 'mixincrossss', 'mixincrossss', 'createaccfee', '','', 10)
        logger.info("+++++ret %s", ret)
        assert self.get_balance('aaaaaaaaamvm') ==  0.9
        assert self.get_balance('hello') == 0.1

        args = {
            'from': 'hello',
            'to': 'aaaaaaaaamvm',
            'quantity': '0.01900000 MEOS',
            'memo': 'transfer to aaaaaaaaamvm'
        }
        r = self.chain.push_action('mixinwtokens', 'transfer', args, {'hello': 'active'})
        # logger.info(r)
        for trace in r['action_traces']:
            act = trace['act']
            account, name = act['account'], act['name']
            data = act['data']
            args = self.chain.unpack_args(account, name, bytes.fromhex(data))
            logger.info("+++%s %s %s", account, name, args)
            if name == 'txrequest':
                assert args['amount'] == '1900000'
                assert args['extra'] == b'transfer to aaaaaaaaamvm'.hex()

    def test_on_event_with_origin_extra(self):
        process_id_str = 'e0148fc6-0e10-470e-8127-166e0829c839'
        asset_id = '6cfe566e-4aad-470b-8c9a-2fd35b49c68d'
        tx_event = self.build_event(asset_id, 1, 1e8, b'hello', b'', int(time.time()*1e9))
        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        print_console(r)
        logger.info('++++%s', r['elapsed'])
        self.chain.produce_block()

        # def build_extra(process, amount, address = 'mixincrossss', extra=b''):
        args = {
            'from': 'aaaaaaaaamvm',
            'to': 'eosio',
            'quantity': '0.10000000 MEOS',
            'memo': 'hello'
        }
        args = self.chain.pack_args('mixinwtokens', 'transfer', args)
        extra = b'\x00' + int.to_bytes(self.chain.s2n('mixinwtokens'), 8, 'little') + \
            int.to_bytes(self.chain.s2n('transfer'), 8, 'little') + args
        originExtra = self.build_extra(process_id_str, 'mixincrossss', extra)
        hash = hashlib.sha256(originExtra).digest()
        extra = b'\x01' + hash + 'https://a.com'.encode()
        tx_event = self.build_event(asset_id, 3, 1e8, extra, originExtra, int(time.time()*1e9))

        r = self.chain.push_action('mixincrossss', 'onevent', tx_event, {MTG_PUBLISHER: 'active'})
        args = {
            'executor': MTG_PUBLISHER, 
            'nonce': 3,
            'origin_extra': originExtra.hex()
        }
        assert self.get_balance('aaaaaaaaamvm') == 0.9

        tx_event = self.build_event(asset_id, 4, 1e8, extra, originExtra, self.current_timestamp())
        tx_event['reason'] = 'test'
        r = self.chain.push_action('mixincrossss', 'onerrorevent', tx_event, {MTG_PUBLISHER: 'active'})
        assert self.get_balance('aaaaaaaaamvm') == 0.9
        self.chain.produce_block()

        ret = self.chain.get_table_rows(True, 'mixincrossss', 'mixincrossss', 'errorevents', '', '', True)
        # logger.info('%s', ret)
        assert len(ret['rows']) == 1

        r = self.chain.push_action('mixincrossss', 'exec', {'executor': MTG_PUBLISHER}, {MTG_PUBLISHER: 'active'})
        self.chain.produce_block()
        ret = self.chain.get_table_rows(True, 'mixincrossss', 'mixincrossss', 'errorevents', '', '', True)
        logger.info('%s', ret)
        assert len(ret['rows']) == 0
        assert self.get_balance('aaaaaaaaamvm') == 1.8

    def test_debug(self):
        r = self.chain.push_action('mixincrossss', 'testname', b'', {MTG_PUBLISHER: 'active'})

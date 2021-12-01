import shlex
import subprocess
from pyeoskit import eosapi
eosapi.set_node('http://127.0.0.1:9000')
info = eosapi.get_info()
ref_block = info['last_irreversible_block_id']
ref_block = ref_block[:24]

cmd = shlex.split(f'./mvm publish -p eos -m ../../mvm-configs/test1.toml -k ../../configs/test4.json -a helloworld11 -e {ref_block}')
subprocess.call(cmd)

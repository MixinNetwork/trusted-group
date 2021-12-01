import time
import shlex
import random
import subprocess
from pyeoskit import eosapi

random.seed(time.time())

eosapi.set_node('http://127.0.0.1:9000')
info = eosapi.get_info()
ref_block = info['last_irreversible_block_id']
ref_block = ref_block[:24]

amount = random.randint(1, 100)
amount = amount/1000
cmd = f'./mvm invoke -m ../../mvm-configs/test1.toml -k ../../configs/test3.json -p 18a62033-8845-455f-bcde-0e205ef4da44 -a {amount} -e {ref_block}'
print(cmd)
cmd = shlex.split(cmd)
subprocess.call(cmd)


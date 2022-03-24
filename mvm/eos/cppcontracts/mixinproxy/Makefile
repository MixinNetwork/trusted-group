all : mixinproxy.wasm

mixinproxy.wasm : mixinproxy.cpp mixinproxy.hpp
	eosio-cpp mixinproxy.cpp

test : mixinproxy.wasm
# run-ipyeos -m pytest -x -s tests/test.py
# run-ipyeos -m pytest -x -s tests/test.py -k test_event
# run-ipyeos -m pytest -x -s tests/test.py -k test_expiration
# run-ipyeos -m pytest -x -s tests/test.py -k test_fee
	run-ipyeos -m pytest -x --log-level=1 --log-cli-level=ERROR -s tests/test.py


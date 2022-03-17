all : mixinproxy.wasm

mixinproxy.wasm : actions.go tables.go mixinproxy.go
	eosio-go build -o mixinproxy.wasm

test : mixinproxy.wasm
	run-ipyeos -m pytest -x -s tests/test.py
#	run-ipyeos -m pytest -x -s tests/test.py -k test_event

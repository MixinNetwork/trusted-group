// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package abi

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// BridgeContractMetaData contains all meta data concerning the BridgeContract contract.
var BridgeContractMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"stateMutability\":\"nonpayable\",\"inputs\":[{\"type\":\"address\",\"name\":\"factory\",\"internalType\":\"address\"},{\"type\":\"address\",\"name\":\"xin\",\"internalType\":\"address\"}]},{\"type\":\"event\",\"name\":\"Bound\",\"inputs\":[{\"type\":\"address\",\"name\":\"from\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"address\",\"name\":\"to\",\"internalType\":\"address\",\"indexed\":true}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Through\",\"inputs\":[{\"type\":\"address\",\"name\":\"asset\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"address\",\"name\":\"from\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"address\",\"name\":\"to\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"uint256\",\"name\":\"amount\",\"internalType\":\"uint256\",\"indexed\":false}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Vault\",\"inputs\":[{\"type\":\"address\",\"name\":\"from\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"uint256\",\"name\":\"amount\",\"internalType\":\"uint256\",\"indexed\":false}],\"anonymous\":false},{\"type\":\"fallback\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}],\"name\":\"BASE\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}],\"name\":\"FACTORY\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}],\"name\":\"OWNER\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}],\"name\":\"XIN\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[],\"name\":\"bind\",\"inputs\":[{\"type\":\"address\",\"name\":\"receiver\",\"internalType\":\"address\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}],\"name\":\"bridges\",\"inputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[],\"name\":\"pass\",\"inputs\":[{\"type\":\"address\",\"name\":\"asset\",\"internalType\":\"address\"},{\"type\":\"uint256\",\"name\":\"amount\",\"internalType\":\"uint256\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[],\"name\":\"vault\",\"inputs\":[{\"type\":\"address\",\"name\":\"asset\",\"internalType\":\"address\"},{\"type\":\"uint256\",\"name\":\"amount\",\"internalType\":\"uint256\"}]},{\"type\":\"receive\",\"stateMutability\":\"payable\"}]",
}

// BridgeContractABI is the input ABI used to generate the binding from.
// Deprecated: Use BridgeContractMetaData.ABI instead.
var BridgeContractABI = BridgeContractMetaData.ABI

// BridgeContract is an auto generated Go binding around an Ethereum contract.
type BridgeContract struct {
	BridgeContractCaller     // Read-only binding to the contract
	BridgeContractTransactor // Write-only binding to the contract
	BridgeContractFilterer   // Log filterer for contract events
}

// BridgeContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type BridgeContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BridgeContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BridgeContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BridgeContractSession struct {
	Contract     *BridgeContract   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BridgeContractCallerSession struct {
	Contract *BridgeContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// BridgeContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BridgeContractTransactorSession struct {
	Contract     *BridgeContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// BridgeContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type BridgeContractRaw struct {
	Contract *BridgeContract // Generic contract binding to access the raw methods on
}

// BridgeContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BridgeContractCallerRaw struct {
	Contract *BridgeContractCaller // Generic read-only contract binding to access the raw methods on
}

// BridgeContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BridgeContractTransactorRaw struct {
	Contract *BridgeContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBridgeContract creates a new instance of BridgeContract, bound to a specific deployed contract.
func NewBridgeContract(address common.Address, backend bind.ContractBackend) (*BridgeContract, error) {
	contract, err := bindBridgeContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BridgeContract{BridgeContractCaller: BridgeContractCaller{contract: contract}, BridgeContractTransactor: BridgeContractTransactor{contract: contract}, BridgeContractFilterer: BridgeContractFilterer{contract: contract}}, nil
}

// NewBridgeContractCaller creates a new read-only instance of BridgeContract, bound to a specific deployed contract.
func NewBridgeContractCaller(address common.Address, caller bind.ContractCaller) (*BridgeContractCaller, error) {
	contract, err := bindBridgeContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeContractCaller{contract: contract}, nil
}

// NewBridgeContractTransactor creates a new write-only instance of BridgeContract, bound to a specific deployed contract.
func NewBridgeContractTransactor(address common.Address, transactor bind.ContractTransactor) (*BridgeContractTransactor, error) {
	contract, err := bindBridgeContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeContractTransactor{contract: contract}, nil
}

// NewBridgeContractFilterer creates a new log filterer instance of BridgeContract, bound to a specific deployed contract.
func NewBridgeContractFilterer(address common.Address, filterer bind.ContractFilterer) (*BridgeContractFilterer, error) {
	contract, err := bindBridgeContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BridgeContractFilterer{contract: contract}, nil
}

// bindBridgeContract binds a generic wrapper to an already deployed contract.
func bindBridgeContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BridgeContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeContract *BridgeContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeContract.Contract.BridgeContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeContract *BridgeContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeContract.Contract.BridgeContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeContract *BridgeContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeContract.Contract.BridgeContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeContract *BridgeContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeContract *BridgeContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeContract *BridgeContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeContract.Contract.contract.Transact(opts, method, params...)
}

// BASE is a free data retrieval call binding the contract method 0xec342ad0.
//
// Solidity: function BASE() view returns(uint256)
func (_BridgeContract *BridgeContractCaller) BASE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeContract.contract.Call(opts, &out, "BASE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BASE is a free data retrieval call binding the contract method 0xec342ad0.
//
// Solidity: function BASE() view returns(uint256)
func (_BridgeContract *BridgeContractSession) BASE() (*big.Int, error) {
	return _BridgeContract.Contract.BASE(&_BridgeContract.CallOpts)
}

// BASE is a free data retrieval call binding the contract method 0xec342ad0.
//
// Solidity: function BASE() view returns(uint256)
func (_BridgeContract *BridgeContractCallerSession) BASE() (*big.Int, error) {
	return _BridgeContract.Contract.BASE(&_BridgeContract.CallOpts)
}

// FACTORY is a free data retrieval call binding the contract method 0x2dd31000.
//
// Solidity: function FACTORY() view returns(address)
func (_BridgeContract *BridgeContractCaller) FACTORY(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeContract.contract.Call(opts, &out, "FACTORY")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FACTORY is a free data retrieval call binding the contract method 0x2dd31000.
//
// Solidity: function FACTORY() view returns(address)
func (_BridgeContract *BridgeContractSession) FACTORY() (common.Address, error) {
	return _BridgeContract.Contract.FACTORY(&_BridgeContract.CallOpts)
}

// FACTORY is a free data retrieval call binding the contract method 0x2dd31000.
//
// Solidity: function FACTORY() view returns(address)
func (_BridgeContract *BridgeContractCallerSession) FACTORY() (common.Address, error) {
	return _BridgeContract.Contract.FACTORY(&_BridgeContract.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x117803e3.
//
// Solidity: function OWNER() view returns(address)
func (_BridgeContract *BridgeContractCaller) OWNER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeContract.contract.Call(opts, &out, "OWNER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OWNER is a free data retrieval call binding the contract method 0x117803e3.
//
// Solidity: function OWNER() view returns(address)
func (_BridgeContract *BridgeContractSession) OWNER() (common.Address, error) {
	return _BridgeContract.Contract.OWNER(&_BridgeContract.CallOpts)
}

// OWNER is a free data retrieval call binding the contract method 0x117803e3.
//
// Solidity: function OWNER() view returns(address)
func (_BridgeContract *BridgeContractCallerSession) OWNER() (common.Address, error) {
	return _BridgeContract.Contract.OWNER(&_BridgeContract.CallOpts)
}

// XIN is a free data retrieval call binding the contract method 0x7ae47b95.
//
// Solidity: function XIN() view returns(address)
func (_BridgeContract *BridgeContractCaller) XIN(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeContract.contract.Call(opts, &out, "XIN")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// XIN is a free data retrieval call binding the contract method 0x7ae47b95.
//
// Solidity: function XIN() view returns(address)
func (_BridgeContract *BridgeContractSession) XIN() (common.Address, error) {
	return _BridgeContract.Contract.XIN(&_BridgeContract.CallOpts)
}

// XIN is a free data retrieval call binding the contract method 0x7ae47b95.
//
// Solidity: function XIN() view returns(address)
func (_BridgeContract *BridgeContractCallerSession) XIN() (common.Address, error) {
	return _BridgeContract.Contract.XIN(&_BridgeContract.CallOpts)
}

// Bridges is a free data retrieval call binding the contract method 0xced67f0c.
//
// Solidity: function bridges(address ) view returns(address)
func (_BridgeContract *BridgeContractCaller) Bridges(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _BridgeContract.contract.Call(opts, &out, "bridges", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridges is a free data retrieval call binding the contract method 0xced67f0c.
//
// Solidity: function bridges(address ) view returns(address)
func (_BridgeContract *BridgeContractSession) Bridges(arg0 common.Address) (common.Address, error) {
	return _BridgeContract.Contract.Bridges(&_BridgeContract.CallOpts, arg0)
}

// Bridges is a free data retrieval call binding the contract method 0xced67f0c.
//
// Solidity: function bridges(address ) view returns(address)
func (_BridgeContract *BridgeContractCallerSession) Bridges(arg0 common.Address) (common.Address, error) {
	return _BridgeContract.Contract.Bridges(&_BridgeContract.CallOpts, arg0)
}

// Bind is a paid mutator transaction binding the contract method 0x81bac14f.
//
// Solidity: function bind(address receiver) returns()
func (_BridgeContract *BridgeContractTransactor) Bind(opts *bind.TransactOpts, receiver common.Address) (*types.Transaction, error) {
	return _BridgeContract.contract.Transact(opts, "bind", receiver)
}

// Bind is a paid mutator transaction binding the contract method 0x81bac14f.
//
// Solidity: function bind(address receiver) returns()
func (_BridgeContract *BridgeContractSession) Bind(receiver common.Address) (*types.Transaction, error) {
	return _BridgeContract.Contract.Bind(&_BridgeContract.TransactOpts, receiver)
}

// Bind is a paid mutator transaction binding the contract method 0x81bac14f.
//
// Solidity: function bind(address receiver) returns()
func (_BridgeContract *BridgeContractTransactorSession) Bind(receiver common.Address) (*types.Transaction, error) {
	return _BridgeContract.Contract.Bind(&_BridgeContract.TransactOpts, receiver)
}

// Pass is a paid mutator transaction binding the contract method 0x0ed1db9f.
//
// Solidity: function pass(address asset, uint256 amount) returns()
func (_BridgeContract *BridgeContractTransactor) Pass(opts *bind.TransactOpts, asset common.Address, amount *big.Int) (*types.Transaction, error) {
	return _BridgeContract.contract.Transact(opts, "pass", asset, amount)
}

// Pass is a paid mutator transaction binding the contract method 0x0ed1db9f.
//
// Solidity: function pass(address asset, uint256 amount) returns()
func (_BridgeContract *BridgeContractSession) Pass(asset common.Address, amount *big.Int) (*types.Transaction, error) {
	return _BridgeContract.Contract.Pass(&_BridgeContract.TransactOpts, asset, amount)
}

// Pass is a paid mutator transaction binding the contract method 0x0ed1db9f.
//
// Solidity: function pass(address asset, uint256 amount) returns()
func (_BridgeContract *BridgeContractTransactorSession) Pass(asset common.Address, amount *big.Int) (*types.Transaction, error) {
	return _BridgeContract.Contract.Pass(&_BridgeContract.TransactOpts, asset, amount)
}

// Vault is a paid mutator transaction binding the contract method 0x3fa16d99.
//
// Solidity: function vault(address asset, uint256 amount) returns()
func (_BridgeContract *BridgeContractTransactor) Vault(opts *bind.TransactOpts, asset common.Address, amount *big.Int) (*types.Transaction, error) {
	return _BridgeContract.contract.Transact(opts, "vault", asset, amount)
}

// Vault is a paid mutator transaction binding the contract method 0x3fa16d99.
//
// Solidity: function vault(address asset, uint256 amount) returns()
func (_BridgeContract *BridgeContractSession) Vault(asset common.Address, amount *big.Int) (*types.Transaction, error) {
	return _BridgeContract.Contract.Vault(&_BridgeContract.TransactOpts, asset, amount)
}

// Vault is a paid mutator transaction binding the contract method 0x3fa16d99.
//
// Solidity: function vault(address asset, uint256 amount) returns()
func (_BridgeContract *BridgeContractTransactorSession) Vault(asset common.Address, amount *big.Int) (*types.Transaction, error) {
	return _BridgeContract.Contract.Vault(&_BridgeContract.TransactOpts, asset, amount)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_BridgeContract *BridgeContractTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _BridgeContract.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_BridgeContract *BridgeContractSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _BridgeContract.Contract.Fallback(&_BridgeContract.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_BridgeContract *BridgeContractTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _BridgeContract.Contract.Fallback(&_BridgeContract.TransactOpts, calldata)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BridgeContract *BridgeContractTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeContract.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BridgeContract *BridgeContractSession) Receive() (*types.Transaction, error) {
	return _BridgeContract.Contract.Receive(&_BridgeContract.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BridgeContract *BridgeContractTransactorSession) Receive() (*types.Transaction, error) {
	return _BridgeContract.Contract.Receive(&_BridgeContract.TransactOpts)
}

// BridgeContractBoundIterator is returned from FilterBound and is used to iterate over the raw logs and unpacked data for Bound events raised by the BridgeContract contract.
type BridgeContractBoundIterator struct {
	Event *BridgeContractBound // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *BridgeContractBoundIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeContractBound)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(BridgeContractBound)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *BridgeContractBoundIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeContractBoundIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeContractBound represents a Bound event raised by the BridgeContract contract.
type BridgeContractBound struct {
	From common.Address
	To   common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterBound is a free log retrieval operation binding the contract event 0x0d128562eaa47ab89086803e64a0f96847c0ed3cc63c26251f29ba1aede09d4e.
//
// Solidity: event Bound(address indexed from, address indexed to)
func (_BridgeContract *BridgeContractFilterer) FilterBound(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*BridgeContractBoundIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeContract.contract.FilterLogs(opts, "Bound", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &BridgeContractBoundIterator{contract: _BridgeContract.contract, event: "Bound", logs: logs, sub: sub}, nil
}

// WatchBound is a free log subscription operation binding the contract event 0x0d128562eaa47ab89086803e64a0f96847c0ed3cc63c26251f29ba1aede09d4e.
//
// Solidity: event Bound(address indexed from, address indexed to)
func (_BridgeContract *BridgeContractFilterer) WatchBound(opts *bind.WatchOpts, sink chan<- *BridgeContractBound, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeContract.contract.WatchLogs(opts, "Bound", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeContractBound)
				if err := _BridgeContract.contract.UnpackLog(event, "Bound", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBound is a log parse operation binding the contract event 0x0d128562eaa47ab89086803e64a0f96847c0ed3cc63c26251f29ba1aede09d4e.
//
// Solidity: event Bound(address indexed from, address indexed to)
func (_BridgeContract *BridgeContractFilterer) ParseBound(log types.Log) (*BridgeContractBound, error) {
	event := new(BridgeContractBound)
	if err := _BridgeContract.contract.UnpackLog(event, "Bound", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeContractThroughIterator is returned from FilterThrough and is used to iterate over the raw logs and unpacked data for Through events raised by the BridgeContract contract.
type BridgeContractThroughIterator struct {
	Event *BridgeContractThrough // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *BridgeContractThroughIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeContractThrough)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(BridgeContractThrough)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *BridgeContractThroughIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeContractThroughIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeContractThrough represents a Through event raised by the BridgeContract contract.
type BridgeContractThrough struct {
	Asset  common.Address
	From   common.Address
	To     common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterThrough is a free log retrieval operation binding the contract event 0x72d27832ca9ac59ff168a29801fcbe15464d099fb1b06cea8475d4a9d47a2248.
//
// Solidity: event Through(address indexed asset, address indexed from, address indexed to, uint256 amount)
func (_BridgeContract *BridgeContractFilterer) FilterThrough(opts *bind.FilterOpts, asset []common.Address, from []common.Address, to []common.Address) (*BridgeContractThroughIterator, error) {

	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeContract.contract.FilterLogs(opts, "Through", assetRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &BridgeContractThroughIterator{contract: _BridgeContract.contract, event: "Through", logs: logs, sub: sub}, nil
}

// WatchThrough is a free log subscription operation binding the contract event 0x72d27832ca9ac59ff168a29801fcbe15464d099fb1b06cea8475d4a9d47a2248.
//
// Solidity: event Through(address indexed asset, address indexed from, address indexed to, uint256 amount)
func (_BridgeContract *BridgeContractFilterer) WatchThrough(opts *bind.WatchOpts, sink chan<- *BridgeContractThrough, asset []common.Address, from []common.Address, to []common.Address) (event.Subscription, error) {

	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _BridgeContract.contract.WatchLogs(opts, "Through", assetRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeContractThrough)
				if err := _BridgeContract.contract.UnpackLog(event, "Through", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseThrough is a log parse operation binding the contract event 0x72d27832ca9ac59ff168a29801fcbe15464d099fb1b06cea8475d4a9d47a2248.
//
// Solidity: event Through(address indexed asset, address indexed from, address indexed to, uint256 amount)
func (_BridgeContract *BridgeContractFilterer) ParseThrough(log types.Log) (*BridgeContractThrough, error) {
	event := new(BridgeContractThrough)
	if err := _BridgeContract.contract.UnpackLog(event, "Through", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeContractVaultIterator is returned from FilterVault and is used to iterate over the raw logs and unpacked data for Vault events raised by the BridgeContract contract.
type BridgeContractVaultIterator struct {
	Event *BridgeContractVault // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *BridgeContractVaultIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeContractVault)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(BridgeContractVault)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *BridgeContractVaultIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeContractVaultIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeContractVault represents a Vault event raised by the BridgeContract contract.
type BridgeContractVault struct {
	From   common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterVault is a free log retrieval operation binding the contract event 0xcc189d00e17c637536854a6446232b39c2adbc24668adad4fa348e9ee1eb37b1.
//
// Solidity: event Vault(address indexed from, uint256 amount)
func (_BridgeContract *BridgeContractFilterer) FilterVault(opts *bind.FilterOpts, from []common.Address) (*BridgeContractVaultIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _BridgeContract.contract.FilterLogs(opts, "Vault", fromRule)
	if err != nil {
		return nil, err
	}
	return &BridgeContractVaultIterator{contract: _BridgeContract.contract, event: "Vault", logs: logs, sub: sub}, nil
}

// WatchVault is a free log subscription operation binding the contract event 0xcc189d00e17c637536854a6446232b39c2adbc24668adad4fa348e9ee1eb37b1.
//
// Solidity: event Vault(address indexed from, uint256 amount)
func (_BridgeContract *BridgeContractFilterer) WatchVault(opts *bind.WatchOpts, sink chan<- *BridgeContractVault, from []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _BridgeContract.contract.WatchLogs(opts, "Vault", fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeContractVault)
				if err := _BridgeContract.contract.UnpackLog(event, "Vault", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseVault is a log parse operation binding the contract event 0xcc189d00e17c637536854a6446232b39c2adbc24668adad4fa348e9ee1eb37b1.
//
// Solidity: event Vault(address indexed from, uint256 amount)
func (_BridgeContract *BridgeContractFilterer) ParseVault(log types.Log) (*BridgeContractVault, error) {
	event := new(BridgeContractVault)
	if err := _BridgeContract.contract.UnpackLog(event, "Vault", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

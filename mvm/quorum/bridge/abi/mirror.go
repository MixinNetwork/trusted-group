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

// MirrorContractMetaData contains all meta data concerning the MirrorContract contract.
var MirrorContractMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"stateMutability\":\"nonpayable\",\"inputs\":[{\"type\":\"address\",\"name\":\"factory\",\"internalType\":\"address\"}]},{\"type\":\"event\",\"name\":\"Bound\",\"inputs\":[{\"type\":\"address\",\"name\":\"from\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"address\",\"name\":\"to\",\"internalType\":\"address\",\"indexed\":true}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"CollectionCreated\",\"inputs\":[{\"type\":\"address\",\"name\":\"at\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"uint256\",\"name\":\"id\",\"internalType\":\"uint256\",\"indexed\":false}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Through\",\"inputs\":[{\"type\":\"address\",\"name\":\"collection\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"address\",\"name\":\"from\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"address\",\"name\":\"to\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"uint256\",\"name\":\"id\",\"internalType\":\"uint256\",\"indexed\":false}],\"anonymous\":false},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}],\"name\":\"FACTORY\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}],\"name\":\"VERSION\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[],\"name\":\"bind\",\"inputs\":[{\"type\":\"address\",\"name\":\"receiver\",\"internalType\":\"address\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}],\"name\":\"bridges\",\"inputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}],\"name\":\"collections\",\"inputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}],\"name\":\"contracts\",\"inputs\":[{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"collection\",\"internalType\":\"address\"},{\"type\":\"uint256\",\"name\":\"id\",\"internalType\":\"uint256\"}],\"name\":\"mints\",\"inputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[{\"type\":\"bytes4\",\"name\":\"\",\"internalType\":\"bytes4\"}],\"name\":\"onERC721Received\",\"inputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"},{\"type\":\"address\",\"name\":\"_from\",\"internalType\":\"address\"},{\"type\":\"uint256\",\"name\":\"_tokenId\",\"internalType\":\"uint256\"},{\"type\":\"bytes\",\"name\":\"_data\",\"internalType\":\"bytes\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[],\"name\":\"pass\",\"inputs\":[{\"type\":\"address\",\"name\":\"asset\",\"internalType\":\"address\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}],\"name\":\"tokens\",\"inputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"},{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}]}]",
}

// MirrorContractABI is the input ABI used to generate the binding from.
// Deprecated: Use MirrorContractMetaData.ABI instead.
var MirrorContractABI = MirrorContractMetaData.ABI

// MirrorContract is an auto generated Go binding around an Ethereum contract.
type MirrorContract struct {
	MirrorContractCaller     // Read-only binding to the contract
	MirrorContractTransactor // Write-only binding to the contract
	MirrorContractFilterer   // Log filterer for contract events
}

// MirrorContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type MirrorContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MirrorContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MirrorContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MirrorContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MirrorContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MirrorContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MirrorContractSession struct {
	Contract     *MirrorContract   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MirrorContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MirrorContractCallerSession struct {
	Contract *MirrorContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// MirrorContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MirrorContractTransactorSession struct {
	Contract     *MirrorContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// MirrorContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type MirrorContractRaw struct {
	Contract *MirrorContract // Generic contract binding to access the raw methods on
}

// MirrorContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MirrorContractCallerRaw struct {
	Contract *MirrorContractCaller // Generic read-only contract binding to access the raw methods on
}

// MirrorContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MirrorContractTransactorRaw struct {
	Contract *MirrorContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMirrorContract creates a new instance of MirrorContract, bound to a specific deployed contract.
func NewMirrorContract(address common.Address, backend bind.ContractBackend) (*MirrorContract, error) {
	contract, err := bindMirrorContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MirrorContract{MirrorContractCaller: MirrorContractCaller{contract: contract}, MirrorContractTransactor: MirrorContractTransactor{contract: contract}, MirrorContractFilterer: MirrorContractFilterer{contract: contract}}, nil
}

// NewMirrorContractCaller creates a new read-only instance of MirrorContract, bound to a specific deployed contract.
func NewMirrorContractCaller(address common.Address, caller bind.ContractCaller) (*MirrorContractCaller, error) {
	contract, err := bindMirrorContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MirrorContractCaller{contract: contract}, nil
}

// NewMirrorContractTransactor creates a new write-only instance of MirrorContract, bound to a specific deployed contract.
func NewMirrorContractTransactor(address common.Address, transactor bind.ContractTransactor) (*MirrorContractTransactor, error) {
	contract, err := bindMirrorContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MirrorContractTransactor{contract: contract}, nil
}

// NewMirrorContractFilterer creates a new log filterer instance of MirrorContract, bound to a specific deployed contract.
func NewMirrorContractFilterer(address common.Address, filterer bind.ContractFilterer) (*MirrorContractFilterer, error) {
	contract, err := bindMirrorContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MirrorContractFilterer{contract: contract}, nil
}

// bindMirrorContract binds a generic wrapper to an already deployed contract.
func bindMirrorContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MirrorContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MirrorContract *MirrorContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MirrorContract.Contract.MirrorContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MirrorContract *MirrorContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MirrorContract.Contract.MirrorContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MirrorContract *MirrorContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MirrorContract.Contract.MirrorContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MirrorContract *MirrorContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MirrorContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MirrorContract *MirrorContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MirrorContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MirrorContract *MirrorContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MirrorContract.Contract.contract.Transact(opts, method, params...)
}

// FACTORY is a free data retrieval call binding the contract method 0x2dd31000.
//
// Solidity: function FACTORY() view returns(address)
func (_MirrorContract *MirrorContractCaller) FACTORY(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MirrorContract.contract.Call(opts, &out, "FACTORY")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FACTORY is a free data retrieval call binding the contract method 0x2dd31000.
//
// Solidity: function FACTORY() view returns(address)
func (_MirrorContract *MirrorContractSession) FACTORY() (common.Address, error) {
	return _MirrorContract.Contract.FACTORY(&_MirrorContract.CallOpts)
}

// FACTORY is a free data retrieval call binding the contract method 0x2dd31000.
//
// Solidity: function FACTORY() view returns(address)
func (_MirrorContract *MirrorContractCallerSession) FACTORY() (common.Address, error) {
	return _MirrorContract.Contract.FACTORY(&_MirrorContract.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_MirrorContract *MirrorContractCaller) VERSION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MirrorContract.contract.Call(opts, &out, "VERSION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_MirrorContract *MirrorContractSession) VERSION() (*big.Int, error) {
	return _MirrorContract.Contract.VERSION(&_MirrorContract.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_MirrorContract *MirrorContractCallerSession) VERSION() (*big.Int, error) {
	return _MirrorContract.Contract.VERSION(&_MirrorContract.CallOpts)
}

// Bridges is a free data retrieval call binding the contract method 0xced67f0c.
//
// Solidity: function bridges(address ) view returns(address)
func (_MirrorContract *MirrorContractCaller) Bridges(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _MirrorContract.contract.Call(opts, &out, "bridges", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridges is a free data retrieval call binding the contract method 0xced67f0c.
//
// Solidity: function bridges(address ) view returns(address)
func (_MirrorContract *MirrorContractSession) Bridges(arg0 common.Address) (common.Address, error) {
	return _MirrorContract.Contract.Bridges(&_MirrorContract.CallOpts, arg0)
}

// Bridges is a free data retrieval call binding the contract method 0xced67f0c.
//
// Solidity: function bridges(address ) view returns(address)
func (_MirrorContract *MirrorContractCallerSession) Bridges(arg0 common.Address) (common.Address, error) {
	return _MirrorContract.Contract.Bridges(&_MirrorContract.CallOpts, arg0)
}

// Collections is a free data retrieval call binding the contract method 0x43add2e6.
//
// Solidity: function collections(address ) view returns(uint256)
func (_MirrorContract *MirrorContractCaller) Collections(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MirrorContract.contract.Call(opts, &out, "collections", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Collections is a free data retrieval call binding the contract method 0x43add2e6.
//
// Solidity: function collections(address ) view returns(uint256)
func (_MirrorContract *MirrorContractSession) Collections(arg0 common.Address) (*big.Int, error) {
	return _MirrorContract.Contract.Collections(&_MirrorContract.CallOpts, arg0)
}

// Collections is a free data retrieval call binding the contract method 0x43add2e6.
//
// Solidity: function collections(address ) view returns(uint256)
func (_MirrorContract *MirrorContractCallerSession) Collections(arg0 common.Address) (*big.Int, error) {
	return _MirrorContract.Contract.Collections(&_MirrorContract.CallOpts, arg0)
}

// Contracts is a free data retrieval call binding the contract method 0x474da79a.
//
// Solidity: function contracts(uint256 ) view returns(address)
func (_MirrorContract *MirrorContractCaller) Contracts(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _MirrorContract.contract.Call(opts, &out, "contracts", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Contracts is a free data retrieval call binding the contract method 0x474da79a.
//
// Solidity: function contracts(uint256 ) view returns(address)
func (_MirrorContract *MirrorContractSession) Contracts(arg0 *big.Int) (common.Address, error) {
	return _MirrorContract.Contract.Contracts(&_MirrorContract.CallOpts, arg0)
}

// Contracts is a free data retrieval call binding the contract method 0x474da79a.
//
// Solidity: function contracts(uint256 ) view returns(address)
func (_MirrorContract *MirrorContractCallerSession) Contracts(arg0 *big.Int) (common.Address, error) {
	return _MirrorContract.Contract.Contracts(&_MirrorContract.CallOpts, arg0)
}

// Mints is a free data retrieval call binding the contract method 0x5660f851.
//
// Solidity: function mints(address ) view returns(address collection, uint256 id)
func (_MirrorContract *MirrorContractCaller) Mints(opts *bind.CallOpts, arg0 common.Address) (struct {
	Collection common.Address
	Id         *big.Int
}, error) {
	var out []interface{}
	err := _MirrorContract.contract.Call(opts, &out, "mints", arg0)

	outstruct := new(struct {
		Collection common.Address
		Id         *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Collection = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Id = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Mints is a free data retrieval call binding the contract method 0x5660f851.
//
// Solidity: function mints(address ) view returns(address collection, uint256 id)
func (_MirrorContract *MirrorContractSession) Mints(arg0 common.Address) (struct {
	Collection common.Address
	Id         *big.Int
}, error) {
	return _MirrorContract.Contract.Mints(&_MirrorContract.CallOpts, arg0)
}

// Mints is a free data retrieval call binding the contract method 0x5660f851.
//
// Solidity: function mints(address ) view returns(address collection, uint256 id)
func (_MirrorContract *MirrorContractCallerSession) Mints(arg0 common.Address) (struct {
	Collection common.Address
	Id         *big.Int
}, error) {
	return _MirrorContract.Contract.Mints(&_MirrorContract.CallOpts, arg0)
}

// Tokens is a free data retrieval call binding the contract method 0x4abf825d.
//
// Solidity: function tokens(address , uint256 ) view returns(address)
func (_MirrorContract *MirrorContractCaller) Tokens(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _MirrorContract.contract.Call(opts, &out, "tokens", arg0, arg1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Tokens is a free data retrieval call binding the contract method 0x4abf825d.
//
// Solidity: function tokens(address , uint256 ) view returns(address)
func (_MirrorContract *MirrorContractSession) Tokens(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _MirrorContract.Contract.Tokens(&_MirrorContract.CallOpts, arg0, arg1)
}

// Tokens is a free data retrieval call binding the contract method 0x4abf825d.
//
// Solidity: function tokens(address , uint256 ) view returns(address)
func (_MirrorContract *MirrorContractCallerSession) Tokens(arg0 common.Address, arg1 *big.Int) (common.Address, error) {
	return _MirrorContract.Contract.Tokens(&_MirrorContract.CallOpts, arg0, arg1)
}

// Bind is a paid mutator transaction binding the contract method 0x81bac14f.
//
// Solidity: function bind(address receiver) returns()
func (_MirrorContract *MirrorContractTransactor) Bind(opts *bind.TransactOpts, receiver common.Address) (*types.Transaction, error) {
	return _MirrorContract.contract.Transact(opts, "bind", receiver)
}

// Bind is a paid mutator transaction binding the contract method 0x81bac14f.
//
// Solidity: function bind(address receiver) returns()
func (_MirrorContract *MirrorContractSession) Bind(receiver common.Address) (*types.Transaction, error) {
	return _MirrorContract.Contract.Bind(&_MirrorContract.TransactOpts, receiver)
}

// Bind is a paid mutator transaction binding the contract method 0x81bac14f.
//
// Solidity: function bind(address receiver) returns()
func (_MirrorContract *MirrorContractTransactorSession) Bind(receiver common.Address) (*types.Transaction, error) {
	return _MirrorContract.Contract.Bind(&_MirrorContract.TransactOpts, receiver)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address _from, uint256 _tokenId, bytes _data) returns(bytes4)
func (_MirrorContract *MirrorContractTransactor) OnERC721Received(opts *bind.TransactOpts, arg0 common.Address, _from common.Address, _tokenId *big.Int, _data []byte) (*types.Transaction, error) {
	return _MirrorContract.contract.Transact(opts, "onERC721Received", arg0, _from, _tokenId, _data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address _from, uint256 _tokenId, bytes _data) returns(bytes4)
func (_MirrorContract *MirrorContractSession) OnERC721Received(arg0 common.Address, _from common.Address, _tokenId *big.Int, _data []byte) (*types.Transaction, error) {
	return _MirrorContract.Contract.OnERC721Received(&_MirrorContract.TransactOpts, arg0, _from, _tokenId, _data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address _from, uint256 _tokenId, bytes _data) returns(bytes4)
func (_MirrorContract *MirrorContractTransactorSession) OnERC721Received(arg0 common.Address, _from common.Address, _tokenId *big.Int, _data []byte) (*types.Transaction, error) {
	return _MirrorContract.Contract.OnERC721Received(&_MirrorContract.TransactOpts, arg0, _from, _tokenId, _data)
}

// Pass is a paid mutator transaction binding the contract method 0x82c4b3b2.
//
// Solidity: function pass(address asset) returns()
func (_MirrorContract *MirrorContractTransactor) Pass(opts *bind.TransactOpts, asset common.Address) (*types.Transaction, error) {
	return _MirrorContract.contract.Transact(opts, "pass", asset)
}

// Pass is a paid mutator transaction binding the contract method 0x82c4b3b2.
//
// Solidity: function pass(address asset) returns()
func (_MirrorContract *MirrorContractSession) Pass(asset common.Address) (*types.Transaction, error) {
	return _MirrorContract.Contract.Pass(&_MirrorContract.TransactOpts, asset)
}

// Pass is a paid mutator transaction binding the contract method 0x82c4b3b2.
//
// Solidity: function pass(address asset) returns()
func (_MirrorContract *MirrorContractTransactorSession) Pass(asset common.Address) (*types.Transaction, error) {
	return _MirrorContract.Contract.Pass(&_MirrorContract.TransactOpts, asset)
}

// MirrorContractBoundIterator is returned from FilterBound and is used to iterate over the raw logs and unpacked data for Bound events raised by the MirrorContract contract.
type MirrorContractBoundIterator struct {
	Event *MirrorContractBound // Event containing the contract specifics and raw log

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
func (it *MirrorContractBoundIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MirrorContractBound)
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
		it.Event = new(MirrorContractBound)
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
func (it *MirrorContractBoundIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MirrorContractBoundIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MirrorContractBound represents a Bound event raised by the MirrorContract contract.
type MirrorContractBound struct {
	From common.Address
	To   common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterBound is a free log retrieval operation binding the contract event 0x0d128562eaa47ab89086803e64a0f96847c0ed3cc63c26251f29ba1aede09d4e.
//
// Solidity: event Bound(address indexed from, address indexed to)
func (_MirrorContract *MirrorContractFilterer) FilterBound(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*MirrorContractBoundIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MirrorContract.contract.FilterLogs(opts, "Bound", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &MirrorContractBoundIterator{contract: _MirrorContract.contract, event: "Bound", logs: logs, sub: sub}, nil
}

// WatchBound is a free log subscription operation binding the contract event 0x0d128562eaa47ab89086803e64a0f96847c0ed3cc63c26251f29ba1aede09d4e.
//
// Solidity: event Bound(address indexed from, address indexed to)
func (_MirrorContract *MirrorContractFilterer) WatchBound(opts *bind.WatchOpts, sink chan<- *MirrorContractBound, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MirrorContract.contract.WatchLogs(opts, "Bound", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MirrorContractBound)
				if err := _MirrorContract.contract.UnpackLog(event, "Bound", log); err != nil {
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
func (_MirrorContract *MirrorContractFilterer) ParseBound(log types.Log) (*MirrorContractBound, error) {
	event := new(MirrorContractBound)
	if err := _MirrorContract.contract.UnpackLog(event, "Bound", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MirrorContractCollectionCreatedIterator is returned from FilterCollectionCreated and is used to iterate over the raw logs and unpacked data for CollectionCreated events raised by the MirrorContract contract.
type MirrorContractCollectionCreatedIterator struct {
	Event *MirrorContractCollectionCreated // Event containing the contract specifics and raw log

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
func (it *MirrorContractCollectionCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MirrorContractCollectionCreated)
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
		it.Event = new(MirrorContractCollectionCreated)
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
func (it *MirrorContractCollectionCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MirrorContractCollectionCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MirrorContractCollectionCreated represents a CollectionCreated event raised by the MirrorContract contract.
type MirrorContractCollectionCreated struct {
	At  common.Address
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterCollectionCreated is a free log retrieval operation binding the contract event 0xae66da0f1658d46bfee58255b370697fedca10e984d310c30f2579d377a04255.
//
// Solidity: event CollectionCreated(address indexed at, uint256 id)
func (_MirrorContract *MirrorContractFilterer) FilterCollectionCreated(opts *bind.FilterOpts, at []common.Address) (*MirrorContractCollectionCreatedIterator, error) {

	var atRule []interface{}
	for _, atItem := range at {
		atRule = append(atRule, atItem)
	}

	logs, sub, err := _MirrorContract.contract.FilterLogs(opts, "CollectionCreated", atRule)
	if err != nil {
		return nil, err
	}
	return &MirrorContractCollectionCreatedIterator{contract: _MirrorContract.contract, event: "CollectionCreated", logs: logs, sub: sub}, nil
}

// WatchCollectionCreated is a free log subscription operation binding the contract event 0xae66da0f1658d46bfee58255b370697fedca10e984d310c30f2579d377a04255.
//
// Solidity: event CollectionCreated(address indexed at, uint256 id)
func (_MirrorContract *MirrorContractFilterer) WatchCollectionCreated(opts *bind.WatchOpts, sink chan<- *MirrorContractCollectionCreated, at []common.Address) (event.Subscription, error) {

	var atRule []interface{}
	for _, atItem := range at {
		atRule = append(atRule, atItem)
	}

	logs, sub, err := _MirrorContract.contract.WatchLogs(opts, "CollectionCreated", atRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MirrorContractCollectionCreated)
				if err := _MirrorContract.contract.UnpackLog(event, "CollectionCreated", log); err != nil {
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

// ParseCollectionCreated is a log parse operation binding the contract event 0xae66da0f1658d46bfee58255b370697fedca10e984d310c30f2579d377a04255.
//
// Solidity: event CollectionCreated(address indexed at, uint256 id)
func (_MirrorContract *MirrorContractFilterer) ParseCollectionCreated(log types.Log) (*MirrorContractCollectionCreated, error) {
	event := new(MirrorContractCollectionCreated)
	if err := _MirrorContract.contract.UnpackLog(event, "CollectionCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MirrorContractThroughIterator is returned from FilterThrough and is used to iterate over the raw logs and unpacked data for Through events raised by the MirrorContract contract.
type MirrorContractThroughIterator struct {
	Event *MirrorContractThrough // Event containing the contract specifics and raw log

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
func (it *MirrorContractThroughIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MirrorContractThrough)
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
		it.Event = new(MirrorContractThrough)
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
func (it *MirrorContractThroughIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MirrorContractThroughIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MirrorContractThrough represents a Through event raised by the MirrorContract contract.
type MirrorContractThrough struct {
	Collection common.Address
	From       common.Address
	To         common.Address
	Id         *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterThrough is a free log retrieval operation binding the contract event 0x72d27832ca9ac59ff168a29801fcbe15464d099fb1b06cea8475d4a9d47a2248.
//
// Solidity: event Through(address indexed collection, address indexed from, address indexed to, uint256 id)
func (_MirrorContract *MirrorContractFilterer) FilterThrough(opts *bind.FilterOpts, collection []common.Address, from []common.Address, to []common.Address) (*MirrorContractThroughIterator, error) {

	var collectionRule []interface{}
	for _, collectionItem := range collection {
		collectionRule = append(collectionRule, collectionItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MirrorContract.contract.FilterLogs(opts, "Through", collectionRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &MirrorContractThroughIterator{contract: _MirrorContract.contract, event: "Through", logs: logs, sub: sub}, nil
}

// WatchThrough is a free log subscription operation binding the contract event 0x72d27832ca9ac59ff168a29801fcbe15464d099fb1b06cea8475d4a9d47a2248.
//
// Solidity: event Through(address indexed collection, address indexed from, address indexed to, uint256 id)
func (_MirrorContract *MirrorContractFilterer) WatchThrough(opts *bind.WatchOpts, sink chan<- *MirrorContractThrough, collection []common.Address, from []common.Address, to []common.Address) (event.Subscription, error) {

	var collectionRule []interface{}
	for _, collectionItem := range collection {
		collectionRule = append(collectionRule, collectionItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MirrorContract.contract.WatchLogs(opts, "Through", collectionRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MirrorContractThrough)
				if err := _MirrorContract.contract.UnpackLog(event, "Through", log); err != nil {
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
// Solidity: event Through(address indexed collection, address indexed from, address indexed to, uint256 id)
func (_MirrorContract *MirrorContractFilterer) ParseThrough(log types.Log) (*MirrorContractThrough, error) {
	event := new(MirrorContractThrough)
	if err := _MirrorContract.contract.UnpackLog(event, "Through", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

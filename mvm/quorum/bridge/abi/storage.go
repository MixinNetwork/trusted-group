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

// StorageContractMetaData contains all meta data concerning the StorageContract contract.
var StorageContractMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"\",\"internalType\":\"bytes\"}],\"name\":\"read\",\"inputs\":[{\"type\":\"uint256\",\"name\":\"_key\",\"internalType\":\"uint256\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[],\"name\":\"write\",\"inputs\":[{\"type\":\"uint256\",\"name\":\"_key\",\"internalType\":\"uint256\"},{\"type\":\"bytes\",\"name\":\"raw\",\"internalType\":\"bytes\"}]}]",
}

// StorageContractABI is the input ABI used to generate the binding from.
// Deprecated: Use StorageContractMetaData.ABI instead.
var StorageContractABI = StorageContractMetaData.ABI

// StorageContract is an auto generated Go binding around an Ethereum contract.
type StorageContract struct {
	StorageContractCaller     // Read-only binding to the contract
	StorageContractTransactor // Write-only binding to the contract
	StorageContractFilterer   // Log filterer for contract events
}

// StorageContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type StorageContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StorageContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StorageContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StorageContractSession struct {
	Contract     *StorageContract  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StorageContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StorageContractCallerSession struct {
	Contract *StorageContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// StorageContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StorageContractTransactorSession struct {
	Contract     *StorageContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// StorageContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type StorageContractRaw struct {
	Contract *StorageContract // Generic contract binding to access the raw methods on
}

// StorageContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StorageContractCallerRaw struct {
	Contract *StorageContractCaller // Generic read-only contract binding to access the raw methods on
}

// StorageContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StorageContractTransactorRaw struct {
	Contract *StorageContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStorageContract creates a new instance of StorageContract, bound to a specific deployed contract.
func NewStorageContract(address common.Address, backend bind.ContractBackend) (*StorageContract, error) {
	contract, err := bindStorageContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StorageContract{StorageContractCaller: StorageContractCaller{contract: contract}, StorageContractTransactor: StorageContractTransactor{contract: contract}, StorageContractFilterer: StorageContractFilterer{contract: contract}}, nil
}

// NewStorageContractCaller creates a new read-only instance of StorageContract, bound to a specific deployed contract.
func NewStorageContractCaller(address common.Address, caller bind.ContractCaller) (*StorageContractCaller, error) {
	contract, err := bindStorageContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StorageContractCaller{contract: contract}, nil
}

// NewStorageContractTransactor creates a new write-only instance of StorageContract, bound to a specific deployed contract.
func NewStorageContractTransactor(address common.Address, transactor bind.ContractTransactor) (*StorageContractTransactor, error) {
	contract, err := bindStorageContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StorageContractTransactor{contract: contract}, nil
}

// NewStorageContractFilterer creates a new log filterer instance of StorageContract, bound to a specific deployed contract.
func NewStorageContractFilterer(address common.Address, filterer bind.ContractFilterer) (*StorageContractFilterer, error) {
	contract, err := bindStorageContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StorageContractFilterer{contract: contract}, nil
}

// bindStorageContract binds a generic wrapper to an already deployed contract.
func bindStorageContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(StorageContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StorageContract *StorageContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StorageContract.Contract.StorageContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StorageContract *StorageContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StorageContract.Contract.StorageContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StorageContract *StorageContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StorageContract.Contract.StorageContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StorageContract *StorageContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StorageContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StorageContract *StorageContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StorageContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StorageContract *StorageContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StorageContract.Contract.contract.Transact(opts, method, params...)
}

// Read is a free data retrieval call binding the contract method 0xed2e5a97.
//
// Solidity: function read(uint256 _key) view returns(bytes)
func (_StorageContract *StorageContractCaller) Read(opts *bind.CallOpts, _key *big.Int) ([]byte, error) {
	var out []interface{}
	err := _StorageContract.contract.Call(opts, &out, "read", _key)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Read is a free data retrieval call binding the contract method 0xed2e5a97.
//
// Solidity: function read(uint256 _key) view returns(bytes)
func (_StorageContract *StorageContractSession) Read(_key *big.Int) ([]byte, error) {
	return _StorageContract.Contract.Read(&_StorageContract.CallOpts, _key)
}

// Read is a free data retrieval call binding the contract method 0xed2e5a97.
//
// Solidity: function read(uint256 _key) view returns(bytes)
func (_StorageContract *StorageContractCallerSession) Read(_key *big.Int) ([]byte, error) {
	return _StorageContract.Contract.Read(&_StorageContract.CallOpts, _key)
}

// Write is a paid mutator transaction binding the contract method 0x7341a70e.
//
// Solidity: function write(uint256 _key, bytes raw) returns()
func (_StorageContract *StorageContractTransactor) Write(opts *bind.TransactOpts, _key *big.Int, raw []byte) (*types.Transaction, error) {
	return _StorageContract.contract.Transact(opts, "write", _key, raw)
}

// Write is a paid mutator transaction binding the contract method 0x7341a70e.
//
// Solidity: function write(uint256 _key, bytes raw) returns()
func (_StorageContract *StorageContractSession) Write(_key *big.Int, raw []byte) (*types.Transaction, error) {
	return _StorageContract.Contract.Write(&_StorageContract.TransactOpts, _key, raw)
}

// Write is a paid mutator transaction binding the contract method 0x7341a70e.
//
// Solidity: function write(uint256 _key, bytes raw) returns()
func (_StorageContract *StorageContractTransactorSession) Write(_key *big.Int, raw []byte) (*types.Transaction, error) {
	return _StorageContract.Contract.Write(&_StorageContract.TransactOpts, _key, raw)
}

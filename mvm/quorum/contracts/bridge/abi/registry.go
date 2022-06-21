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

// RegistryEvent is an auto generated low-level Go binding around an user-defined struct.
type RegistryEvent struct {
	Nonce     uint64
	User      common.Address
	Asset     common.Address
	Amount    *big.Int
	Extra     []byte
	Timestamp uint64
	Sig       [2]*big.Int
}

// RegistryContractMetaData contains all meta data concerning the RegistryContract contract.
var RegistryContractMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"stateMutability\":\"nonpayable\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"raw\",\"internalType\":\"bytes\"},{\"type\":\"uint128\",\"name\":\"pid\",\"internalType\":\"uint128\"}]},{\"type\":\"event\",\"name\":\"AssetCreated\",\"inputs\":[{\"type\":\"address\",\"name\":\"at\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"uint256\",\"name\":\"id\",\"internalType\":\"uint256\",\"indexed\":false}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Halted\",\"inputs\":[{\"type\":\"bool\",\"name\":\"state\",\"internalType\":\"bool\",\"indexed\":false}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Iterated\",\"inputs\":[{\"type\":\"uint256[4]\",\"name\":\"from\",\"internalType\":\"uint256[4]\",\"indexed\":false},{\"type\":\"uint256[4]\",\"name\":\"to\",\"internalType\":\"uint256[4]\",\"indexed\":false}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"MixinEvent\",\"inputs\":[{\"type\":\"tuple\",\"name\":\"evt\",\"internalType\":\"structRegistry.Event\",\"indexed\":false,\"components\":[{\"type\":\"uint64\",\"name\":\"nonce\",\"internalType\":\"uint64\"},{\"type\":\"address\",\"name\":\"user\",\"internalType\":\"address\"},{\"type\":\"address\",\"name\":\"asset\",\"internalType\":\"address\"},{\"type\":\"uint256\",\"name\":\"amount\",\"internalType\":\"uint256\"},{\"type\":\"bytes\",\"name\":\"extra\",\"internalType\":\"bytes\"},{\"type\":\"uint64\",\"name\":\"timestamp\",\"internalType\":\"uint64\"},{\"type\":\"uint256[2]\",\"name\":\"sig\",\"internalType\":\"uint256[2]\"}]}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"MixinTransaction\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"raw\",\"internalType\":\"bytes\",\"indexed\":false}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"UserCreated\",\"inputs\":[{\"type\":\"address\",\"name\":\"at\",\"internalType\":\"address\",\"indexed\":true},{\"type\":\"bytes\",\"name\":\"members\",\"internalType\":\"bytes\",\"indexed\":false}],\"anonymous\":false},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}],\"name\":\"GROUP\",\"inputs\":[{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"bool\",\"name\":\"\",\"internalType\":\"bool\"}],\"name\":\"HALTED\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint64\",\"name\":\"\",\"internalType\":\"uint64\"}],\"name\":\"INBOUND\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint64\",\"name\":\"\",\"internalType\":\"uint64\"}],\"name\":\"OUTBOUND\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint128\",\"name\":\"\",\"internalType\":\"uint128\"}],\"name\":\"PID\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}],\"name\":\"VERSION\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint128\",\"name\":\"\",\"internalType\":\"uint128\"}],\"name\":\"assets\",\"inputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}],\"name\":\"balances\",\"inputs\":[{\"type\":\"uint128\",\"name\":\"\",\"internalType\":\"uint128\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[{\"type\":\"bool\",\"name\":\"\",\"internalType\":\"bool\"}],\"name\":\"burn\",\"inputs\":[{\"type\":\"address\",\"name\":\"user\",\"internalType\":\"address\"},{\"type\":\"uint256\",\"name\":\"amount\",\"internalType\":\"uint256\"},{\"type\":\"bytes\",\"name\":\"extra\",\"internalType\":\"bytes\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[{\"type\":\"bool\",\"name\":\"\",\"internalType\":\"bool\"}],\"name\":\"claim\",\"inputs\":[{\"type\":\"address\",\"name\":\"asset\",\"internalType\":\"address\"},{\"type\":\"uint256\",\"name\":\"amount\",\"internalType\":\"uint256\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}],\"name\":\"contracts\",\"inputs\":[{\"type\":\"uint256\",\"name\":\"\",\"internalType\":\"uint256\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[],\"name\":\"halt\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"raw\",\"internalType\":\"bytes\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[],\"name\":\"iterate\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"raw\",\"internalType\":\"bytes\"}]},{\"type\":\"function\",\"stateMutability\":\"nonpayable\",\"outputs\":[{\"type\":\"bool\",\"name\":\"\",\"internalType\":\"bool\"}],\"name\":\"mixin\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"raw\",\"internalType\":\"bytes\"}]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"\",\"internalType\":\"bytes\"}],\"name\":\"users\",\"inputs\":[{\"type\":\"address\",\"name\":\"\",\"internalType\":\"address\"}]}]",
}

// RegistryContractABI is the input ABI used to generate the binding from.
// Deprecated: Use RegistryContractMetaData.ABI instead.
var RegistryContractABI = RegistryContractMetaData.ABI

// RegistryContract is an auto generated Go binding around an Ethereum contract.
type RegistryContract struct {
	RegistryContractCaller     // Read-only binding to the contract
	RegistryContractTransactor // Write-only binding to the contract
	RegistryContractFilterer   // Log filterer for contract events
}

// RegistryContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type RegistryContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RegistryContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RegistryContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegistryContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RegistryContractSession struct {
	Contract     *RegistryContract // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RegistryContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RegistryContractCallerSession struct {
	Contract *RegistryContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// RegistryContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RegistryContractTransactorSession struct {
	Contract     *RegistryContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// RegistryContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type RegistryContractRaw struct {
	Contract *RegistryContract // Generic contract binding to access the raw methods on
}

// RegistryContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RegistryContractCallerRaw struct {
	Contract *RegistryContractCaller // Generic read-only contract binding to access the raw methods on
}

// RegistryContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RegistryContractTransactorRaw struct {
	Contract *RegistryContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRegistryContract creates a new instance of RegistryContract, bound to a specific deployed contract.
func NewRegistryContract(address common.Address, backend bind.ContractBackend) (*RegistryContract, error) {
	contract, err := bindRegistryContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RegistryContract{RegistryContractCaller: RegistryContractCaller{contract: contract}, RegistryContractTransactor: RegistryContractTransactor{contract: contract}, RegistryContractFilterer: RegistryContractFilterer{contract: contract}}, nil
}

// NewRegistryContractCaller creates a new read-only instance of RegistryContract, bound to a specific deployed contract.
func NewRegistryContractCaller(address common.Address, caller bind.ContractCaller) (*RegistryContractCaller, error) {
	contract, err := bindRegistryContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryContractCaller{contract: contract}, nil
}

// NewRegistryContractTransactor creates a new write-only instance of RegistryContract, bound to a specific deployed contract.
func NewRegistryContractTransactor(address common.Address, transactor bind.ContractTransactor) (*RegistryContractTransactor, error) {
	contract, err := bindRegistryContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RegistryContractTransactor{contract: contract}, nil
}

// NewRegistryContractFilterer creates a new log filterer instance of RegistryContract, bound to a specific deployed contract.
func NewRegistryContractFilterer(address common.Address, filterer bind.ContractFilterer) (*RegistryContractFilterer, error) {
	contract, err := bindRegistryContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RegistryContractFilterer{contract: contract}, nil
}

// bindRegistryContract binds a generic wrapper to an already deployed contract.
func bindRegistryContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(RegistryContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RegistryContract *RegistryContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RegistryContract.Contract.RegistryContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RegistryContract *RegistryContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RegistryContract.Contract.RegistryContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RegistryContract *RegistryContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RegistryContract.Contract.RegistryContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RegistryContract *RegistryContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RegistryContract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RegistryContract *RegistryContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RegistryContract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RegistryContract *RegistryContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RegistryContract.Contract.contract.Transact(opts, method, params...)
}

// GROUP is a free data retrieval call binding the contract method 0x81ebf1c3.
//
// Solidity: function GROUP(uint256 ) view returns(uint256)
func (_RegistryContract *RegistryContractCaller) GROUP(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "GROUP", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GROUP is a free data retrieval call binding the contract method 0x81ebf1c3.
//
// Solidity: function GROUP(uint256 ) view returns(uint256)
func (_RegistryContract *RegistryContractSession) GROUP(arg0 *big.Int) (*big.Int, error) {
	return _RegistryContract.Contract.GROUP(&_RegistryContract.CallOpts, arg0)
}

// GROUP is a free data retrieval call binding the contract method 0x81ebf1c3.
//
// Solidity: function GROUP(uint256 ) view returns(uint256)
func (_RegistryContract *RegistryContractCallerSession) GROUP(arg0 *big.Int) (*big.Int, error) {
	return _RegistryContract.Contract.GROUP(&_RegistryContract.CallOpts, arg0)
}

// HALTED is a free data retrieval call binding the contract method 0x678d7732.
//
// Solidity: function HALTED() view returns(bool)
func (_RegistryContract *RegistryContractCaller) HALTED(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "HALTED")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HALTED is a free data retrieval call binding the contract method 0x678d7732.
//
// Solidity: function HALTED() view returns(bool)
func (_RegistryContract *RegistryContractSession) HALTED() (bool, error) {
	return _RegistryContract.Contract.HALTED(&_RegistryContract.CallOpts)
}

// HALTED is a free data retrieval call binding the contract method 0x678d7732.
//
// Solidity: function HALTED() view returns(bool)
func (_RegistryContract *RegistryContractCallerSession) HALTED() (bool, error) {
	return _RegistryContract.Contract.HALTED(&_RegistryContract.CallOpts)
}

// INBOUND is a free data retrieval call binding the contract method 0x85835923.
//
// Solidity: function INBOUND() view returns(uint64)
func (_RegistryContract *RegistryContractCaller) INBOUND(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "INBOUND")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// INBOUND is a free data retrieval call binding the contract method 0x85835923.
//
// Solidity: function INBOUND() view returns(uint64)
func (_RegistryContract *RegistryContractSession) INBOUND() (uint64, error) {
	return _RegistryContract.Contract.INBOUND(&_RegistryContract.CallOpts)
}

// INBOUND is a free data retrieval call binding the contract method 0x85835923.
//
// Solidity: function INBOUND() view returns(uint64)
func (_RegistryContract *RegistryContractCallerSession) INBOUND() (uint64, error) {
	return _RegistryContract.Contract.INBOUND(&_RegistryContract.CallOpts)
}

// OUTBOUND is a free data retrieval call binding the contract method 0x48093204.
//
// Solidity: function OUTBOUND() view returns(uint64)
func (_RegistryContract *RegistryContractCaller) OUTBOUND(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "OUTBOUND")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// OUTBOUND is a free data retrieval call binding the contract method 0x48093204.
//
// Solidity: function OUTBOUND() view returns(uint64)
func (_RegistryContract *RegistryContractSession) OUTBOUND() (uint64, error) {
	return _RegistryContract.Contract.OUTBOUND(&_RegistryContract.CallOpts)
}

// OUTBOUND is a free data retrieval call binding the contract method 0x48093204.
//
// Solidity: function OUTBOUND() view returns(uint64)
func (_RegistryContract *RegistryContractCallerSession) OUTBOUND() (uint64, error) {
	return _RegistryContract.Contract.OUTBOUND(&_RegistryContract.CallOpts)
}

// PID is a free data retrieval call binding the contract method 0x5eaec0e4.
//
// Solidity: function PID() view returns(uint128)
func (_RegistryContract *RegistryContractCaller) PID(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "PID")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PID is a free data retrieval call binding the contract method 0x5eaec0e4.
//
// Solidity: function PID() view returns(uint128)
func (_RegistryContract *RegistryContractSession) PID() (*big.Int, error) {
	return _RegistryContract.Contract.PID(&_RegistryContract.CallOpts)
}

// PID is a free data retrieval call binding the contract method 0x5eaec0e4.
//
// Solidity: function PID() view returns(uint128)
func (_RegistryContract *RegistryContractCallerSession) PID() (*big.Int, error) {
	return _RegistryContract.Contract.PID(&_RegistryContract.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_RegistryContract *RegistryContractCaller) VERSION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "VERSION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_RegistryContract *RegistryContractSession) VERSION() (*big.Int, error) {
	return _RegistryContract.Contract.VERSION(&_RegistryContract.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_RegistryContract *RegistryContractCallerSession) VERSION() (*big.Int, error) {
	return _RegistryContract.Contract.VERSION(&_RegistryContract.CallOpts)
}

// Assets is a free data retrieval call binding the contract method 0xf11b8188.
//
// Solidity: function assets(address ) view returns(uint128)
func (_RegistryContract *RegistryContractCaller) Assets(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "assets", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Assets is a free data retrieval call binding the contract method 0xf11b8188.
//
// Solidity: function assets(address ) view returns(uint128)
func (_RegistryContract *RegistryContractSession) Assets(arg0 common.Address) (*big.Int, error) {
	return _RegistryContract.Contract.Assets(&_RegistryContract.CallOpts, arg0)
}

// Assets is a free data retrieval call binding the contract method 0xf11b8188.
//
// Solidity: function assets(address ) view returns(uint128)
func (_RegistryContract *RegistryContractCallerSession) Assets(arg0 common.Address) (*big.Int, error) {
	return _RegistryContract.Contract.Assets(&_RegistryContract.CallOpts, arg0)
}

// Balances is a free data retrieval call binding the contract method 0x8d46b0c9.
//
// Solidity: function balances(uint128 ) view returns(uint256)
func (_RegistryContract *RegistryContractCaller) Balances(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "balances", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Balances is a free data retrieval call binding the contract method 0x8d46b0c9.
//
// Solidity: function balances(uint128 ) view returns(uint256)
func (_RegistryContract *RegistryContractSession) Balances(arg0 *big.Int) (*big.Int, error) {
	return _RegistryContract.Contract.Balances(&_RegistryContract.CallOpts, arg0)
}

// Balances is a free data retrieval call binding the contract method 0x8d46b0c9.
//
// Solidity: function balances(uint128 ) view returns(uint256)
func (_RegistryContract *RegistryContractCallerSession) Balances(arg0 *big.Int) (*big.Int, error) {
	return _RegistryContract.Contract.Balances(&_RegistryContract.CallOpts, arg0)
}

// Contracts is a free data retrieval call binding the contract method 0x474da79a.
//
// Solidity: function contracts(uint256 ) view returns(address)
func (_RegistryContract *RegistryContractCaller) Contracts(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "contracts", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Contracts is a free data retrieval call binding the contract method 0x474da79a.
//
// Solidity: function contracts(uint256 ) view returns(address)
func (_RegistryContract *RegistryContractSession) Contracts(arg0 *big.Int) (common.Address, error) {
	return _RegistryContract.Contract.Contracts(&_RegistryContract.CallOpts, arg0)
}

// Contracts is a free data retrieval call binding the contract method 0x474da79a.
//
// Solidity: function contracts(uint256 ) view returns(address)
func (_RegistryContract *RegistryContractCallerSession) Contracts(arg0 *big.Int) (common.Address, error) {
	return _RegistryContract.Contract.Contracts(&_RegistryContract.CallOpts, arg0)
}

// Users is a free data retrieval call binding the contract method 0xa87430ba.
//
// Solidity: function users(address ) view returns(bytes)
func (_RegistryContract *RegistryContractCaller) Users(opts *bind.CallOpts, arg0 common.Address) ([]byte, error) {
	var out []interface{}
	err := _RegistryContract.contract.Call(opts, &out, "users", arg0)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Users is a free data retrieval call binding the contract method 0xa87430ba.
//
// Solidity: function users(address ) view returns(bytes)
func (_RegistryContract *RegistryContractSession) Users(arg0 common.Address) ([]byte, error) {
	return _RegistryContract.Contract.Users(&_RegistryContract.CallOpts, arg0)
}

// Users is a free data retrieval call binding the contract method 0xa87430ba.
//
// Solidity: function users(address ) view returns(bytes)
func (_RegistryContract *RegistryContractCallerSession) Users(arg0 common.Address) ([]byte, error) {
	return _RegistryContract.Contract.Users(&_RegistryContract.CallOpts, arg0)
}

// Burn is a paid mutator transaction binding the contract method 0x44d17187.
//
// Solidity: function burn(address user, uint256 amount, bytes extra) returns(bool)
func (_RegistryContract *RegistryContractTransactor) Burn(opts *bind.TransactOpts, user common.Address, amount *big.Int, extra []byte) (*types.Transaction, error) {
	return _RegistryContract.contract.Transact(opts, "burn", user, amount, extra)
}

// Burn is a paid mutator transaction binding the contract method 0x44d17187.
//
// Solidity: function burn(address user, uint256 amount, bytes extra) returns(bool)
func (_RegistryContract *RegistryContractSession) Burn(user common.Address, amount *big.Int, extra []byte) (*types.Transaction, error) {
	return _RegistryContract.Contract.Burn(&_RegistryContract.TransactOpts, user, amount, extra)
}

// Burn is a paid mutator transaction binding the contract method 0x44d17187.
//
// Solidity: function burn(address user, uint256 amount, bytes extra) returns(bool)
func (_RegistryContract *RegistryContractTransactorSession) Burn(user common.Address, amount *big.Int, extra []byte) (*types.Transaction, error) {
	return _RegistryContract.Contract.Burn(&_RegistryContract.TransactOpts, user, amount, extra)
}

// Claim is a paid mutator transaction binding the contract method 0xaad3ec96.
//
// Solidity: function claim(address asset, uint256 amount) returns(bool)
func (_RegistryContract *RegistryContractTransactor) Claim(opts *bind.TransactOpts, asset common.Address, amount *big.Int) (*types.Transaction, error) {
	return _RegistryContract.contract.Transact(opts, "claim", asset, amount)
}

// Claim is a paid mutator transaction binding the contract method 0xaad3ec96.
//
// Solidity: function claim(address asset, uint256 amount) returns(bool)
func (_RegistryContract *RegistryContractSession) Claim(asset common.Address, amount *big.Int) (*types.Transaction, error) {
	return _RegistryContract.Contract.Claim(&_RegistryContract.TransactOpts, asset, amount)
}

// Claim is a paid mutator transaction binding the contract method 0xaad3ec96.
//
// Solidity: function claim(address asset, uint256 amount) returns(bool)
func (_RegistryContract *RegistryContractTransactorSession) Claim(asset common.Address, amount *big.Int) (*types.Transaction, error) {
	return _RegistryContract.Contract.Claim(&_RegistryContract.TransactOpts, asset, amount)
}

// Halt is a paid mutator transaction binding the contract method 0x944e7cb1.
//
// Solidity: function halt(bytes raw) returns()
func (_RegistryContract *RegistryContractTransactor) Halt(opts *bind.TransactOpts, raw []byte) (*types.Transaction, error) {
	return _RegistryContract.contract.Transact(opts, "halt", raw)
}

// Halt is a paid mutator transaction binding the contract method 0x944e7cb1.
//
// Solidity: function halt(bytes raw) returns()
func (_RegistryContract *RegistryContractSession) Halt(raw []byte) (*types.Transaction, error) {
	return _RegistryContract.Contract.Halt(&_RegistryContract.TransactOpts, raw)
}

// Halt is a paid mutator transaction binding the contract method 0x944e7cb1.
//
// Solidity: function halt(bytes raw) returns()
func (_RegistryContract *RegistryContractTransactorSession) Halt(raw []byte) (*types.Transaction, error) {
	return _RegistryContract.Contract.Halt(&_RegistryContract.TransactOpts, raw)
}

// Iterate is a paid mutator transaction binding the contract method 0xbab54626.
//
// Solidity: function iterate(bytes raw) returns()
func (_RegistryContract *RegistryContractTransactor) Iterate(opts *bind.TransactOpts, raw []byte) (*types.Transaction, error) {
	return _RegistryContract.contract.Transact(opts, "iterate", raw)
}

// Iterate is a paid mutator transaction binding the contract method 0xbab54626.
//
// Solidity: function iterate(bytes raw) returns()
func (_RegistryContract *RegistryContractSession) Iterate(raw []byte) (*types.Transaction, error) {
	return _RegistryContract.Contract.Iterate(&_RegistryContract.TransactOpts, raw)
}

// Iterate is a paid mutator transaction binding the contract method 0xbab54626.
//
// Solidity: function iterate(bytes raw) returns()
func (_RegistryContract *RegistryContractTransactorSession) Iterate(raw []byte) (*types.Transaction, error) {
	return _RegistryContract.Contract.Iterate(&_RegistryContract.TransactOpts, raw)
}

// Mixin is a paid mutator transaction binding the contract method 0x5cae8005.
//
// Solidity: function mixin(bytes raw) returns(bool)
func (_RegistryContract *RegistryContractTransactor) Mixin(opts *bind.TransactOpts, raw []byte) (*types.Transaction, error) {
	return _RegistryContract.contract.Transact(opts, "mixin", raw)
}

// Mixin is a paid mutator transaction binding the contract method 0x5cae8005.
//
// Solidity: function mixin(bytes raw) returns(bool)
func (_RegistryContract *RegistryContractSession) Mixin(raw []byte) (*types.Transaction, error) {
	return _RegistryContract.Contract.Mixin(&_RegistryContract.TransactOpts, raw)
}

// Mixin is a paid mutator transaction binding the contract method 0x5cae8005.
//
// Solidity: function mixin(bytes raw) returns(bool)
func (_RegistryContract *RegistryContractTransactorSession) Mixin(raw []byte) (*types.Transaction, error) {
	return _RegistryContract.Contract.Mixin(&_RegistryContract.TransactOpts, raw)
}

// RegistryContractAssetCreatedIterator is returned from FilterAssetCreated and is used to iterate over the raw logs and unpacked data for AssetCreated events raised by the RegistryContract contract.
type RegistryContractAssetCreatedIterator struct {
	Event *RegistryContractAssetCreated // Event containing the contract specifics and raw log

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
func (it *RegistryContractAssetCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryContractAssetCreated)
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
		it.Event = new(RegistryContractAssetCreated)
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
func (it *RegistryContractAssetCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryContractAssetCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryContractAssetCreated represents a AssetCreated event raised by the RegistryContract contract.
type RegistryContractAssetCreated struct {
	At  common.Address
	Id  *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterAssetCreated is a free log retrieval operation binding the contract event 0x20df459a0f7f1bc64a42346a9e6536111a3512be01de7a0f5327a4e13b337038.
//
// Solidity: event AssetCreated(address indexed at, uint256 id)
func (_RegistryContract *RegistryContractFilterer) FilterAssetCreated(opts *bind.FilterOpts, at []common.Address) (*RegistryContractAssetCreatedIterator, error) {

	var atRule []interface{}
	for _, atItem := range at {
		atRule = append(atRule, atItem)
	}

	logs, sub, err := _RegistryContract.contract.FilterLogs(opts, "AssetCreated", atRule)
	if err != nil {
		return nil, err
	}
	return &RegistryContractAssetCreatedIterator{contract: _RegistryContract.contract, event: "AssetCreated", logs: logs, sub: sub}, nil
}

// WatchAssetCreated is a free log subscription operation binding the contract event 0x20df459a0f7f1bc64a42346a9e6536111a3512be01de7a0f5327a4e13b337038.
//
// Solidity: event AssetCreated(address indexed at, uint256 id)
func (_RegistryContract *RegistryContractFilterer) WatchAssetCreated(opts *bind.WatchOpts, sink chan<- *RegistryContractAssetCreated, at []common.Address) (event.Subscription, error) {

	var atRule []interface{}
	for _, atItem := range at {
		atRule = append(atRule, atItem)
	}

	logs, sub, err := _RegistryContract.contract.WatchLogs(opts, "AssetCreated", atRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryContractAssetCreated)
				if err := _RegistryContract.contract.UnpackLog(event, "AssetCreated", log); err != nil {
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

// ParseAssetCreated is a log parse operation binding the contract event 0x20df459a0f7f1bc64a42346a9e6536111a3512be01de7a0f5327a4e13b337038.
//
// Solidity: event AssetCreated(address indexed at, uint256 id)
func (_RegistryContract *RegistryContractFilterer) ParseAssetCreated(log types.Log) (*RegistryContractAssetCreated, error) {
	event := new(RegistryContractAssetCreated)
	if err := _RegistryContract.contract.UnpackLog(event, "AssetCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryContractHaltedIterator is returned from FilterHalted and is used to iterate over the raw logs and unpacked data for Halted events raised by the RegistryContract contract.
type RegistryContractHaltedIterator struct {
	Event *RegistryContractHalted // Event containing the contract specifics and raw log

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
func (it *RegistryContractHaltedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryContractHalted)
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
		it.Event = new(RegistryContractHalted)
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
func (it *RegistryContractHaltedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryContractHaltedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryContractHalted represents a Halted event raised by the RegistryContract contract.
type RegistryContractHalted struct {
	State bool
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterHalted is a free log retrieval operation binding the contract event 0x92333b0b676476985757350034668cb9ee247674ac7a7479de10cd761381f733.
//
// Solidity: event Halted(bool state)
func (_RegistryContract *RegistryContractFilterer) FilterHalted(opts *bind.FilterOpts) (*RegistryContractHaltedIterator, error) {

	logs, sub, err := _RegistryContract.contract.FilterLogs(opts, "Halted")
	if err != nil {
		return nil, err
	}
	return &RegistryContractHaltedIterator{contract: _RegistryContract.contract, event: "Halted", logs: logs, sub: sub}, nil
}

// WatchHalted is a free log subscription operation binding the contract event 0x92333b0b676476985757350034668cb9ee247674ac7a7479de10cd761381f733.
//
// Solidity: event Halted(bool state)
func (_RegistryContract *RegistryContractFilterer) WatchHalted(opts *bind.WatchOpts, sink chan<- *RegistryContractHalted) (event.Subscription, error) {

	logs, sub, err := _RegistryContract.contract.WatchLogs(opts, "Halted")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryContractHalted)
				if err := _RegistryContract.contract.UnpackLog(event, "Halted", log); err != nil {
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

// ParseHalted is a log parse operation binding the contract event 0x92333b0b676476985757350034668cb9ee247674ac7a7479de10cd761381f733.
//
// Solidity: event Halted(bool state)
func (_RegistryContract *RegistryContractFilterer) ParseHalted(log types.Log) (*RegistryContractHalted, error) {
	event := new(RegistryContractHalted)
	if err := _RegistryContract.contract.UnpackLog(event, "Halted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryContractIteratedIterator is returned from FilterIterated and is used to iterate over the raw logs and unpacked data for Iterated events raised by the RegistryContract contract.
type RegistryContractIteratedIterator struct {
	Event *RegistryContractIterated // Event containing the contract specifics and raw log

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
func (it *RegistryContractIteratedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryContractIterated)
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
		it.Event = new(RegistryContractIterated)
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
func (it *RegistryContractIteratedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryContractIteratedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryContractIterated represents a Iterated event raised by the RegistryContract contract.
type RegistryContractIterated struct {
	From [4]*big.Int
	To   [4]*big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterIterated is a free log retrieval operation binding the contract event 0x20b54a4e4d72fb59d7f4da768f89618921cb2abc9d2ede08c065bb1f36c745f5.
//
// Solidity: event Iterated(uint256[4] from, uint256[4] to)
func (_RegistryContract *RegistryContractFilterer) FilterIterated(opts *bind.FilterOpts) (*RegistryContractIteratedIterator, error) {

	logs, sub, err := _RegistryContract.contract.FilterLogs(opts, "Iterated")
	if err != nil {
		return nil, err
	}
	return &RegistryContractIteratedIterator{contract: _RegistryContract.contract, event: "Iterated", logs: logs, sub: sub}, nil
}

// WatchIterated is a free log subscription operation binding the contract event 0x20b54a4e4d72fb59d7f4da768f89618921cb2abc9d2ede08c065bb1f36c745f5.
//
// Solidity: event Iterated(uint256[4] from, uint256[4] to)
func (_RegistryContract *RegistryContractFilterer) WatchIterated(opts *bind.WatchOpts, sink chan<- *RegistryContractIterated) (event.Subscription, error) {

	logs, sub, err := _RegistryContract.contract.WatchLogs(opts, "Iterated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryContractIterated)
				if err := _RegistryContract.contract.UnpackLog(event, "Iterated", log); err != nil {
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

// ParseIterated is a log parse operation binding the contract event 0x20b54a4e4d72fb59d7f4da768f89618921cb2abc9d2ede08c065bb1f36c745f5.
//
// Solidity: event Iterated(uint256[4] from, uint256[4] to)
func (_RegistryContract *RegistryContractFilterer) ParseIterated(log types.Log) (*RegistryContractIterated, error) {
	event := new(RegistryContractIterated)
	if err := _RegistryContract.contract.UnpackLog(event, "Iterated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegistryContractMixinEventIterator is returned from FilterMixinEvent and is used to iterate over the raw logs and unpacked data for MixinEvent events raised by the RegistryContract contract.
type RegistryContractMixinEventIterator struct {
	Event *RegistryContractMixinEvent // Event containing the contract specifics and raw log

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
func (it *RegistryContractMixinEventIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RegistryContractMixinEvent)
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
		it.Event = new(RegistryContractMixinEvent)
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
func (it *RegistryContractMixinEventIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RegistryContractMixinEventIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RegistryContractMixinEvent represents a MixinEvent event raised by the RegistryContract contract.
type RegistryContractMixinEvent struct {
	Evt RegistryEvent
	Raw types.Log // Blockchain specific contextual infos
}

// FilterMixinEvent is a free log retrieval operation binding the contract event 0xbf9be0caf58b62993c79cd8f1c0b53386c571be762dcafde0ed58d45fe73e621.
//
// Solidity: event MixinEvent((uint64,address,address,uint256,bytes,uint64,uint256[2]) evt)
func (_RegistryContract *RegistryContractFilterer) FilterMixinEvent(opts *bind.FilterOpts) (*RegistryContractMixinEventIterator, error) {

	logs, sub, err := _RegistryContract.contract.FilterLogs(opts, "MixinEvent")
	if err != nil {
		return nil, err
	}
	return &RegistryContractMixinEventIterator{contract: _RegistryContract.contract, event: "MixinEvent", logs: logs, sub: sub}, nil
}

// WatchMixinEvent is a free log subscription operation binding the contract event 0xbf9be0caf58b62993c79cd8f1c0b53386c571be762dcafde0ed58d45fe73e621.
//
// Solidity: event MixinEvent((uint64,address,address,uint256,bytes,uint64,uint256[2]) evt)
func (_RegistryContract *RegistryContractFilterer) WatchMixinEvent(opts *bind.WatchOpts, sink chan<- *RegistryContractMixinEvent) (event.Subscription, error) {

	logs, sub, err := _RegistryContract.contract.WatchLogs(opts, "MixinEvent")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RegistryContractMixinEvent)
				if err := _RegistryContract.contract.UnpackLog(event, "MixinEvent", log); err != nil {
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

// ParseMixinEvent is a log parse operation binding the contract event 0xbf9be0caf58b62993c79cd8f1c0b53386c571be762dcafde0ed58d45fe73e621.
//
// Solidity: event MixinEvent((uint64,address,address,uint256,bytes,uint64,uint256[2]) evt)
func (_RegistryContract *RegistryContractFilterer) ParseMixinEvent(log types.Log) (*RegistryContractMixinEvent, error) {
	event := new(RegistryContractMixinEvent)
	if err := _RegistryContract.contract.UnpackLog(event, "MixinEvent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

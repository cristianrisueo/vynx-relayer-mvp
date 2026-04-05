// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

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
	_ = abi.ConvertType
)

// VynxSettlementMetaData contains all meta data concerning the VynxSettlement contract.
var VynxSettlementMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"version\",\"type\":\"string\"},{\"name\":\"chainId\",\"type\":\"uint256\"},{\"name\":\"verifyingContract\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"settle\",\"stateMutability\":\"nonpayable\",\"inputs\":[{\"name\":\"intentId\",\"type\":\"bytes32\"},{\"name\":\"sender\",\"type\":\"address\"},{\"name\":\"tokenIn\",\"type\":\"address\"},{\"name\":\"tokenOut\",\"type\":\"address\"},{\"name\":\"amountIn\",\"type\":\"uint256\"},{\"name\":\"minAmountOut\",\"type\":\"uint256\"},{\"name\":\"deadline\",\"type\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint256\"},{\"name\":\"winningSolver\",\"type\":\"address\"},{\"name\":\"amountOut\",\"type\":\"uint256\"},{\"name\":\"relayerSig\",\"type\":\"bytes\"}],\"outputs\":[]},{\"type\":\"event\",\"name\":\"Settled\",\"anonymous\":false,\"inputs\":[{\"name\":\"intentId\",\"type\":\"bytes32\",\"indexed\":true},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true},{\"name\":\"winningSolver\",\"type\":\"address\",\"indexed\":true},{\"name\":\"amountOut\",\"type\":\"uint256\",\"indexed\":false}]}]",
}

// VynxSettlementABI is the input ABI used to generate the binding from.
// Deprecated: Use VynxSettlementMetaData.ABI instead.
var VynxSettlementABI = VynxSettlementMetaData.ABI

// VynxSettlement is an auto generated Go binding around an Ethereum contract.
type VynxSettlement struct {
	VynxSettlementCaller     // Read-only binding to the contract
	VynxSettlementTransactor // Write-only binding to the contract
	VynxSettlementFilterer   // Log filterer for contract events
}

// VynxSettlementCaller is an auto generated read-only Go binding around an Ethereum contract.
type VynxSettlementCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VynxSettlementTransactor is an auto generated write-only Go binding around an Ethereum contract.
type VynxSettlementTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VynxSettlementFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type VynxSettlementFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VynxSettlementSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type VynxSettlementSession struct {
	Contract     *VynxSettlement   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// VynxSettlementCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type VynxSettlementCallerSession struct {
	Contract *VynxSettlementCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// VynxSettlementTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type VynxSettlementTransactorSession struct {
	Contract     *VynxSettlementTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// VynxSettlementRaw is an auto generated low-level Go binding around an Ethereum contract.
type VynxSettlementRaw struct {
	Contract *VynxSettlement // Generic contract binding to access the raw methods on
}

// VynxSettlementCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type VynxSettlementCallerRaw struct {
	Contract *VynxSettlementCaller // Generic read-only contract binding to access the raw methods on
}

// VynxSettlementTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type VynxSettlementTransactorRaw struct {
	Contract *VynxSettlementTransactor // Generic write-only contract binding to access the raw methods on
}

// NewVynxSettlement creates a new instance of VynxSettlement, bound to a specific deployed contract.
func NewVynxSettlement(address common.Address, backend bind.ContractBackend) (*VynxSettlement, error) {
	contract, err := bindVynxSettlement(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &VynxSettlement{VynxSettlementCaller: VynxSettlementCaller{contract: contract}, VynxSettlementTransactor: VynxSettlementTransactor{contract: contract}, VynxSettlementFilterer: VynxSettlementFilterer{contract: contract}}, nil
}

// NewVynxSettlementCaller creates a new read-only instance of VynxSettlement, bound to a specific deployed contract.
func NewVynxSettlementCaller(address common.Address, caller bind.ContractCaller) (*VynxSettlementCaller, error) {
	contract, err := bindVynxSettlement(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &VynxSettlementCaller{contract: contract}, nil
}

// NewVynxSettlementTransactor creates a new write-only instance of VynxSettlement, bound to a specific deployed contract.
func NewVynxSettlementTransactor(address common.Address, transactor bind.ContractTransactor) (*VynxSettlementTransactor, error) {
	contract, err := bindVynxSettlement(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &VynxSettlementTransactor{contract: contract}, nil
}

// NewVynxSettlementFilterer creates a new log filterer instance of VynxSettlement, bound to a specific deployed contract.
func NewVynxSettlementFilterer(address common.Address, filterer bind.ContractFilterer) (*VynxSettlementFilterer, error) {
	contract, err := bindVynxSettlement(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &VynxSettlementFilterer{contract: contract}, nil
}

// bindVynxSettlement binds a generic wrapper to an already deployed contract.
func bindVynxSettlement(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := VynxSettlementMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VynxSettlement *VynxSettlementRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _VynxSettlement.Contract.VynxSettlementCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VynxSettlement *VynxSettlementRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VynxSettlement.Contract.VynxSettlementTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VynxSettlement *VynxSettlementRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VynxSettlement.Contract.VynxSettlementTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VynxSettlement *VynxSettlementCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _VynxSettlement.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VynxSettlement *VynxSettlementTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VynxSettlement.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VynxSettlement *VynxSettlementTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VynxSettlement.Contract.contract.Transact(opts, method, params...)
}

// Settle is a paid mutator transaction binding the contract method 0xd1c49b5b.
//
// Solidity: function settle(bytes32 intentId, address sender, address tokenIn, address tokenOut, uint256 amountIn, uint256 minAmountOut, uint256 deadline, uint256 nonce, address winningSolver, uint256 amountOut, bytes relayerSig) returns()
func (_VynxSettlement *VynxSettlementTransactor) Settle(opts *bind.TransactOpts, intentId [32]byte, sender common.Address, tokenIn common.Address, tokenOut common.Address, amountIn *big.Int, minAmountOut *big.Int, deadline *big.Int, nonce *big.Int, winningSolver common.Address, amountOut *big.Int, relayerSig []byte) (*types.Transaction, error) {
	return _VynxSettlement.contract.Transact(opts, "settle", intentId, sender, tokenIn, tokenOut, amountIn, minAmountOut, deadline, nonce, winningSolver, amountOut, relayerSig)
}

// Settle is a paid mutator transaction binding the contract method 0xd1c49b5b.
//
// Solidity: function settle(bytes32 intentId, address sender, address tokenIn, address tokenOut, uint256 amountIn, uint256 minAmountOut, uint256 deadline, uint256 nonce, address winningSolver, uint256 amountOut, bytes relayerSig) returns()
func (_VynxSettlement *VynxSettlementSession) Settle(intentId [32]byte, sender common.Address, tokenIn common.Address, tokenOut common.Address, amountIn *big.Int, minAmountOut *big.Int, deadline *big.Int, nonce *big.Int, winningSolver common.Address, amountOut *big.Int, relayerSig []byte) (*types.Transaction, error) {
	return _VynxSettlement.Contract.Settle(&_VynxSettlement.TransactOpts, intentId, sender, tokenIn, tokenOut, amountIn, minAmountOut, deadline, nonce, winningSolver, amountOut, relayerSig)
}

// Settle is a paid mutator transaction binding the contract method 0xd1c49b5b.
//
// Solidity: function settle(bytes32 intentId, address sender, address tokenIn, address tokenOut, uint256 amountIn, uint256 minAmountOut, uint256 deadline, uint256 nonce, address winningSolver, uint256 amountOut, bytes relayerSig) returns()
func (_VynxSettlement *VynxSettlementTransactorSession) Settle(intentId [32]byte, sender common.Address, tokenIn common.Address, tokenOut common.Address, amountIn *big.Int, minAmountOut *big.Int, deadline *big.Int, nonce *big.Int, winningSolver common.Address, amountOut *big.Int, relayerSig []byte) (*types.Transaction, error) {
	return _VynxSettlement.Contract.Settle(&_VynxSettlement.TransactOpts, intentId, sender, tokenIn, tokenOut, amountIn, minAmountOut, deadline, nonce, winningSolver, amountOut, relayerSig)
}

// VynxSettlementSettledIterator is returned from FilterSettled and is used to iterate over the raw logs and unpacked data for Settled events raised by the VynxSettlement contract.
type VynxSettlementSettledIterator struct {
	Event *VynxSettlementSettled // Event containing the contract specifics and raw log

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
func (it *VynxSettlementSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VynxSettlementSettled)
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
		it.Event = new(VynxSettlementSettled)
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
func (it *VynxSettlementSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VynxSettlementSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VynxSettlementSettled represents a Settled event raised by the VynxSettlement contract.
type VynxSettlementSettled struct {
	IntentId      [32]byte
	Sender        common.Address
	WinningSolver common.Address
	AmountOut     *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterSettled is a free log retrieval operation binding the contract event 0x4190759d37d5cfe7a1a70e06ec7508a05d12fd9cb76f353da1c9e028e5a48dcf.
//
// Solidity: event Settled(bytes32 indexed intentId, address indexed sender, address indexed winningSolver, uint256 amountOut)
func (_VynxSettlement *VynxSettlementFilterer) FilterSettled(opts *bind.FilterOpts, intentId [][32]byte, sender []common.Address, winningSolver []common.Address) (*VynxSettlementSettledIterator, error) {

	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var winningSolverRule []interface{}
	for _, winningSolverItem := range winningSolver {
		winningSolverRule = append(winningSolverRule, winningSolverItem)
	}

	logs, sub, err := _VynxSettlement.contract.FilterLogs(opts, "Settled", intentIdRule, senderRule, winningSolverRule)
	if err != nil {
		return nil, err
	}
	return &VynxSettlementSettledIterator{contract: _VynxSettlement.contract, event: "Settled", logs: logs, sub: sub}, nil
}

// WatchSettled is a free log subscription operation binding the contract event 0x4190759d37d5cfe7a1a70e06ec7508a05d12fd9cb76f353da1c9e028e5a48dcf.
//
// Solidity: event Settled(bytes32 indexed intentId, address indexed sender, address indexed winningSolver, uint256 amountOut)
func (_VynxSettlement *VynxSettlementFilterer) WatchSettled(opts *bind.WatchOpts, sink chan<- *VynxSettlementSettled, intentId [][32]byte, sender []common.Address, winningSolver []common.Address) (event.Subscription, error) {

	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var winningSolverRule []interface{}
	for _, winningSolverItem := range winningSolver {
		winningSolverRule = append(winningSolverRule, winningSolverItem)
	}

	logs, sub, err := _VynxSettlement.contract.WatchLogs(opts, "Settled", intentIdRule, senderRule, winningSolverRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VynxSettlementSettled)
				if err := _VynxSettlement.contract.UnpackLog(event, "Settled", log); err != nil {
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

// ParseSettled is a log parse operation binding the contract event 0x4190759d37d5cfe7a1a70e06ec7508a05d12fd9cb76f353da1c9e028e5a48dcf.
//
// Solidity: event Settled(bytes32 indexed intentId, address indexed sender, address indexed winningSolver, uint256 amountOut)
func (_VynxSettlement *VynxSettlementFilterer) ParseSettled(log types.Log) (*VynxSettlementSettled, error) {
	event := new(VynxSettlementSettled)
	if err := _VynxSettlement.contract.UnpackLog(event, "Settled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

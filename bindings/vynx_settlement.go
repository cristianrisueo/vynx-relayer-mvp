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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_relayerSigner\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"REFUND_TIMEOUT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"claimFunds\",\"inputs\":[{\"name\":\"intentId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"solver\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"relayerSignature\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"escrows\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"agent\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"token\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"lockedAt\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"isResolved\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"lockIntent\",\"inputs\":[{\"name\":\"intentId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"token\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"refundIntent\",\"inputs\":[{\"name\":\"intentId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"relayerSigner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"FundsClaimed\",\"inputs\":[{\"name\":\"intentId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"solver\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"token\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"IntentLocked\",\"inputs\":[{\"name\":\"intentId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"agent\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"token\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"IntentRefunded\",\"inputs\":[{\"name\":\"intentId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"agent\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"token\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"AddressEmptyCode\",\"inputs\":[{\"name\":\"target\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AddressInsufficientBalance\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"AmountMustBeGreaterThanZero\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureLength\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureS\",\"inputs\":[{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"EscrowAlreadyResolved\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"EscrowNotFound\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"FailedInnerCall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"IntentAlreadyExists\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ReentrancyGuardReentrantCall\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"RefundNotReadyYet\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"SafeERC20FailedOperation\",\"inputs\":[{\"name\":\"token\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
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

// REFUNDTIMEOUT is a free data retrieval call binding the contract method 0x7d57900a.
//
// Solidity: function REFUND_TIMEOUT() view returns(uint256)
func (_VynxSettlement *VynxSettlementCaller) REFUNDTIMEOUT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _VynxSettlement.contract.Call(opts, &out, "REFUND_TIMEOUT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// REFUNDTIMEOUT is a free data retrieval call binding the contract method 0x7d57900a.
//
// Solidity: function REFUND_TIMEOUT() view returns(uint256)
func (_VynxSettlement *VynxSettlementSession) REFUNDTIMEOUT() (*big.Int, error) {
	return _VynxSettlement.Contract.REFUNDTIMEOUT(&_VynxSettlement.CallOpts)
}

// REFUNDTIMEOUT is a free data retrieval call binding the contract method 0x7d57900a.
//
// Solidity: function REFUND_TIMEOUT() view returns(uint256)
func (_VynxSettlement *VynxSettlementCallerSession) REFUNDTIMEOUT() (*big.Int, error) {
	return _VynxSettlement.Contract.REFUNDTIMEOUT(&_VynxSettlement.CallOpts)
}

// Escrows is a free data retrieval call binding the contract method 0x2d83549c.
//
// Solidity: function escrows(bytes32 ) view returns(address agent, address token, uint256 amount, uint256 lockedAt, bool isResolved)
func (_VynxSettlement *VynxSettlementCaller) Escrows(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Agent      common.Address
	Token      common.Address
	Amount     *big.Int
	LockedAt   *big.Int
	IsResolved bool
}, error) {
	var out []interface{}
	err := _VynxSettlement.contract.Call(opts, &out, "escrows", arg0)

	outstruct := new(struct {
		Agent      common.Address
		Token      common.Address
		Amount     *big.Int
		LockedAt   *big.Int
		IsResolved bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Agent = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Token = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Amount = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.LockedAt = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.IsResolved = *abi.ConvertType(out[4], new(bool)).(*bool)

	return *outstruct, err

}

// Escrows is a free data retrieval call binding the contract method 0x2d83549c.
//
// Solidity: function escrows(bytes32 ) view returns(address agent, address token, uint256 amount, uint256 lockedAt, bool isResolved)
func (_VynxSettlement *VynxSettlementSession) Escrows(arg0 [32]byte) (struct {
	Agent      common.Address
	Token      common.Address
	Amount     *big.Int
	LockedAt   *big.Int
	IsResolved bool
}, error) {
	return _VynxSettlement.Contract.Escrows(&_VynxSettlement.CallOpts, arg0)
}

// Escrows is a free data retrieval call binding the contract method 0x2d83549c.
//
// Solidity: function escrows(bytes32 ) view returns(address agent, address token, uint256 amount, uint256 lockedAt, bool isResolved)
func (_VynxSettlement *VynxSettlementCallerSession) Escrows(arg0 [32]byte) (struct {
	Agent      common.Address
	Token      common.Address
	Amount     *big.Int
	LockedAt   *big.Int
	IsResolved bool
}, error) {
	return _VynxSettlement.Contract.Escrows(&_VynxSettlement.CallOpts, arg0)
}

// RelayerSigner is a free data retrieval call binding the contract method 0xa744d7ff.
//
// Solidity: function relayerSigner() view returns(address)
func (_VynxSettlement *VynxSettlementCaller) RelayerSigner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _VynxSettlement.contract.Call(opts, &out, "relayerSigner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RelayerSigner is a free data retrieval call binding the contract method 0xa744d7ff.
//
// Solidity: function relayerSigner() view returns(address)
func (_VynxSettlement *VynxSettlementSession) RelayerSigner() (common.Address, error) {
	return _VynxSettlement.Contract.RelayerSigner(&_VynxSettlement.CallOpts)
}

// RelayerSigner is a free data retrieval call binding the contract method 0xa744d7ff.
//
// Solidity: function relayerSigner() view returns(address)
func (_VynxSettlement *VynxSettlementCallerSession) RelayerSigner() (common.Address, error) {
	return _VynxSettlement.Contract.RelayerSigner(&_VynxSettlement.CallOpts)
}

// ClaimFunds is a paid mutator transaction binding the contract method 0x91313481.
//
// Solidity: function claimFunds(bytes32 intentId, address solver, bytes relayerSignature) returns()
func (_VynxSettlement *VynxSettlementTransactor) ClaimFunds(opts *bind.TransactOpts, intentId [32]byte, solver common.Address, relayerSignature []byte) (*types.Transaction, error) {
	return _VynxSettlement.contract.Transact(opts, "claimFunds", intentId, solver, relayerSignature)
}

// ClaimFunds is a paid mutator transaction binding the contract method 0x91313481.
//
// Solidity: function claimFunds(bytes32 intentId, address solver, bytes relayerSignature) returns()
func (_VynxSettlement *VynxSettlementSession) ClaimFunds(intentId [32]byte, solver common.Address, relayerSignature []byte) (*types.Transaction, error) {
	return _VynxSettlement.Contract.ClaimFunds(&_VynxSettlement.TransactOpts, intentId, solver, relayerSignature)
}

// ClaimFunds is a paid mutator transaction binding the contract method 0x91313481.
//
// Solidity: function claimFunds(bytes32 intentId, address solver, bytes relayerSignature) returns()
func (_VynxSettlement *VynxSettlementTransactorSession) ClaimFunds(intentId [32]byte, solver common.Address, relayerSignature []byte) (*types.Transaction, error) {
	return _VynxSettlement.Contract.ClaimFunds(&_VynxSettlement.TransactOpts, intentId, solver, relayerSignature)
}

// LockIntent is a paid mutator transaction binding the contract method 0x47643164.
//
// Solidity: function lockIntent(bytes32 intentId, address token, uint256 amount) returns()
func (_VynxSettlement *VynxSettlementTransactor) LockIntent(opts *bind.TransactOpts, intentId [32]byte, token common.Address, amount *big.Int) (*types.Transaction, error) {
	return _VynxSettlement.contract.Transact(opts, "lockIntent", intentId, token, amount)
}

// LockIntent is a paid mutator transaction binding the contract method 0x47643164.
//
// Solidity: function lockIntent(bytes32 intentId, address token, uint256 amount) returns()
func (_VynxSettlement *VynxSettlementSession) LockIntent(intentId [32]byte, token common.Address, amount *big.Int) (*types.Transaction, error) {
	return _VynxSettlement.Contract.LockIntent(&_VynxSettlement.TransactOpts, intentId, token, amount)
}

// LockIntent is a paid mutator transaction binding the contract method 0x47643164.
//
// Solidity: function lockIntent(bytes32 intentId, address token, uint256 amount) returns()
func (_VynxSettlement *VynxSettlementTransactorSession) LockIntent(intentId [32]byte, token common.Address, amount *big.Int) (*types.Transaction, error) {
	return _VynxSettlement.Contract.LockIntent(&_VynxSettlement.TransactOpts, intentId, token, amount)
}

// RefundIntent is a paid mutator transaction binding the contract method 0x4ca18dbf.
//
// Solidity: function refundIntent(bytes32 intentId) returns()
func (_VynxSettlement *VynxSettlementTransactor) RefundIntent(opts *bind.TransactOpts, intentId [32]byte) (*types.Transaction, error) {
	return _VynxSettlement.contract.Transact(opts, "refundIntent", intentId)
}

// RefundIntent is a paid mutator transaction binding the contract method 0x4ca18dbf.
//
// Solidity: function refundIntent(bytes32 intentId) returns()
func (_VynxSettlement *VynxSettlementSession) RefundIntent(intentId [32]byte) (*types.Transaction, error) {
	return _VynxSettlement.Contract.RefundIntent(&_VynxSettlement.TransactOpts, intentId)
}

// RefundIntent is a paid mutator transaction binding the contract method 0x4ca18dbf.
//
// Solidity: function refundIntent(bytes32 intentId) returns()
func (_VynxSettlement *VynxSettlementTransactorSession) RefundIntent(intentId [32]byte) (*types.Transaction, error) {
	return _VynxSettlement.Contract.RefundIntent(&_VynxSettlement.TransactOpts, intentId)
}

// VynxSettlementFundsClaimedIterator is returned from FilterFundsClaimed and is used to iterate over the raw logs and unpacked data for FundsClaimed events raised by the VynxSettlement contract.
type VynxSettlementFundsClaimedIterator struct {
	Event *VynxSettlementFundsClaimed // Event containing the contract specifics and raw log

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
func (it *VynxSettlementFundsClaimedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VynxSettlementFundsClaimed)
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
		it.Event = new(VynxSettlementFundsClaimed)
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
func (it *VynxSettlementFundsClaimedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VynxSettlementFundsClaimedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VynxSettlementFundsClaimed represents a FundsClaimed event raised by the VynxSettlement contract.
type VynxSettlementFundsClaimed struct {
	IntentId [32]byte
	Solver   common.Address
	Token    common.Address
	Amount   *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterFundsClaimed is a free log retrieval operation binding the contract event 0x96f0dd91e3ee7218a125c7be1b53194f1e2c83a5708e1b9eb267b8e843db1459.
//
// Solidity: event FundsClaimed(bytes32 indexed intentId, address indexed solver, address token, uint256 amount)
func (_VynxSettlement *VynxSettlementFilterer) FilterFundsClaimed(opts *bind.FilterOpts, intentId [][32]byte, solver []common.Address) (*VynxSettlementFundsClaimedIterator, error) {

	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var solverRule []interface{}
	for _, solverItem := range solver {
		solverRule = append(solverRule, solverItem)
	}

	logs, sub, err := _VynxSettlement.contract.FilterLogs(opts, "FundsClaimed", intentIdRule, solverRule)
	if err != nil {
		return nil, err
	}
	return &VynxSettlementFundsClaimedIterator{contract: _VynxSettlement.contract, event: "FundsClaimed", logs: logs, sub: sub}, nil
}

// WatchFundsClaimed is a free log subscription operation binding the contract event 0x96f0dd91e3ee7218a125c7be1b53194f1e2c83a5708e1b9eb267b8e843db1459.
//
// Solidity: event FundsClaimed(bytes32 indexed intentId, address indexed solver, address token, uint256 amount)
func (_VynxSettlement *VynxSettlementFilterer) WatchFundsClaimed(opts *bind.WatchOpts, sink chan<- *VynxSettlementFundsClaimed, intentId [][32]byte, solver []common.Address) (event.Subscription, error) {

	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var solverRule []interface{}
	for _, solverItem := range solver {
		solverRule = append(solverRule, solverItem)
	}

	logs, sub, err := _VynxSettlement.contract.WatchLogs(opts, "FundsClaimed", intentIdRule, solverRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VynxSettlementFundsClaimed)
				if err := _VynxSettlement.contract.UnpackLog(event, "FundsClaimed", log); err != nil {
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

// ParseFundsClaimed is a log parse operation binding the contract event 0x96f0dd91e3ee7218a125c7be1b53194f1e2c83a5708e1b9eb267b8e843db1459.
//
// Solidity: event FundsClaimed(bytes32 indexed intentId, address indexed solver, address token, uint256 amount)
func (_VynxSettlement *VynxSettlementFilterer) ParseFundsClaimed(log types.Log) (*VynxSettlementFundsClaimed, error) {
	event := new(VynxSettlementFundsClaimed)
	if err := _VynxSettlement.contract.UnpackLog(event, "FundsClaimed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VynxSettlementIntentLockedIterator is returned from FilterIntentLocked and is used to iterate over the raw logs and unpacked data for IntentLocked events raised by the VynxSettlement contract.
type VynxSettlementIntentLockedIterator struct {
	Event *VynxSettlementIntentLocked // Event containing the contract specifics and raw log

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
func (it *VynxSettlementIntentLockedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VynxSettlementIntentLocked)
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
		it.Event = new(VynxSettlementIntentLocked)
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
func (it *VynxSettlementIntentLockedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VynxSettlementIntentLockedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VynxSettlementIntentLocked represents a IntentLocked event raised by the VynxSettlement contract.
type VynxSettlementIntentLocked struct {
	IntentId [32]byte
	Agent    common.Address
	Token    common.Address
	Amount   *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterIntentLocked is a free log retrieval operation binding the contract event 0xec5484ac216447df716d3bc6585af602671a859f40626fe755e33992e99c6e8c.
//
// Solidity: event IntentLocked(bytes32 indexed intentId, address indexed agent, address indexed token, uint256 amount)
func (_VynxSettlement *VynxSettlementFilterer) FilterIntentLocked(opts *bind.FilterOpts, intentId [][32]byte, agent []common.Address, token []common.Address) (*VynxSettlementIntentLockedIterator, error) {

	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var agentRule []interface{}
	for _, agentItem := range agent {
		agentRule = append(agentRule, agentItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _VynxSettlement.contract.FilterLogs(opts, "IntentLocked", intentIdRule, agentRule, tokenRule)
	if err != nil {
		return nil, err
	}
	return &VynxSettlementIntentLockedIterator{contract: _VynxSettlement.contract, event: "IntentLocked", logs: logs, sub: sub}, nil
}

// WatchIntentLocked is a free log subscription operation binding the contract event 0xec5484ac216447df716d3bc6585af602671a859f40626fe755e33992e99c6e8c.
//
// Solidity: event IntentLocked(bytes32 indexed intentId, address indexed agent, address indexed token, uint256 amount)
func (_VynxSettlement *VynxSettlementFilterer) WatchIntentLocked(opts *bind.WatchOpts, sink chan<- *VynxSettlementIntentLocked, intentId [][32]byte, agent []common.Address, token []common.Address) (event.Subscription, error) {

	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var agentRule []interface{}
	for _, agentItem := range agent {
		agentRule = append(agentRule, agentItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _VynxSettlement.contract.WatchLogs(opts, "IntentLocked", intentIdRule, agentRule, tokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VynxSettlementIntentLocked)
				if err := _VynxSettlement.contract.UnpackLog(event, "IntentLocked", log); err != nil {
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

// ParseIntentLocked is a log parse operation binding the contract event 0xec5484ac216447df716d3bc6585af602671a859f40626fe755e33992e99c6e8c.
//
// Solidity: event IntentLocked(bytes32 indexed intentId, address indexed agent, address indexed token, uint256 amount)
func (_VynxSettlement *VynxSettlementFilterer) ParseIntentLocked(log types.Log) (*VynxSettlementIntentLocked, error) {
	event := new(VynxSettlementIntentLocked)
	if err := _VynxSettlement.contract.UnpackLog(event, "IntentLocked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VynxSettlementIntentRefundedIterator is returned from FilterIntentRefunded and is used to iterate over the raw logs and unpacked data for IntentRefunded events raised by the VynxSettlement contract.
type VynxSettlementIntentRefundedIterator struct {
	Event *VynxSettlementIntentRefunded // Event containing the contract specifics and raw log

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
func (it *VynxSettlementIntentRefundedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VynxSettlementIntentRefunded)
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
		it.Event = new(VynxSettlementIntentRefunded)
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
func (it *VynxSettlementIntentRefundedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VynxSettlementIntentRefundedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VynxSettlementIntentRefunded represents a IntentRefunded event raised by the VynxSettlement contract.
type VynxSettlementIntentRefunded struct {
	IntentId [32]byte
	Agent    common.Address
	Token    common.Address
	Amount   *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterIntentRefunded is a free log retrieval operation binding the contract event 0xd1e3d51e5e30d3df29d43e15436a70c5c1540c3ae08b4cc95bbe4d7b3da724c6.
//
// Solidity: event IntentRefunded(bytes32 indexed intentId, address indexed agent, address token, uint256 amount)
func (_VynxSettlement *VynxSettlementFilterer) FilterIntentRefunded(opts *bind.FilterOpts, intentId [][32]byte, agent []common.Address) (*VynxSettlementIntentRefundedIterator, error) {

	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var agentRule []interface{}
	for _, agentItem := range agent {
		agentRule = append(agentRule, agentItem)
	}

	logs, sub, err := _VynxSettlement.contract.FilterLogs(opts, "IntentRefunded", intentIdRule, agentRule)
	if err != nil {
		return nil, err
	}
	return &VynxSettlementIntentRefundedIterator{contract: _VynxSettlement.contract, event: "IntentRefunded", logs: logs, sub: sub}, nil
}

// WatchIntentRefunded is a free log subscription operation binding the contract event 0xd1e3d51e5e30d3df29d43e15436a70c5c1540c3ae08b4cc95bbe4d7b3da724c6.
//
// Solidity: event IntentRefunded(bytes32 indexed intentId, address indexed agent, address token, uint256 amount)
func (_VynxSettlement *VynxSettlementFilterer) WatchIntentRefunded(opts *bind.WatchOpts, sink chan<- *VynxSettlementIntentRefunded, intentId [][32]byte, agent []common.Address) (event.Subscription, error) {

	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var agentRule []interface{}
	for _, agentItem := range agent {
		agentRule = append(agentRule, agentItem)
	}

	logs, sub, err := _VynxSettlement.contract.WatchLogs(opts, "IntentRefunded", intentIdRule, agentRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VynxSettlementIntentRefunded)
				if err := _VynxSettlement.contract.UnpackLog(event, "IntentRefunded", log); err != nil {
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

// ParseIntentRefunded is a log parse operation binding the contract event 0xd1e3d51e5e30d3df29d43e15436a70c5c1540c3ae08b4cc95bbe4d7b3da724c6.
//
// Solidity: event IntentRefunded(bytes32 indexed intentId, address indexed agent, address token, uint256 amount)
func (_VynxSettlement *VynxSettlementFilterer) ParseIntentRefunded(log types.Log) (*VynxSettlementIntentRefunded, error) {
	event := new(VynxSettlementIntentRefunded)
	if err := _VynxSettlement.contract.UnpackLog(event, "IntentRefunded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

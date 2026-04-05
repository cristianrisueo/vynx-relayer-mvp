package txmanager

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// gasPriceOracleAddr is the canonical OP Stack GasPriceOracle precompile address.
// It is identical on Base mainnet (8453), Base Sepolia (84532), and any standard
// OP Stack chain. The contract exposes getL1Fee(bytes) which returns the L1 Data
// Fee that the sequencer will charge for a given transaction payload.
const gasPriceOracleAddr = "0x420000000000000000000000000000000000000F"

// gasOverheadFactor is the safety multiplier applied to raw gas estimates.
// A 10% buffer prevents "out of gas" failures caused by minor execution variance
// between the estimation block and the actual inclusion block.
const gasOverheadFactor = 1.10

// gasPriceOracleABIJSON is the minimal ABI fragment for getL1Fee only.
// Full ABI is not needed — pulling in a heavyweight dependency for a single
// view call is an anti-pattern on the HFT hot path.
const gasPriceOracleABIJSON = `[{
	"type": "function",
	"name": "getL1Fee",
	"inputs":  [{"name": "_data", "type": "bytes"}],
	"outputs": [{"name": "",     "type": "uint256"}],
	"stateMutability": "view"
}]`

// OPStackClient wraps go-ethereum's ethclient with OP Stack–aware gas estimation.
//
// On Base L2 (and all OP Stack chains), every transaction incurs two fees:
//   - L2 Execution Fee: standard EVM gas × (baseFee + priorityFee). Estimated
//     with EstimateGas and inflated by gasOverheadFactor.
//   - L1 Data Fee: charged by the sequencer based on the compressed calldata size.
//     This fee is NOT set in the transaction itself — the sequencer deducts it from
//     the sender's balance at inclusion time. EstimateL1DataFee provides the
//     pre-flight value for monitoring and sufficiency checks.
//
// Callers should verify that the relayer wallet balance covers both components
// before submitting high-value settlement transactions.
type OPStackClient struct {
	client     *ethclient.Client
	oracleAddr common.Address
	oracleABI  abi.ABI
}

// NewOPStackClient constructs an OPStackClient backed by the provided ethclient.
func NewOPStackClient(client *ethclient.Client) (*OPStackClient, error) {
	parsed, err := abi.JSON(strings.NewReader(gasPriceOracleABIJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse GasPriceOracle ABI: %w", err)
	}
	return &OPStackClient{
		client:     client,
		oracleAddr: common.HexToAddress(gasPriceOracleAddr),
		oracleABI:  parsed,
	}, nil
}

// SuggestGasTipCap returns the current recommended EIP-1559 priority fee from
// the L2 sequencer. Returns an error if the RPC call fails.
func (c *OPStackClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	tip, err := c.client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas tip cap: %w", err)
	}
	return tip, nil
}

// EstimateL1DataFee calls the OP Stack GasPriceOracle precompile to compute the
// L1 Data Fee that the sequencer will charge for rawTxData. The fee is expressed
// in wei and is informational — it is deducted automatically by the sequencer and
// does not appear as a field inside the transaction object.
//
// Use this before submission to verify the relayer wallet balance is sufficient.
// rawTxData should be the RLP-encoded signed transaction bytes.
func (c *OPStackClient) EstimateL1DataFee(ctx context.Context, rawTxData []byte) (*big.Int, error) {
	callData, err := c.oracleABI.Pack("getL1Fee", rawTxData)
	if err != nil {
		return nil, fmt.Errorf("failed to pack getL1Fee call: %w", err)
	}

	result, err := c.client.CallContract(ctx, ethereum.CallMsg{
		To:   &c.oracleAddr,
		Data: callData,
	}, nil) // nil = latest block
	if err != nil {
		return nil, fmt.Errorf("failed to call GasPriceOracle.getL1Fee: %w", err)
	}

	unpacked, err := c.oracleABI.Methods["getL1Fee"].Outputs.Unpack(result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack getL1Fee result: %w", err)
	}
	if len(unpacked) == 0 {
		return nil, fmt.Errorf("failed to unpack getL1Fee: empty result")
	}

	fee, ok := unpacked[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected type from getL1Fee: want *big.Int")
	}
	return new(big.Int).Set(fee), nil
}

// EstimateGasWithBuffer estimates the L2 execution gas for msg and applies a 10%
// safety buffer to the raw estimate. The buffered value should be used as the
// GasLimit in bind.TransactOpts to prevent "out of gas" failures under load.
func (c *OPStackClient) EstimateGasWithBuffer(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	raw, err := c.client.EstimateGas(ctx, msg)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas: %w", err)
	}
	return uint64(float64(raw) * gasOverheadFactor), nil
}

// Client returns the underlying ethclient for operations not covered by OPStackClient
// (e.g. block queries, event subscriptions, raw transaction broadcasting).
func (c *OPStackClient) Client() *ethclient.Client {
	return c.client
}

// Command simulate is a live end-to-end integration driver for the VynX Relayer.
// It runs the full escrow settlement cycle against a REAL VynxSettlement contract
// deployed on Anvil — no bytecode injection or contract mocking.
//
// Flow:
//  1. Deploy a MockToken ERC-20 on Anvil
//  2. Mint tokens to the user (agent)
//  3. Agent approves VynxSettlement for token spend
//  4. Agent calls lockIntent on-chain (escrow deposit)
//  5. Submit intent to relayer via HTTP POST
//  6. Connect WebSocket as solver, listen for broadcast, send bid
//  7. Wait for relayer to call claimFunds on-chain (relayer signature verification)
//  8. Verify the Settled/FundsClaimed event and token transfer
//
// Prerequisites:
//
//	anvil --chain-id 31337                                                       (terminal 1)
//	make deploy-anvil   (in vynx-settlement-mvp, deploys VynxSettlement)
//	SETTLEMENT_CONTRACT_ADDRESS=0x... make build && ./bin/relayer                 (terminal 2)
//	make simulate                                                                (terminal 3)
package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/websocket"

	"github.com/cristianrisueo/vynx-relayer-mvp/bindings"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/core"
	"github.com/cristianrisueo/vynx-relayer-mvp/internal/signer"
)

// ── Defaults ──────────────────────────────────────────────────────────────────

const (
	defaultRelayerHTTP    = "http://localhost:8080"
	defaultRelayerWS      = "ws://localhost:8080"
	defaultRPCURL         = "http://127.0.0.1:8545"
	defaultChainID        = uint64(31337)
	defaultSettlementAddr = "0x5FbDB2315678afecb367f032d93F642f64180aa3"

	// Token amounts.
	lockAmount = "1000000000000000000" // 1 token (18 decimals)
	bidAmount  = "950000000000000000"  // 0.95 token
	bidGas     = "1000000000"          // 1 gwei

	// How long to wait after bid submission for auction close + on-chain settlement.
	settlementWait    = 8 * time.Second
	simulationTimeout = 60 * time.Second
)

// mockTokenBytecode is the compiled creation bytecode of a minimal ERC-20 with
// mint(), approve(), transfer(), transferFrom(), and balanceOf().
// Compiled with Solc 0.8.33 via Foundry, optimizer enabled (200 runs).
const mockTokenBytecode = "608060405234610106576100135f5461010a565b601f81116100b4575b507f4d6f636b546f6b656e00000000000000000000000000000000000000000000125f5560015461004c9061010a565b601f811161007f575b6008634d4f434b60e01b016001556002805460ff191660121790556040516106f990816101438239f35b60048111156100555760015f52601f60205f20910160051c5f5b8181106100a7575050610055565b5f83820155600101610099565b600981111561001c575f8080527f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56391601f0160051c905b8181106100f957505061001c565b5f838201556001016100eb565b5f80fd5b90600182811c92168015610138575b602083101461012457565b634e487b7160e01b5f52602260045260245ffd5b91607f169161011956fe60806040526004361015610011575f80fd5b5f3560e01c806306fdde0314610532578063095ea7b3146104b957806318160ddd1461049c57806323b872dd14610372578063313ce5671461035257806340c10f19146102d857806370a08231146102a057806395d89b4114610182578063a9059cbb146100db5763dd62ed3e14610087575f80fd5b346100d75760403660031901126100d7576100a061062e565b6100a8610644565b6001600160a01b039182165f908152600560209081526040808320949093168252928352819020549051908152f35b5f80fd5b346100d75760403660031901126100d7576100f461062e565b60243590335f5260046020526101108260405f2054101561065a565b335f52600460205260405f20610127838254610695565b905560018060a01b031690815f52600460205260405f206101498282546106b6565b90556040519081527fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef60203392a3602060405160018152f35b346100d7575f3660031901126100d7576040515f6001548060011c90600181168015610296575b602083108114610282578285529081156102665750600114610210575b50819003601f01601f191681019067ffffffffffffffff8211818310176101fc57604082905281906101f89082610604565b0390f35b634e487b7160e01b5f52604160045260245ffd5b60015f9081529091507fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf65b828210610250575060209150820101826101c6565b600181602092548385880101520191019061023b565b90506020925060ff191682840152151560051b820101826101c6565b634e487b7160e01b5f52602260045260245ffd5b91607f16916101a9565b346100d75760203660031901126100d7576001600160a01b036102c161062e565b165f526004602052602060405f2054604051908152f35b346100d75760403660031901126100d7576102f161062e565b5f7fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef602060243593610325856003546106b6565b60035560018060a01b03169384845260048252604084206103478282546106b6565b9055604051908152a3005b346100d7575f3660031901126100d757602060ff60025416604051908152f35b346100d75760603660031901126100d75761038b61062e565b610393610644565b6001600160a01b039091165f8181526005602090815260408083203384529091529020546044359290831161046b5760207fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef91835f52600482526103fd8560405f2054101561065a565b5f84815260058352604080822033835284529020805461041e908790610695565b9055835f526004825260405f20610436868254610695565b905560018060a01b031693845f526004825260405f206104578282546106b6565b9055604051908152a3602060405160018152f35b60405162461bcd60e51b8152602060048201526009602482015268616c6c6f77616e636560b81b6044820152606490fd5b346100d7575f3660031901126100d7576020600354604051908152f35b346100d75760403660031901126100d7576104d261062e565b335f8181526005602090815260408083206001600160a01b03909516808452948252918290206024359081905591519182527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591a3602060405160018152f35b346100d7575f3660031901126100d7576040515f5f548060011c906001811680156105fa575b6020831081146102825782855290811561026657506001146105a65750819003601f01601f191681019067ffffffffffffffff8211818310176101fc57604082905281906101f89082610604565b5f8080529091507f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e5635b8282106105e4575060209150820101826101c6565b60018160209254838588010152019101906105cf565b91607f1691610558565b602060409281835280519182918282860152018484015e5f828201840152601f01601f1916010190565b600435906001600160a01b03821682036100d757565b602435906001600160a01b03821682036100d757565b1561066157565b60405162461bcd60e51b815260206004820152600c60248201526b1a5b9cdd59999a58da595b9d60a21b6044820152606490fd5b919082039182116106a257565b634e487b7160e01b5f52601160045260245ffd5b919082018092116106a25756fea2646970667358221220f599811ca9099746a01a51fedfeb8b97deeb76487161b1ee3d6a02b59442a97c64736f6c63430008210033"

// ── JSON wire types ───────────────────────────────────────────────────────────

type intentHTTPRequest struct {
	ID           string `json:"id"`
	Sender       string `json:"sender"`
	TokenIn      string `json:"token_in"`
	TokenOut     string `json:"token_out"`
	AmountIn     string `json:"amount_in"`
	MinAmountOut string `json:"min_amount_out"`
	Deadline     int64  `json:"deadline"`
	Nonce        uint64 `json:"nonce"`
	Signature    string `json:"signature"`
}

type wsBroadcast struct {
	Type   string          `json:"type"`
	Intent json.RawMessage `json:"intent,omitempty"`
}

type wsIntentPayload struct {
	ID string `json:"id"`
}

type wsBidRequest struct {
	IntentID  string `json:"intent_id"`
	Solver    string `json:"solver"`
	AmountOut string `json:"amount_out"`
	GasPrice  string `json:"gas_price"`
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	relayerHTTP := envOrDefault("RELAYER_HTTP", defaultRelayerHTTP)
	relayerWS := envOrDefault("RELAYER_WS", defaultRelayerWS)
	rpcURL := envOrDefault("BASE_RPC_URL", defaultRPCURL)
	settlementAddr := common.HexToAddress(envOrDefault("SETTLEMENT_CONTRACT_ADDRESS", defaultSettlementAddr))
	chainID := big.NewInt(int64(defaultChainID))

	ctx, cancel := context.WithTimeout(context.Background(), simulationTimeout)
	defer cancel()

	// ── 1. Connect to Anvil ────────────────────────────────────────────────
	ethCl, err := ethclient.DialContext(ctx, rpcURL)
	must(err, "dial Anvil RPC")
	defer ethCl.Close()

	// ── 2. Ephemeral keys ──────────────────────────────────────────────────
	userKey, err := crypto.GenerateKey()
	must(err, "generate user key")
	solverKey, err := crypto.GenerateKey()
	must(err, "generate solver key")

	// Use Anvil account #0 as the funder (10k ETH, can send gas + deploy tokens).
	funderKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	must(err, "parse funder key")

	userAddr := crypto.PubkeyToAddress(userKey.PublicKey)
	solverAddr := crypto.PubkeyToAddress(solverKey.PublicKey)
	_ = funderKey.PublicKey // funder used only for signing, address derived inline

	printBanner(userAddr, solverAddr, settlementAddr, relayerHTTP)

	// ── 3. Fund the ephemeral user with ETH (for gas) ──────────────────────
	logf("[1] Funding ephemeral user %s with ETH for gas ...", userAddr.Hex()[:10]+"...")
	sendETH(ctx, ethCl, funderKey, userAddr, chainID, big.NewInt(1e18)) // 1 ETH for gas

	// ── 4. Deploy MockToken ERC-20 ─────────────────────────────────────────
	logf("[2] Deploying MockToken ERC-20 ...")
	tokenAddr := deployMockToken(ctx, ethCl, funderKey, chainID)
	logf("    MockToken deployed at: %s", tokenAddr.Hex())

	// ── 5. Mint tokens to user + approve VynxSettlement ────────────────────
	logf("[3] Minting 1 token to user and approving settlement contract ...")
	amount, _ := new(big.Int).SetString(lockAmount, 10)
	callMint(ctx, ethCl, funderKey, tokenAddr, userAddr, amount, chainID)
	callApprove(ctx, ethCl, userKey, tokenAddr, settlementAddr, amount, chainID)

	// ── 6. Agent calls lockIntent on-chain ─────────────────────────────────
	intentID := fmt.Sprintf("sim-%d", time.Now().UnixNano())
	var intentIDBytes [32]byte
	copy(intentIDBytes[:], []byte(intentID))

	logf("[4] Agent calling lockIntent on VynxSettlement ...")
	callLockIntent(ctx, ethCl, userKey, settlementAddr, intentIDBytes, tokenAddr, amount, chainID)
	logf("    Escrow locked: id=%s token=%s amount=%s", intentID, tokenAddr.Hex()[:10]+"...", lockAmount)

	// ── 7. Connect WebSocket (register as solver) ──────────────────────────
	logf("[5] Connecting WebSocket solver channel → %s/v1/ws/solvers", relayerWS)
	wsConn, _, err := websocket.DefaultDialer.DialContext( //nolint:gosec
		ctx, relayerWS+"/v1/ws/solvers", nil,
	)
	if err != nil {
		fatalf("WebSocket dial failed: %v\n\n    Ensure the relayer is running: make build && ./bin/relayer", err)
	}
	defer func() { _ = wsConn.Close() }()
	logf("    Connected.")

	// ── 8. Build Intent for relayer HTTP ───────────────────────────────────
	deadline := time.Now().Add(5 * time.Minute)
	const nonce = uint64(1)

	minAmt, _ := new(big.Int).SetString("900000000000000000", 10)

	intent := &core.Intent{
		ID:           core.IntentID(intentID),
		Sender:       userAddr,
		TokenIn:      tokenAddr,
		TokenOut:     tokenAddr, // same token for simplicity in this simulation
		AmountIn:     amount,
		MinAmountOut: minAmt,
		Deadline:     deadline,
		Nonce:        nonce,
	}

	// EIP-712 sign the intent (user signature — for forward-compat, not verified by contract).
	domain := signer.Domain{
		Name:              "VynX",
		Version:           "1",
		ChainID:           defaultChainID,
		VerifyingContract: settlementAddr,
	}
	digest, err := signer.HashIntent(domain, intent)
	must(err, "compute EIP-712 digest")
	sig, err := crypto.Sign(digest[:], userKey)
	must(err, "sign EIP-712 digest")
	sigHex := "0x" + hex.EncodeToString(sig)

	// ── 9. WS goroutine — listen for broadcast, reply with bid ─────────────
	bidSent := make(chan struct{})
	go func() {
		defer close(bidSent)
		for {
			_, raw, readErr := wsConn.ReadMessage()
			if readErr != nil {
				return
			}
			var envelope wsBroadcast
			if jsonErr := json.Unmarshal(raw, &envelope); jsonErr != nil {
				logf("[WS] unparseable message: %v", jsonErr)
				continue
			}
			if envelope.Type != "new_intent" {
				continue
			}
			var payload wsIntentPayload
			if jsonErr := json.Unmarshal(envelope.Intent, &payload); jsonErr != nil {
				logf("[WS] unparseable intent payload: %v", jsonErr)
				continue
			}
			logf("[7] Intent broadcast received (id: %s)", payload.ID)

			bid := wsBidRequest{
				IntentID:  payload.ID,
				Solver:    solverAddr.Hex(),
				AmountOut: bidAmount,
				GasPrice:  bidGas,
			}
			bidBytes, marshalErr := json.Marshal(bid)
			if marshalErr != nil {
				logf("[WS] bid marshal error: %v", marshalErr)
				return
			}
			if writeErr := wsConn.WriteMessage(websocket.TextMessage, bidBytes); writeErr != nil {
				logf("[WS] bid send error: %v", writeErr)
				return
			}
			logf("[8] Bid submitted: solver=%s amount_out=%s",
				solverAddr.Hex()[:10]+"...", bidAmount)
			return
		}
	}()

	// ── 10. POST Intent to relayer ─────────────────────────────────────────
	httpBody := intentHTTPRequest{
		ID:           intentID,
		Sender:       userAddr.Hex(),
		TokenIn:      tokenAddr.Hex(),
		TokenOut:     tokenAddr.Hex(),
		AmountIn:     lockAmount,
		MinAmountOut: "900000000000000000",
		Deadline:     deadline.Unix(),
		Nonce:        nonce,
		Signature:    sigHex,
	}

	bodyBytes, err := json.Marshal(httpBody)
	must(err, "marshal intent HTTP request")

	logf("[6] POST %s/v1/intent ...", relayerHTTP)

	postReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		relayerHTTP+"/v1/intent", //nolint:gosec
		bytes.NewReader(bodyBytes),
	)
	must(err, "build HTTP request")
	postReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(postReq)
	if err != nil {
		fatalf("POST /v1/intent failed: %v\n\n    Is the relayer running?", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var respBody map[string]string
	if jsonErr := json.NewDecoder(resp.Body).Decode(&respBody); jsonErr != nil {
		fatalf("decode POST response: %v", jsonErr)
	}
	logf("    → %d %s  (intent_id: %s, status: %s)",
		resp.StatusCode, http.StatusText(resp.StatusCode),
		respBody["intent_id"], respBody["status"])

	if resp.StatusCode != http.StatusAccepted {
		fatalf("relayer rejected intent (status %d) — check relayer logs", resp.StatusCode)
	}

	// ── 11. Wait for bid, then settlement cycle ────────────────────────────
	select {
	case <-bidSent:
	case <-time.After(10 * time.Second):
		fatalf("timeout waiting for intent broadcast — check relayer logs")
	}

	logf("[9] Waiting %s for auction close + on-chain claimFunds ...", settlementWait)
	time.Sleep(settlementWait)

	// ── 12. Verify on-chain state ──────────────────────────────────────────
	logf("[10] Verifying on-chain escrow state ...")
	settlement, err := bindings.NewVynxSettlement(settlementAddr, ethCl)
	must(err, "bind VynxSettlement for verification")

	escrow, err := settlement.Escrows(&bind.CallOpts{Context: ctx}, intentIDBytes)
	must(err, "read escrow state")

	if escrow.IsResolved {
		logf("    ✓ Escrow is RESOLVED — claimFunds succeeded!")
		logf("    ✓ Agent:  %s", escrow.Agent.Hex())
		logf("    ✓ Token:  %s", escrow.Token.Hex())
		logf("    ✓ Amount: %s", escrow.Amount.String())
	} else {
		fatalf("Escrow is NOT resolved — claimFunds did not execute. Check relayer logs for revert reason.")
	}

	logf("")
	logf("Simulation complete. Full escrow cycle verified against real VynxSettlement contract.")
}

// ── On-chain helpers ──────────────────────────────────────────────────────────

func txOpts(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, chainID *big.Int) *bind.TransactOpts {
	addr := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, addr)
	must(err, "PendingNonceAt for "+addr.Hex())

	tip, err := client.SuggestGasTipCap(ctx)
	must(err, "SuggestGasTipCap")

	return &bind.TransactOpts{
		From:      addr,
		Nonce:     new(big.Int).SetUint64(nonce),
		GasTipCap: tip,
		Context:   ctx,
		Signer: func(a common.Address, tx *types.Transaction) (*types.Transaction, error) {
			ethSigner := types.LatestSignerForChainID(chainID)
			return types.SignTx(tx, ethSigner, key)
		},
	}
}

func sendETH(ctx context.Context, client *ethclient.Client, fromKey *ecdsa.PrivateKey, to common.Address, chainID, amount *big.Int) {
	from := crypto.PubkeyToAddress(fromKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, from)
	must(err, "PendingNonceAt for funder")

	gasPrice, err := client.SuggestGasPrice(ctx)
	must(err, "SuggestGasPrice")

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasFeeCap: gasPrice,
		GasTipCap: big.NewInt(1e9),
		Gas:       21000,
		To:        &to,
		Value:     amount,
	})

	ethSigner := types.LatestSignerForChainID(chainID)
	signed, err := types.SignTx(tx, ethSigner, fromKey)
	must(err, "sign ETH transfer")

	must(client.SendTransaction(ctx, signed), "send ETH transfer")
	waitMined(ctx, client, signed.Hash(), "ETH transfer")
}

func deployMockToken(ctx context.Context, client *ethclient.Client, deployerKey *ecdsa.PrivateKey, chainID *big.Int) common.Address {
	bytecode, err := hex.DecodeString(mockTokenBytecode)
	must(err, "decode MockToken bytecode")

	deployer := crypto.PubkeyToAddress(deployerKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, deployer)
	must(err, "PendingNonceAt for deployer")

	gasPrice, err := client.SuggestGasPrice(ctx)
	must(err, "SuggestGasPrice for deploy")

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasFeeCap: gasPrice,
		GasTipCap: big.NewInt(1e9),
		Gas:       2_000_000,
		Data:      bytecode,
	})

	ethSigner := types.LatestSignerForChainID(chainID)
	signed, err := types.SignTx(tx, ethSigner, deployerKey)
	must(err, "sign deploy tx")

	must(client.SendTransaction(ctx, signed), "send deploy tx")
	receipt := waitMined(ctx, client, signed.Hash(), "MockToken deploy")
	return receipt.ContractAddress
}

// callMint calls MockToken.mint(to, amount) — selector 0x40c10f19.
func callMint(ctx context.Context, client *ethclient.Client, minterKey *ecdsa.PrivateKey, token, to common.Address, amount, chainID *big.Int) {
	// mint(address,uint256) = 0x40c10f19
	data := common.Hex2Bytes("40c10f19")
	data = append(data, common.LeftPadBytes(to.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	sendContractTx(ctx, client, minterKey, token, data, chainID, "mint")
}

// callApprove calls MockToken.approve(spender, amount) — selector 0x095ea7b3.
func callApprove(ctx context.Context, client *ethclient.Client, ownerKey *ecdsa.PrivateKey, token, spender common.Address, amount, chainID *big.Int) {
	// approve(address,uint256) = 0x095ea7b3
	data := common.Hex2Bytes("095ea7b3")
	data = append(data, common.LeftPadBytes(spender.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	sendContractTx(ctx, client, ownerKey, token, data, chainID, "approve")
}

// callLockIntent calls VynxSettlement.lockIntent(intentId, token, amount) — selector 0x47643164.
func callLockIntent(ctx context.Context, client *ethclient.Client, agentKey *ecdsa.PrivateKey, settlement common.Address, intentID [32]byte, token common.Address, amount, chainID *big.Int) {
	// lockIntent(bytes32,address,uint256) = 0x47643164
	data := common.Hex2Bytes("47643164")
	data = append(data, intentID[:]...)
	data = append(data, common.LeftPadBytes(token.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	sendContractTx(ctx, client, agentKey, settlement, data, chainID, "lockIntent")
}

func sendContractTx(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, to common.Address, data []byte, chainID *big.Int, label string) {
	from := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, from)
	must(err, "PendingNonceAt for "+label)

	gasPrice, err := client.SuggestGasPrice(ctx)
	must(err, "SuggestGasPrice for "+label)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasFeeCap: gasPrice,
		GasTipCap: big.NewInt(1e9),
		Gas:       500_000,
		To:        &to,
		Data:      data,
	})

	ethSigner := types.LatestSignerForChainID(chainID)
	signed, err := types.SignTx(tx, ethSigner, key)
	must(err, "sign "+label+" tx")

	must(client.SendTransaction(ctx, signed), "send "+label+" tx")
	waitMined(ctx, client, signed.Hash(), label)
}

func waitMined(ctx context.Context, client *ethclient.Client, txHash common.Hash, label string) *types.Receipt {
	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			if receipt.Status == 0 {
				fatalf("%s transaction reverted (tx: %s)", label, txHash.Hex())
			}
			return receipt
		}
		select {
		case <-ctx.Done():
			fatalf("timeout waiting for %s to be mined", label)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// ── General helpers ───────────────────────────────────────────────────────────

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func must(err error, context string) {
	if err != nil {
		fatalf("%s: %v", context, err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nFATAL: "+format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func printBanner(user, solver, settlement common.Address, relayerHTTP string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║       VynX Live Simulation (Real Contract)       ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Printf("  User (ephemeral):    %s\n", user.Hex())
	fmt.Printf("  Solver (ephemeral):  %s\n", solver.Hex())
	fmt.Printf("  Settlement contract: %s\n", settlement.Hex())
	fmt.Printf("  Relayer:             %s\n", relayerHTTP)
	fmt.Println()
}

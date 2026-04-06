# VynX Relayer Runbook

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `BASE_RPC_URL` | string | -- | Yes | HTTP or WebSocket RPC endpoint (Base L2 node or Anvil) |
| `RELAYER_PRIVATE_KEY` | string | -- | Yes | 32-byte hex ECDSA key (no `0x` prefix) for the relayer wallet |
| `SETTLEMENT_CONTRACT_ADDRESS` | string | -- | Yes | Deployed `VynxSettlement` contract address |
| `CHAIN_ID` | uint64 | -- | Yes | EVM chain ID (`8453` = Base mainnet, `31337` = Anvil) |
| `AUCTION_TIMEOUT_MS` | uint64 | `200` | No | Auction bid window in milliseconds |
| `PORT` | string | `8080` | No | HTTP server listen port |

A `.env` file is loaded automatically if present. Shell-level exports always take precedence.

---

## Prerequisites

- Go 1.21+
- `golangci-lint` (for `make lint`)
- `abigen` from go-ethereum (for `make bindings`)
- Foundry (`anvil`, `forge`) for local development

---

## Local E2E Setup (Anvil)

### 1. Start Anvil

```bash
anvil --chain-id 31337
```

Anvil starts with 10 deterministic accounts, each funded with 10,000 ETH.

### 2. Deploy VynxSettlement

From the `vynx-settlement-mvp` repository:

```bash
cd ../vynx-settlement-mvp
RELAYER_SIGNER="0x70997970C51812dc3A010C7d01b50e0d17dc79C8" make deploy-anvil
```

This deploys the contract at the deterministic address `0x5FbDB2315678afecb367f032d93F642f64180aa3`.

**Important:** The `RELAYER_SIGNER` address must match the public key derived from the relayer's `RELAYER_PRIVATE_KEY`.

### 3. Configure Environment

```bash
cp .env.example .env
```

Edit `.env`:

```env
BASE_RPC_URL=http://127.0.0.1:8545
RELAYER_PRIVATE_KEY=59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d
SETTLEMENT_CONTRACT_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
CHAIN_ID=31337
AUCTION_TIMEOUT_MS=200
PORT=8080
```

The private key above is Anvil account #1, whose public address (`0x70997970...`) matches the `RELAYER_SIGNER` used at contract deployment.

### 4. Build and Run

```bash
make build
./bin/relayer
```

### 5. Run the E2E Simulation

In a separate terminal (with Anvil and the relayer both running):

```bash
make simulate
```

The simulator performs a full escrow cycle:
1. Deploys a MockToken ERC-20 contract
2. Mints tokens to an ephemeral user
3. Approves the settlement contract for spending
4. Calls `lockIntent` to escrow tokens on-chain
5. Submits an intent to the relayer via HTTP
6. Connects as a solver via WebSocket, listens for the broadcast, and sends a bid
7. Waits for the relayer to call `claimFunds` on-chain
8. Verifies the escrow is marked as resolved

---

## Makefile Targets

| Target | Command | Description |
|--------|---------|-------------|
| `make build` | `go build -o bin/relayer ./cmd/relayer/...` | Compile the relayer binary |
| `make test` | `go test -race -v -count=1 ./...` | Run all tests with race detector |
| `make lint` | `golangci-lint run ./...` | Static analysis (errcheck, gosec, staticcheck) |
| `make bindings` | `abigen --abi ... --pkg bindings ...` | Regenerate Go bindings from ABI JSON |
| `make simulate` | `go run ./cmd/simulate/...` | Run E2E simulation against Anvil |
| `make tidy` | `go mod tidy` | Synchronize go.mod and go.sum |
| `make clean` | `rm -rf bin/` | Remove compiled binaries |

---

## Troubleshooting

### `ECDSAInvalidSignature` on `claimFunds`

**Cause:** The ECDSA V-byte is not normalized. Go's `crypto.Sign` produces V in {0, 1}; OpenZeppelin ECDSA v5 expects V in {27, 28}.

**Fix:** Ensure the executor normalizes V after signing:
```go
if sig[64] < 27 {
    sig[64] += 27
}
```

This is already implemented in `internal/txmanager/executor.go`.

### `nonce too low` errors

**Cause:** The in-memory nonce counter has drifted from the node's authoritative pending nonce. This can happen after a failed transaction that consumed a nonce on the node but was not acknowledged by the relayer.

**Behavior:** The executor detects nonce errors and calls `NonceQueue.Resync()` automatically, which re-fetches the pending nonce via RPC and resets the atomic counter.

### `auction engine at capacity` (503)

**Cause:** The intent channel buffer (256) is full, meaning the auction engine is consuming intents slower than they arrive.

**Fix:** This is transient backpressure. The client should retry with exponential backoff. If persistent, investigate whether auction goroutines are leaking (check for missing `Cleanup` calls).

### `relayerSigner` mismatch

**Cause:** The `RELAYER_PRIVATE_KEY` in `.env` does not match the `RELAYER_SIGNER` address used when deploying the VynxSettlement contract.

**Verification:**
```bash
# Derive the public address from the private key
cast wallet address --private-key $RELAYER_PRIVATE_KEY

# Check the contract's relayerSigner
cast call $SETTLEMENT_CONTRACT_ADDRESS "relayerSigner()(address)" --rpc-url $BASE_RPC_URL
```

Both addresses must match.

### WebSocket solvers disconnected

**Cause:** The solver's outbound queue (256 messages) is full. Slow consumers are disconnected immediately to prevent head-of-line blocking for other solvers.

**Fix:** Ensure solver implementations consume messages promptly. The relayer logs these events at `WARN` level.

---

## Production Considerations

- **Key management:** Replace `RELAYER_PRIVATE_KEY` env var with a KMS-backed signer (AWS KMS, GCP Cloud KMS, or HashiCorp Vault)
- **WebSocket origin check:** The `CheckOrigin` function currently allows all origins. Restrict to known solver domains in production
- **Metrics:** Add Prometheus instrumentation for auction latency, bid counts, settlement success/failure rates, and nonce resyncs
- **Rate limiting:** Add per-IP rate limiting on `POST /v1/intent` to prevent intent spam
- **TLS:** Run behind a reverse proxy (nginx, Caddy) with TLS termination

# VynX Core Relayer

VynX Core Relayer is a High-Frequency Trading (HFT) off-chain settlement engine for the Base L2 network (OP Stack). It receives user swap intents over HTTP, runs a 200 ms Order Flow Auction (OFA) to select the best solver bid, and submits the winning settlement transaction to the `VynxSettlement` Solidity contract. The entire hot path operates exclusively in RAM — no database reads or writes occur between intent ingestion and on-chain submission — making sub-200 ms end-to-end latency achievable on Base mainnet.

---

## Architecture

### RAM-Only Hot Path

All auction state, bid lists, and pending vouchers live in process memory. There is no PostgreSQL, Redis, or any other external store in the execution path. The design rationale is strict: any synchronous database round-trip (typically 1–10 ms for a local Postgres, 20–100 ms for a remote one) would breach the 200 ms total latency budget that the spec mandates. Memory is reclaimed aggressively after each auction closes via `delete(map, key)`, preventing the in-memory auction registry from growing unboundedly under sustained load.

### Event Bus (Channel-Based Dependency Injection)

Packages `internal/auction` and `internal/txmanager` have zero knowledge of each other. All inter-slice communication uses typed Go channels created in `cmd/relayer/main.go` and injected as plain channel arguments:

```
intentCh   chan *core.Intent    ingress.Handler  →  auction.Engine
bidCh      chan *core.Bid       ingress.Hub      →  auction.Engine
voucherCh  <-chan *core.Voucher auction.Engine   →  txmanager.Executor
txFailedCh chan core.IntentID   txmanager.Executor → failure drainer
```

This eliminates circular imports entirely. The Go compiler enforces the boundary at build time — no linter rule or convention document is needed.

### Lock Sharding (Per-Intent `sync.Mutex`)

The auction engine keeps one `auctionEntry` per active intent, each with its own `sync.Mutex`. Concurrent bid submissions for different intents never contend on the same lock. A global `sync.RWMutex` on the `Engine` guards only map pointer reads and writes (microsecond-duration critical sections). Under a Multi-Path Parallel (MPP) burst — where hundreds of intents arrive within the same millisecond — this sharding prevents the single-lock bottleneck that would serialise all bid processing through one contention point.

Locking hierarchy (always acquired in this order to prevent deadlocks):

1. `Engine.mu` — to look up or register an entry pointer.
2. `auctionEntry.mu` — to append a bid or select a winner within that entry.

The outer lock is released before the inner lock is ever acquired.

### Goroutine-per-Auction Model

Each call to `Engine.StartAuction` launches a dedicated goroutine that owns the full lifecycle of one auction: it waits for `time.NewTimer(200ms)`, selects the winner, emits a `Voucher`, and calls `Cleanup`. Failures in one auction goroutine (e.g., no bids, nil winner) do not affect any other auction goroutine. `time.NewTimer` is used instead of `time.After` because `time.After` leaks the underlying goroutine until the timer fires even if the caller exits early — a subtle but real goroutine leak that accumulates under load.

### OP Stack Gas Estimation

Base L2 charges two independent fees per transaction:

- **L2 Execution Fee**: standard EVM gas × (baseFee + priorityFee). Estimated with `ethclient.EstimateGas` and inflated by 10% for inclusion variance.
- **L1 Data Fee**: charged by the sequencer based on the compressed calldata size. The fee is NOT a transaction field — the sequencer deducts it from the sender balance at inclusion. `OPStackClient.EstimateL1DataFee` queries the canonical `GasPriceOracle` precompile at `0x420000000000000000000000000000000000000F` to retrieve this value pre-flight.

Standard Ethereum tools that only call `EstimateGas` will systematically underestimate the true cost of every Base transaction, causing "insufficient funds" reverts under load.

### Atomic Nonce Queue

`NonceQueue` manages the relayer's transaction nonce with `sync/atomic.Uint64`. Each call to `Next()` performs a single `Add(1)` instruction and returns the pre-increment value — no mutex, no contention, O(1) unconditionally. Under concurrent settlement (future multi-executor design), every goroutine receives a distinct nonce without any serialisation point. On nonce desync errors (node rejects with "nonce too low"), `Resync` performs one `PendingNonceAt` RPC call and atomically overwrites the counter, re-anchoring all subsequent `Next()` calls.

---

## Repository Structure

```
vynx-relayer-mvp/
├── Makefile                          # build, test, lint, bindings targets
├── CLAUDE.md                         # AI engineer system prompt and unbreakable rules
├── vynx_relayer_spec.md              # authoritative architecture spec (source of truth)
├── go.mod / go.sum                   # module dependencies
├── bindings/
│   ├── abi/VynxSettlement.json       # Solidity contract ABI (input to abigen)
│   └── vynx_settlement.go            # auto-generated Go bindings (do not edit)
├── cmd/
│   └── relayer/
│       ├── main.go                   # DI root: wires all slices via Event Bus channels
│       └── main_test.go              # end-to-end routing smoke test
└── internal/
    ├── core/
    │   ├── intent.go                 # Intent domain type (user swap request)
    │   ├── bid.go                    # Bid domain type (solver offer)
    │   └── voucher.go                # Voucher domain type (auction settlement proof)
    ├── signer/
    │   ├── key_vault.go              # ECDSA key isolation; raw bytes zeroed after parse
    │   └── eip712.go                 # EIP-712 domain separator and intent digest
    ├── auction/
    │   ├── state.go                  # Engine registry; per-intent lock sharding; Cleanup GC
    │   └── matcher.go                # NewEngine, StartAuction, SubmitBid, runAuction goroutine
    ├── txmanager/
    │   ├── nonce_queue.go            # Wait-free atomic nonce counter with Resync fallback
    │   ├── opstack_client.go         # OP Stack–aware gas wrapper (L2 exec + L1 data fee)
    │   └── executor.go               # Voucher consumer; EIP-712 sign; settle() dispatcher
    └── ingress/
        ├── handler.go                # POST /v1/intent REST handler; GET /health
        └── mempool_ws.go             # WebSocket Hub; solver bid ingestion; intent broadcast
```

---

## Environment Variables

| Variable                    | Type    | Default | Required | Purpose                                                                 |
|-----------------------------|---------|---------|----------|-------------------------------------------------------------------------|
| `BASE_RPC_URL`              | string  | —       | Yes      | WebSocket or HTTP RPC endpoint for the Base L2 node (or Anvil locally) |
| `RELAYER_PRIVATE_KEY`       | string  | —       | Yes      | 32-byte hex ECDSA private key (no `0x` prefix) for the relayer wallet  |
| `SETTLEMENT_CONTRACT_ADDRESS` | string | —      | Yes      | Deployed `VynxSettlement` contract address (`0x...`)                    |
| `CHAIN_ID`                  | uint64  | —       | Yes      | EVM chain ID (`8453` for Base mainnet, `31337` for Anvil)              |
| `AUCTION_TIMEOUT_MS`        | uint64  | `200`   | No       | Per-auction bid window in milliseconds; reduce in tests                |
| `PORT`                      | string  | `8080`  | No       | TCP port on which the HTTP server listens                              |

Environment variables are loaded from a `.env` file if present. Shell-level exports always take precedence over `.env` values — existing environment variables are never overwritten.

---

## Quick Start

**Prerequisites:** Go 1.21+, `golangci-lint`, `abigen` (from `go-ethereum`).

```bash
# 1. Install dependencies
go mod tidy

# 2. Generate Go bindings from the Solidity ABI (required before first build)
make bindings

# 3. Compile the relayer binary to bin/relayer
make build

# 4. Run the full test suite with the race detector
make test

# 5. Run the linter (errcheck, gosec, staticcheck)
make lint
```

For local development against Anvil:

```bash
# Start Anvil with a deterministic key set
anvil --chain-id 31337

# Copy and populate the environment file
cp .env.example .env
# Edit .env: set BASE_RPC_URL=http://127.0.0.1:8545, CHAIN_ID=31337,
# RELAYER_PRIVATE_KEY=<anvil account 0 key>, SETTLEMENT_CONTRACT_ADDRESS=<deployed addr>

./bin/relayer
```

---

## API

### `POST /v1/intent`

Submits a new swap intent. Returns `202 Accepted` immediately; the auction runs asynchronously.

**Request body:**

```json
{
  "id":             "intent-uuid-v4",
  "sender":         "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
  "token_in":       "0x4200000000000000000000000000000000000006",
  "token_out":      "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
  "amount_in":      "1000000000000000000",
  "min_amount_out": "990000000",
  "deadline":       1775000000,
  "nonce":          1,
  "signature":      "0xabc123..."
}
```

- `amount_in` and `min_amount_out`: decimal strings representing wei-precision integers.
- `deadline`: Unix timestamp in seconds; must be in the future.
- `signature`: hex-encoded EIP-712 signature (optional in MVP; included for forward compatibility).

**Response `202 Accepted`:**

```json
{
  "intent_id": "intent-uuid-v4",
  "status":    "queued"
}
```

**Error responses:**

| Status | Condition                                           |
|--------|-----------------------------------------------------|
| `400`  | Missing required field, non-positive amount, expired deadline, or malformed signature |
| `405`  | HTTP method other than POST                         |
| `503`  | Auction engine channel at capacity; client should retry |

---

### `GET /health`

Liveness probe. Returns `200 OK` with no meaningful latency.

```json
{"status": "ok"}
```

---

### `WS /v1/ws/solvers`

WebSocket endpoint for solver connections. After upgrade, the relayer pushes a `new_intent` message for every accepted intent. Solvers respond with bid messages on the same connection.

**Relayer → Solver (intent broadcast):**

```json
{
  "type": "new_intent",
  "intent": {
    "id":             "intent-uuid-v4",
    "token_in":       "0x4200000000000000000000000000000000000006",
    "token_out":      "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
    "amount_in":      "1000000000000000000",
    "min_amount_out": "990000000",
    "deadline":       1775000000
  }
}
```

**Solver → Relayer (bid):**

```json
{
  "intent_id":  "intent-uuid-v4",
  "solver":     "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
  "amount_out": "995000000",
  "gas_price":  "1000000000"
}
```

- `amount_out` and `gas_price`: decimal wei strings.
- Bids with non-positive `amount_out` or `gas_price` are silently dropped with a `WARN` log.
- Solvers whose outbound queue fills (slow consumers) are disconnected immediately to prevent head-of-line blocking for fast solvers.

---

## Design Decisions

- **No database (RAM-only + `delete(map, key)` GC).** A database round-trip of even 1 ms is incompatible with a 200 ms end-to-end budget that also includes two RPC calls (gas estimation and transaction broadcast). After each auction closes, `Engine.Cleanup` calls `delete(e.auctions, id)` to release the entry — this is the only GC mechanism needed. The Go runtime reclaims the underlying memory at the next collection cycle.

- **`gorilla/websocket` over Go's `net/http` stdlib.** Go's standard library has no built-in WebSocket implementation. Using raw HTTP hijacking to implement the WebSocket framing protocol manually would introduce significant complexity and deviation from RFC 6455. `gorilla/websocket` is the de facto standard Go WebSocket library.

- **`time.NewTimer` not `time.After` in auction goroutines.** `time.After` returns a channel backed by a `time.Timer` that is allocated per call and cannot be garbage-collected until the timer fires. In an auction engine processing thousands of intents, using `time.After` would leak one goroutine per auction for the full 200 ms duration even if the parent goroutine exited early. `time.NewTimer` allows explicit `Stop()` via `defer`, releasing the timer resource immediately on early exit.

- **`select/default` for all channel sends.** Every outbound channel send in the hot path (voucher emission, txFailed notification, bid relay, intent broadcast) uses a non-blocking `select/default`. If the consumer is behind, the message is dropped and logged at `ERROR` or `WARN`. Blocking an auction goroutine or an HTTP handler on a slow consumer would cascade latency across unrelated intents.

- **`crypto.ToECDSA` + `clear(b)` over `crypto.HexToECDSA`.** `crypto.HexToECDSA` is a convenience wrapper that does not zero the intermediate byte slice after parsing. `NewKeyVaultFromHex` decodes the hex string to `[]byte`, calls `crypto.ToECDSA`, and immediately runs `clear(b)` (Go 1.21+ builtin) to overwrite the raw key bytes in memory. This limits the window during which a heap dump or memory inspection could expose the private key.

- **Buffered channels for all Event Bus slots.** All channels have explicit buffer capacities (`intentBuf=256`, `bidBuf=10_000`, `txFailBuf=128`). Zero-capacity channels would make every producer synchronously wait for a consumer, transforming the event bus into a serialisation barrier. The buffer sizes are tuned to absorb realistic burst peaks without dropping messages; they are not infinite to preserve back-pressure semantics.

- **Interface injection for `settlementCaller` and `gasTipCapper`.** `Executor` depends on two package-private interfaces rather than concrete `*bindings.VynxSettlement` and `*OPStackClient` types. This enables the entire executor test suite to run with stub implementations without any live RPC node, Anvil, or deployed contract, while the production wiring in `main.go` passes the real concrete types directly.

---

## Testing

The test suite is designed to surface concurrency defects under the Go race detector. Every test file calls `t.Parallel()` and the full suite is run as `go test -race -v -count=1 ./...` via `make test`.

| Test file                              | Coverage scope                                                                     |
|----------------------------------------|------------------------------------------------------------------------------------|
| `internal/auction/matcher_test.go`     | Basic winner selection, duplicate auction rejection, no-bid cleanup, and two stress tests: 10k concurrent bids into one auction, and 50 simultaneous auctions with 200 bids each |
| `internal/txmanager/txmanager_test.go` | Sequential and concurrent `NonceQueue.Next()` (10k goroutines, zero duplicates, full range coverage), `Resync` re-anchoring, and `isNonceError` pattern matching |
| `internal/txmanager/executor_test.go`  | Route-failure-to-txFailedCh, non-blocking txFailedCh (zero-capacity channel, 3 vouchers, no deadlock), context cancellation exit, channel-close exit, and 50-goroutine concurrent voucher injection |
| `cmd/relayer/main_test.go`             | End-to-end HTTP routing smoke test (intent submission → auction start path via in-process wiring) |

The race detector is mandatory: `go test ./...` without `-race` is not an acceptable substitute. Any data race detected by the race detector is treated as a build-blocking defect.

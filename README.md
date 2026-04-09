# VynX Relayer

**RAM-only Order Flow Auction engine for Base L2.** Sub-200ms intent-to-settlement latency with zero database dependencies.

The VynX Relayer receives user swap intents over HTTP, runs a real-time auction among competing solvers via WebSocket, and settles the winning bid on-chain through the `VynxSettlement` escrow contract. The entire hot path -- from intent ingestion to transaction broadcast -- executes exclusively in process memory.

---

## Key Properties

- **Zero-DB architecture** -- all auction state, bid lists, and nonce management live in RAM. No PostgreSQL, Redis, or disk I/O on the hot path.
- **Lock-sharded concurrency** -- per-intent mutexes eliminate contention between independent auctions. Hundreds of concurrent intents share no locks.
- **Non-blocking event bus** -- typed Go channels with `select/default` ensure no goroutine ever blocks another. Slow consumers are dropped, not waited on.
- **OP Stack native** -- dual-fee gas estimation (L2 execution + L1 data fee) via the Base `GasPriceOracle` precompile.
- **EIP-191 settlement** -- escrow-based `lockIntent` / `claimFunds` / `refundIntent` flow with on-chain ECDSA signature verification.

---

## Quick Start

> **Grant reviewer?** The complete settlement stack — Relayer + TypeScript agent +
> CDP MPC wallet + Mock Solver — runs with a single command from the monorepo root:
> `make reviewer-demo`. The steps below describe running the Relayer in isolation.

```bash
# Prerequisites: Go 1.21+, Anvil (Foundry), golangci-lint

# 1. Build
make build

# 2. Run tests (race detector enabled)
make test

# 3. Start Anvil + deploy VynxSettlement (see docs/RUNBOOK.md)
anvil --chain-id 31337

# 4. Configure .env and run
cp .env.example .env   # edit with your Anvil keys
./bin/relayer

# 5. E2E simulation (separate terminal)
make simulate
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](docs/ARCHITECTURE.md) | Event Bus topology, lock sharding, concurrency model, ECDSA V-byte normalization, OP Stack gas model |
| [API Reference](docs/API.md) | REST endpoints (`POST /v1/intent`, `GET /health`), WebSocket protocol (`WS /v1/ws/solvers`), auction mechanics |
| [Runbook](docs/RUNBOOK.md) | Environment variables, local Anvil setup, E2E simulation, Makefile targets, troubleshooting guide |

---

## Architecture at a Glance

```
POST /v1/intent ──> intentCh ──> Auction Engine ──> voucherCh ──> Executor ──> VynxSettlement (Base L2)
                                      ^
WS /v1/ws/solvers ──> bidCh ─────────┘
       ^
       │ (Mock Solver in scripts/run_workflow.ts submits a bid after 2 s to close the demo loop)
```

All channels are wired in `cmd/relayer/main.go`. Packages `internal/auction`, `internal/txmanager`, and `internal/ingress` have zero circular imports -- the Event Bus pattern enforces this at compile time.

The auction window is configured via `AUCTION_TIMEOUT_MS` (default `200`; set to `4000` in
`docker-compose.yml` to accommodate the Mock Solver's simulated computation delay).

---

## Repository Structure

```
vynx-relayer-mvp/
├── cmd/relayer/          # DI root: wires all slices via Event Bus channels
├── cmd/simulate/         # E2E simulation against Anvil
├── internal/
│   ├── core/             # Domain types: Intent, Bid, Voucher
│   ├── signer/           # KeyVault (ECDSA isolation), ClaimDigest (EIP-191)
│   ├── auction/          # OFA engine with lock sharding
│   ├── txmanager/        # Executor, NonceQueue (atomic), OPStackClient
│   └── ingress/          # HTTP handler, WebSocket hub
├── bindings/             # Auto-generated Go bindings (abigen)
├── docs/                 # Architecture, API, and Runbook documentation
└── Makefile              # build, test, lint, bindings, simulate
```

---

## License

Proprietary. All rights reserved.

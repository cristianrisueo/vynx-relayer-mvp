# VynX Relayer API Reference

Base URL: `http://localhost:8080` (configurable via `PORT` env var)

---

## REST Endpoints

### `POST /v1/intent`

Submit a new swap intent. Returns `202 Accepted` immediately; the auction runs asynchronously in the background.

**Request:**

```http
POST /v1/intent HTTP/1.1
Content-Type: application/json

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

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique intent identifier (UUID v4 recommended) |
| `sender` | string | Yes | Ethereum address of the intent originator |
| `token_in` | string | Yes | ERC-20 token address being sold |
| `token_out` | string | Yes | ERC-20 token address being bought |
| `amount_in` | string | Yes | Wei-precision decimal string; must be positive |
| `min_amount_out` | string | Yes | Wei-precision decimal string; must be non-negative |
| `deadline` | integer | Yes | Unix timestamp (seconds); must be in the future |
| `nonce` | integer | Yes | Intent nonce for replay protection |
| `signature` | string | No | Hex-encoded EIP-712 signature (forward-compatible; not verified in MVP) |

**Response `202 Accepted`:**

```json
{
  "intent_id": "intent-uuid-v4",
  "status": "queued"
}
```

**Error Responses:**

| Status | Condition |
|--------|-----------|
| `400 Bad Request` | Missing required field, non-positive amount, expired deadline, malformed hex signature |
| `405 Method Not Allowed` | HTTP method other than POST |
| `503 Service Unavailable` | Auction engine channel at capacity; client should retry with backoff |

**Notes:**
- Unknown JSON fields are rejected (strict decoding via `DisallowUnknownFields`)
- The handler is non-blocking: if the intent channel is full, it returns 503 rather than stalling

---

### `GET /health`

Liveness probe. Returns `200 OK` with negligible latency.

**Response:**

```json
{"status": "ok"}
```

---

## WebSocket Endpoint

### `WS /v1/ws/solvers`

Bidirectional WebSocket for solver connections. After upgrade, the relayer pushes intent broadcasts; solvers respond with bids on the same connection.

**Connection:**

```
ws://localhost:8080/v1/ws/solvers
```

**Timing Parameters:**

| Parameter | Value | Description |
|-----------|-------|-------------|
| Max message size | 4,096 bytes | Per-frame read limit |
| Pong wait | 60s | Idle read deadline per client |
| Ping period | 54s | Keepalive ping interval |
| Write wait | 10s | Per-frame write deadline |
| Send buffer | 256 messages | Per-client outbound queue depth |

---

#### Relayer -> Solver: Intent Broadcast

Sent to all connected solvers when a new intent is accepted.

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

---

#### Solver -> Relayer: Bid Submission

```json
{
  "intent_id":  "intent-uuid-v4",
  "solver":     "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
  "amount_out": "995000000",
  "gas_price":  "1000000000"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `intent_id` | string | Yes | Must match an active auction |
| `solver` | string | Yes | Solver's Ethereum address |
| `amount_out` | string | Yes | Decimal wei string; must be positive |
| `gas_price` | string | Yes | Decimal wei string; must be positive |

**Behavior:**
- Bids with non-positive `amount_out` or `gas_price` are silently dropped with a `WARN` log
- Late bids (after auction timer expires) are rejected at `DEBUG` level
- Solvers whose outbound queue fills (slow consumers) are disconnected immediately to prevent head-of-line blocking

---

## Auction Mechanics

1. Intent arrives via `POST /v1/intent`
2. Intent is broadcast to all connected solvers via WebSocket
3. Auction window opens (default: 200ms, configurable via `AUCTION_TIMEOUT_MS`)
4. Solvers submit bids over WebSocket
5. Timer fires -> winner selected: **highest `amount_out`**, tie-break by **highest `gas_price`**
6. Voucher emitted to Executor for on-chain settlement
7. Auction entry cleaned up (`delete(map, key)`)

If no bids are received within the auction window, no voucher is emitted and the entry is cleaned up.

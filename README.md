# Polyester Go Examples

Runnable examples for the official Polyester Go SDK.

These examples are intentionally small. Start with read-only market data, then move to
authenticated reads, then opt in to live devnet order writes, transfers, withdrawals,
or chain Funding UserOps when your credentials and balances are ready.

## Requirements

- Go 1.25+
- A Polyester API key for authenticated examples
- Trading balance for order-writing examples
- Funding balance + owner private key for chain Funding→Trading / Funding→external submit

## Install

```bash
GOPRIVATE='github.com/Fabric-Labs/*' \
GONOSUMDB='github.com/Fabric-Labs/*' \
go mod download
```

This repo may `replace` the SDK with a sibling `../polyester-sdk-go` checkout when local
APIs (for example `OrderKey` / `chain`) are ahead of the published tag. A standalone clone
without the sibling still resolves the published module version in `go.mod`.

The SDK repository is private; your GitHub account and Git credentials must have access.

## Configure

```bash
cp .env.example .env
```

Fill in:

- `POLYESTER_API_KEY_ID`
- `POLYESTER_API_PRIVATE_KEY`
- `POLYESTER_ACCOUNT_ID`

If you already have `polyester-sdk-go/.env` configured for devnet, you can reuse the same
three values here. Placeholder text from `.env.example` is ignored for public examples but
will fail authenticated or trading examples until replaced with real credentials.
Username is optional; API-key authentication with the Account ID is valid.

Public market-data examples can run without credentials. Authenticated reads and all order
examples require an API key.

For a subaccount-scoped key, attach an **API-key policy** that grants ledger
read permission for balance reads and private balance streams. Add trading
permission for order mutations. The API-key policy is distinct from the
subaccount policy, and both apply.

The SDK does not implicitly read a `.env` file. These examples load `.env`, then pass
credentials explicitly to the client constructor.

Load env vars in your shell when running examples:

```bash
set -a && source .env && set +a
```

## Safety Model

Opt-in flags are **separate**. Never overload `POLYESTER_EXAMPLES_ENABLE_TRADING` onto
transfers, withdrawals, or chain submit:

| Flag | Gates |
| --- | --- |
| `POLYESTER_EXAMPLES_ENABLE_TRADING=1` | Order / trigger writes (`03`, `07`–`12`, `18`, live `10`) |
| `POLYESTER_EXAMPLES_ENABLE_TRANSFERS=1` | Internal transfer submit (`14`) + dest account env |
| `POLYESTER_EXAMPLES_ENABLE_WITHDRAWALS=1` | API-key withdraw **submit** (`15`; prepare always runs) |
| `POLYESTER_EXAMPLES_ENABLE_CHAIN_FUNDING_TO_TRADING=1` | Funding→Trading UserOp **submit** (`16`; encode always) |
| `POLYESTER_EXAMPLES_ENABLE_CHAIN_EXTERNAL_SUBMIT=1` | Funding→external UserOp **submit** (`17`; encode always) |

The default max quote notional for order examples is:

```bash
POLYESTER_EXAMPLES_MAX_QUOTE=10
```

Use a devnet API key with a policy that allows the actions you enable. These examples are
educational, not production trading systems.

## Qty / price dual path

Examples use **decimal strings** for human-readable order qty and price.

For bots already in wire units, prefer scaled inputs:

```go
scale, ok := client.Catalogs.BaseQuantityScaleForSymbol(symbol)
if !ok {
    log.Fatal("base quantity scale is unavailable; wait for hydrated catalogs")
}
qty := models.MustQtyScaled(1_000_000).WithScale(scale).WithSymbol(symbol)
price := models.PriceFromTicksInt(100_000_000)
_, err = client.Orders.Create(ctx, models.CreateOrderRequest{
    Symbol: &symbol, Side: "buy", OrderType: "limit", TIF: &tif,
    Qty: models.QtyFromScaled(qty), Price: &price, PostOnly: true,
}, nil)
// Reads: order.Price.Ticks(), order.OrigQty.Scaled()
```

`PriceTicks.Ticks()` returns protocol units (1e6), not market tick-size alignment.
Transfers/withdraws use `AssetAmountInput`, not order `QtyInput`.
Private order-stream `QtyScaled.Scale()` metadata may be `nil`. Treat `.Scaled()`
as raw wire units and resolve the base quantity scale from hydrated catalogs by
symbol or `symbol_id`; never invent a fallback scale.

## Funding vs Trading

Deposits land in the **Funding** account. Spot orders spend **Trading** balance.

Before running live order examples, move funds from Funding to Unified Trading:

- In the Polyester UI / wallet flow, or
- Via example `16-funding-to-trading` (encodes `TradingGateway.deposit`; set
  `POLYESTER_EXAMPLES_ENABLE_CHAIN_FUNDING_TO_TRADING=1` plus
  `POLYESTER_OWNER_PRIVATE_KEY` to broadcast a UserOp)

API-key Trading→Funding withdraw is example `15` (prepare always; submit only when
`POLYESTER_EXAMPLES_ENABLE_WITHDRAWALS=1`).

## Examples

Run examples from the repository root after configuring `.env`.

| Command | Credentials | Opt-in | What it teaches |
| --- | --- | --- | --- |
| `01-public-market-data` | Optional | — | REST overview, trades, candles |
| `02-balances-and-orders-read` | Required | — | Balances, open orders, history |
| `03-place-and-cancel-limit-order` | Required | `ENABLE_TRADING` | Post-only limit create, cancel, cleanup |
| `04-public-realtime-trades` | Optional | — | Public trade websocket |
| `05-public-orderbook-stream` | Optional | — | Snapshot + stream order book |
| `06-market-overview-stream` | Optional | — | Snapshot + stream market overview |
| `07-batch-create-and-cancel-all` | Required | `ENABLE_TRADING` | Batch create + prefix-targeted per-order cancel |
| `08-batch-replace` | Required | `ENABLE_TRADING` | Batch create, `batch_replace` price, cleanup |
| `09-batch-cancel` | Required | `ENABLE_TRADING` | Batch create, `Orders.BatchCancel` by client id |
| `10-rsi-signal-bot` | Required for live | Optional `ENABLE_TRADING` | Candles + RSI; optional small limit |
| `11-twap-trigger` | Required | `ENABLE_TRADING` | Triggers API TWAP create → list/get → cancel |
| `12-ladder-trigger` | Required | `ENABLE_TRADING` | Triggers API ladder create → list/get → cancel |
| `13-private-realtime` | Required | — | Private orders + balances websocket |
| `14-internal-transfer` | Required | `ENABLE_TRANSFERS` + dest account | Tiny `InternalTransfers.Create` |
| `15-api-key-trading-withdraw` | Required | Prepare always; `ENABLE_WITHDRAWALS` to submit | Trading→Funding prepare / submit |
| `16-funding-to-trading` | Required | Encode always; `ENABLE_CHAIN_FUNDING_TO_TRADING` to submit | Encode deposit; optional UserOp |
| `17-funding-to-external` | Required | Encode needs dest; `ENABLE_CHAIN_EXTERNAL_SUBMIT` to submit | Encode withdrawToChain; optional UserOp |
| `18-trailing-stop-trigger` | Required | `ENABLE_TRADING` | Standalone trailing-stop (SELL market-IOC) create → list/get → cancel |

Suggested order: `01` → `04`/`05`/`06` → `02` → `13` → `10` (dry) → `03` → `07`/`08`/`09` →
`11`/`12`/`18` → money-movement examples when those flags are intentionally enabled.

### Live smoke

```bash
make live-smoke
# or: bash scripts/live-smoke.sh
```

Runs all examples in order. Gated examples print `SKIP` and continue when their flag is
missing. Set `LIVE_SMOKE_STRICT=1` to fail instead of skipping.

### Read-Only

```bash
go run ./cmd/01-public-market-data
go run ./cmd/04-public-realtime-trades
go run ./cmd/05-public-orderbook-stream
go run ./cmd/06-market-overview-stream
```

Realtime examples exit after 30 seconds if no data arrives (common on quiet devnet markets).

### Authenticated Reads

```bash
go run ./cmd/02-balances-and-orders-read
go run ./cmd/13-private-realtime
```

`13` subscribes to private orders and balances concurrently and prints up to
`POLYESTER_EXAMPLES_STREAM_COUNT` events per stream (or 30s timeout). No trading flag.

### Explicit Live Writes

```bash
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/03-place-and-cancel-limit-order
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/07-batch-create-and-cancel-all
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/08-batch-replace
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/09-batch-cancel
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/11-twap-trigger
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/12-ladder-trigger
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/18-trailing-stop-trigger
```

`07` cleans up with prefix-targeted per-order cancel. `09` demonstrates `Orders.BatchCancel`.
TWAP/ladder/trailing-stop use the Triggers API (separate lifecycle from normal orders):
create → list/get → cancel, plus best-effort `cancel_all` for resting child orders.
Standalone trailing stops are SELL market-IOC; list/get project `trigger_type`, `side`, and
`parent_order_id` (empty for standalone).

```bash
go run ./cmd/10-rsi-signal-bot
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/10-rsi-signal-bot
```

### Transfers / withdrawals / chain

```bash
POLYESTER_EXAMPLES_ENABLE_TRANSFERS=1 \
POLYESTER_EXAMPLES_TRANSFER_DEST_ACCOUNT_ID=... \
go run ./cmd/14-internal-transfer

go run ./cmd/15-api-key-trading-withdraw
POLYESTER_EXAMPLES_ENABLE_WITHDRAWALS=1 go run ./cmd/15-api-key-trading-withdraw

go run ./cmd/16-funding-to-trading
POLYESTER_EXAMPLES_ENABLE_CHAIN_FUNDING_TO_TRADING=1 \
POLYESTER_OWNER_PRIVATE_KEY=0x... \
go run ./cmd/16-funding-to-trading

POLYESTER_EXAMPLES_EXTERNAL_DESTINATION=0x... \
go run ./cmd/17-funding-to-external
POLYESTER_EXAMPLES_ENABLE_CHAIN_EXTERNAL_SUBMIT=1 \
POLYESTER_OWNER_PRIVATE_KEY=0x... \
POLYESTER_EXAMPLES_EXTERNAL_DESTINATION=0x... \
go run ./cmd/17-funding-to-external
```

## Useful Settings

- `POLYESTER_EXAMPLES_SYMBOL`: default `BTC-USDT` (devnet has live BTC orderbook/candles; ETH-USDT may be quiet)
- `POLYESTER_EXAMPLES_TIMEFRAME`: default `1m`
- `POLYESTER_EXAMPLES_CANDLE_LIMIT`: default `100`
- `POLYESTER_EXAMPLES_MAX_QUOTE`: default `10`
- `POLYESTER_EXAMPLES_RSI_PERIOD`: default `14`
- `POLYESTER_EXAMPLES_RSI_OVERSOLD`: default `30`
- `POLYESTER_EXAMPLES_RSI_OVERBOUGHT`: default `70`
- `POLYESTER_EXAMPLES_ORDER_TIMEOUT_SEC`: default `15`
- `POLYESTER_EXAMPLES_STREAM_COUNT`: default `5`
- `POLYESTER_EXAMPLES_ORDERBOOK_DEPTH`: default `50`
- `POLYESTER_EXAMPLES_TRANSFER_AMOUNT`: default `0.01`
- `POLYESTER_EXAMPLES_WITHDRAW_AMOUNT`: default `0.01`
- `POLYESTER_EXAMPLES_CHAIN_AMOUNT`: default `1`
- `POLYESTER_EXAMPLES_EXTERNAL_DESTINATION`: required to encode example `17`
- `POLYESTER_EXAMPLES_EXTERNAL_CHAIN_ID`: default `6`
- `POLYESTER_OWNER_PRIVATE_KEY`: smart-account owner for UserOp submit (`16`/`17`)

## Notes For Bot Builders

- Pass decimal strings for `qty` and `price`. Do not use floats for order inputs.
- `GetCandles` is newest-first. Sort by `TsSec` ascending before feeding a
  chronological indicator such as RSI.
- Use client order IDs for idempotency and cleanup.
- Check open orders before placing a new bot order.
- Decide how your production bot will track positions. The RSI example intentionally avoids a
  persistent state file and only uses balances plus open orders.
- Treat the RSI strategy as a teaching example. It is deliberately naive.

## Development

```bash
go test ./...
go build ./cmd/...
make live-smoke
```

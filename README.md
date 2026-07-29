# Polyester Go Examples

Runnable examples for the official Polyester Go SDK.

These examples are intentionally small. Start with read-only market data, then move to
authenticated reads, then opt in to live devnet order writes when your API key and trading
balance are ready.

## Requirements

- Go 1.25+
- A Polyester API key for authenticated examples
- Trading balance for order-writing examples

## Install

```bash
go mod download
```

This repository uses a local `replace` for `polyester-sdk-go` when working inside the
Fabric monorepo. Remove or adjust the `replace` directive in `go.mod` when consuming
published SDK versions from another checkout.

Requires `polyester-sdk-go` **v0.1.0a27+** (`BatchReplace` / `GetBatchReplaceStatus`).

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

Public market-data examples can run without credentials. Authenticated reads and all order
examples require an API key.

The SDK does not implicitly read a `.env` file. These examples load `.env`, then pass
credentials explicitly to the client constructor.

Load env vars in your shell when running examples:

```bash
set -a && source .env && set +a
```

## Safety Model

Live order writes are disabled by default. Any example that places orders requires:

```bash
POLYESTER_EXAMPLES_ENABLE_TRADING=1
```

The default max quote notional is:

```bash
POLYESTER_EXAMPLES_MAX_QUOTE=10
```

Use a devnet API key with a policy that allows trading. These examples are educational, not
production trading systems.

## Qty / price dual path

Examples use **decimal strings** for human-readable order qty and price.

For bots already in wire units, prefer scaled inputs:

```go
qty := models.MustQtyScaled(1_000_000).WithScale(8)
price := models.PriceFromTicksInt(100_000_000)
_, err = client.Orders.Create(ctx, models.CreateOrderRequest{
    Symbol: &symbol, Side: "buy", OrderType: "limit", TIF: &tif,
    Qty: models.QtyFromScaled(qty), Price: &price, PostOnly: true,
}, nil)
// Reads: order.Price.Ticks, order.OrigQty.Scaled
```

`PriceTicks.Ticks` are protocol units (1e6), not market tick-size alignment.
Transfers/withdraws use `AssetAmountInput`, not order `QtyInput`.

## Funding vs Trading

Deposits land in the Funding account. Spot orders spend Trading balance.

Before running live order examples, move funds from Funding to Unified Trading in the Polyester UI
or through the wallet/on-chain flow. The current SDK examples do not automate Funding to Trading
movement because that path is wallet-driven, not a simple API-key RPC.

## Examples

Run examples from the repository root after configuring `.env`.

| Command | Credentials | Live orders | What it teaches |
| --- | --- | --- | --- |
| `01-public-market-data` | Optional | No | REST overview, trades, candles |
| `02-balances-and-orders-read` | Required | No | Balances, open orders, history |
| `03-place-and-cancel-limit-order` | Required | Yes (`POLYESTER_EXAMPLES_ENABLE_TRADING=1`) | Post-only limit create, cancel, cleanup |
| `04-public-realtime-trades` | Optional | No | Public trade websocket |
| `05-public-orderbook-stream` | Optional | No | Snapshot + stream order book |
| `06-market-overview-stream` | Optional | No | Snapshot + stream market overview |
| `07-batch-create-and-cancel-all` | Required | Yes (`POLYESTER_EXAMPLES_ENABLE_TRADING=1`) | Batch limit create, `cancel_all` cleanup |
| `08-batch-replace` | Required | Yes (`POLYESTER_EXAMPLES_ENABLE_TRADING=1`) | Batch create, `BatchReplace` + status, cleanup |
| `10-rsi-signal-bot` | Required | Optional (`POLYESTER_EXAMPLES_ENABLE_TRADING=1`) | Candles + RSI signal; optional small limit order |

Suggested order: `01` → `04`/`05`/`06` → `02` → `10` (dry) → `03` → `07` / `10` (live) when ready.

TWAP, ladder, and standalone trigger examples are intentionally omitted for v1. They use the
triggers API (separate lifecycle from normal orders) and are a poor fit for a small cookbook.

### Read-Only

```bash
go run ./cmd/01-public-market-data
```

Lists markets, recent trades, and candles using public Polyester market data.

```bash
go run ./cmd/06-market-overview-stream
```

Subscribes to merged market-overview rows (REST snapshot plus live websocket updates).

```bash
go run ./cmd/04-public-realtime-trades
```

Subscribes to public trade updates.

```bash
go run ./cmd/05-public-orderbook-stream
```

Creates a snapshot-plus-stream order book subscription and prints top-of-book updates.

Realtime examples exit after 30 seconds if no data arrives (common on quiet devnet markets).

### Authenticated Reads

```bash
go run ./cmd/02-balances-and-orders-read
```

Prints ledger balances, open orders, and recent order history.

### Explicit Live Writes

```bash
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/03-place-and-cancel-limit-order
```

Places a small post-only buy limit order, waits for it to appear in open orders (when devnet
read indexing is healthy), cancels it, and attempts cleanup with `cancel_all`.

On devnet, `orders.create` may return `accepted` while `list_open` / `orders.get` stay empty for
a while. That is a known backend read-indexing issue, not an SDK bug. The example still submits
cancel by `order_id` when reads lag.

```bash
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/07-batch-create-and-cancel-all
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/08-batch-replace
```

Places two small post-only buy limits via `batch_create`, optionally checks open orders, then
flattens with `cancel_all`. Each order uses half of `POLYESTER_EXAMPLES_MAX_QUOTE` so total
notional stays within the safety cap.

`08-batch-replace` creates two post-only buys, submits `Orders.BatchReplace` for a same-symbol quote refresh, prints the admission receipt, polls `Orders.GetBatchReplaceStatus`, then cleans up owned orders by prefix.


```bash
go run ./cmd/10-rsi-signal-bot
```

Fetches Polyester candles, computes RSI in plain Go, and prints the signal. By default this is
read-only.

```bash
POLYESTER_EXAMPLES_ENABLE_TRADING=1 go run ./cmd/10-rsi-signal-bot
```

When RSI crosses a threshold, places a small live limit order, monitors briefly, and cancels it if
it remains open. Uses `cancel_all` in a `defer` for cleanup.

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

## Notes For Bot Builders

- Pass decimal strings for `qty` and `price`. Do not use floats for order inputs.
- Use client order IDs for idempotency and cleanup.
- Check open orders before placing a new bot order.
- Decide how your production bot will track positions. The RSI example intentionally avoids a
  persistent state file and only uses balances plus open orders.
- Treat the RSI strategy as a teaching example. It is deliberately naive.

## Development

```bash
go test ./...
```

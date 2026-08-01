package polyesterexamples

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	polyester "github.com/Fabric-Labs/polyester-sdk-go"
	"github.com/Fabric-Labs/polyester-sdk-go/codecs"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
)

var defaultSymbolCandidates = []string{"BTC-USDT", "ETH-USDT", "SOL-USDT", "BNB-USDT"}

const defaultSymbol = "BTC-USDT"

// PickSymbol chooses a tradable symbol from spot config.
func PickSymbol(spotRaw map[string]any, preferred string) string {
	symbols := map[string]struct{}{}
	for _, pair := range pairsFromSpot(spotRaw) {
		if symbol, _ := pair["symbol"].(string); symbol != "" {
			symbols[symbol] = struct{}{}
		}
	}
	if preferred != "" {
		if _, ok := symbols[preferred]; ok {
			return preferred
		}
	}
	for _, candidate := range defaultSymbolCandidates {
		if _, ok := symbols[candidate]; ok {
			return candidate
		}
	}
	for symbol := range symbols {
		return symbol
	}
	if preferred != "" {
		return preferred
	}
	return defaultSymbol
}

// PairForSymbol returns pair metadata for a symbol.
func PairForSymbol(spotRaw map[string]any, symbol string) (map[string]any, error) {
	for _, pair := range pairsFromSpot(spotRaw) {
		if sym, _ := pair["symbol"].(string); sym == symbol {
			return pair, nil
		}
	}
	return nil, fmt.Errorf("symbol %q was not found in spot config", symbol)
}

func pairsFromSpot(spotRaw map[string]any) []map[string]any {
	out := make([]map[string]any, 0)
	for _, key := range []string{"pairs", "symbols"} {
		raw, _ := spotRaw[key].([]any)
		for _, item := range raw {
			if pair, ok := item.(map[string]any); ok {
				out = append(out, pair)
			}
		}
	}
	return out
}

func BaseAssetSymbol(pair map[string]any, symbol string) string {
	if value := stringField(pair, "baseAsset", "base_asset"); value != "" {
		return value
	}
	parts := strings.SplitN(symbol, "-", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return symbol
}

func QuoteAssetSymbol(pair map[string]any, symbol string) string {
	if value := stringField(pair, "quoteAsset", "quote_asset"); value != "" {
		return value
	}
	parts := strings.SplitN(symbol, "-", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return "USDT"
}

func BaseAssetID(client *polyester.Client, pair map[string]any, symbol string) *uint32 {
	if id := uintField(pair, "baseAssetId", "base_asset_id"); id != nil {
		return id
	}
	return client.Catalogs.LedgerIDForAsset(BaseAssetSymbol(pair, symbol))
}

func QuoteAssetID(client *polyester.Client, pair map[string]any, symbol string) *uint32 {
	if id := uintField(pair, "quoteAssetId", "quote_asset_id"); id != nil {
		return id
	}
	return client.Catalogs.LedgerIDForAsset(QuoteAssetSymbol(pair, symbol))
}

func TickSize(pair map[string]any) float64 {
	return parseDecimalField(pair, 0.01, "tickSize", "tick_size")
}

func StepSize(pair map[string]any) float64 {
	return parseDecimalField(pair, 0.001, "stepSize", "step_size")
}

func MinQtyBase(pair map[string]any) float64 {
	step := StepSize(pair)
	return parseDecimalField(pair, step, "minQtyBase", "min_qty_base")
}

func MinNotionalQuote(pair map[string]any) float64 {
	return parseDecimalField(pair, 10, "minNotionalQuote", "min_notional_quote")
}

// FormatDecimal trims trailing zeroes from a float string.
func FormatDecimal(value float64) string {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimRight(text, ".")
	}
	if text == "" {
		return "0"
	}
	return text
}

// FormatQty formats a quantity aligned to the pair step size without float artifacts.
func FormatQty(value float64, pair map[string]any) string {
	step := StepSize(pair)
	if step <= 0 {
		return FormatDecimal(value)
	}
	units := math.Round(value / step)
	aligned := units * step
	decimals := stepDecimals(pair)
	text := strconv.FormatFloat(aligned, 'f', decimals, 64)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimRight(text, ".")
	}
	if text == "" {
		return "0"
	}
	return text
}

func stepDecimals(pair map[string]any) int {
	stepText := stringField(pair, "stepSize", "step_size")
	if stepText == "" {
		stepText = FormatDecimal(StepSize(pair))
	}
	if index := strings.Index(stepText, "."); index >= 0 {
		return len(stepText) - index - 1
	}
	return 0
}

// AvailableTradingBalance returns human-readable trading balance for an asset id.
func AvailableTradingBalance(balances models.BalancesList, assetID uint32) float64 {
	for _, row := range balances.Balances {
		if row.AssetID != assetID {
			continue
		}
		raw := row.Available
		if raw == "" {
			raw = row.Trading
		}
		formatted, err := codecs.FormatLedgerU128(raw, codecs.LedgerScale)
		if err != nil {
			return 0
		}
		value, err := strconv.ParseFloat(formatted, 64)
		if err != nil {
			return 0
		}
		return value
	}
	return 0
}

// SlightlyLowerLimitPrice returns price minus one tick (for batch_modify demos).
func SlightlyLowerLimitPrice(price string, pair map[string]any) (string, error) {
	value, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return "", fmt.Errorf("parse price %q: %w", price, err)
	}
	step := TickSize(pair)
	if step <= 0 {
		step = 0.01
	}
	lower := value - step
	if lower < step {
		lower = step
	}
	return FormatDecimal(roundToTick(alignToStep(lower, step, false), step)), nil
}

// PickUSDTZipperAsset selects USDT (ledger_id=1) from deposit/withdraw config.
func PickUSDTZipperAsset(cfg models.DepositWithdrawConfig) (models.ZipperAssetConfig, bool) {
	for _, asset := range cfg.Assets {
		if asset.LedgerID == 1 || strings.EqualFold(asset.Asset, "USDT") {
			return asset, true
		}
	}
	return models.ZipperAssetConfig{}, false
}

// HumanAmountToE18 converts a human decimal amount to ledger e18 units.
func HumanAmountToE18(amount float64) *big.Int {
	if amount <= 0 {
		return big.NewInt(0)
	}
	rat := new(big.Rat).SetFloat64(amount)
	if rat == nil {
		return big.NewInt(0)
	}
	scale := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	scaled := new(big.Rat).Mul(rat, scale)
	return new(big.Int).Quo(scaled.Num(), scaled.Denom())
}

// FarBelowMarketPrice returns a post-only buy price slightly below the reference price.
func FarBelowMarketPrice(lastPrice float64, pair map[string]any) string {
	step := TickSize(pair)
	target := lastPrice * 0.995
	if target < step {
		target = step
	}
	return FormatDecimal(roundToTick(alignToStep(target, step, false), step))
}

// ResolvePostOnlyBuyLimitPrice returns a post-only buy price slightly below the live book.
func ResolvePostOnlyBuyLimitPrice(ctx context.Context, client *polyester.Client, symbol string, pair map[string]any) (string, error) {
	tickSize := formatTickSize(pair)
	if client != nil {
		book, err := client.Orderbook.Get(ctx, symbol, 5)
		if err == nil {
			if price, ok := postOnlyBuyPriceFromBook(book, tickSize); ok {
				return price, nil
			}
		}
		overview, err := client.MarketOverview.List(ctx, []string{symbol}, 5, "", false)
		if err == nil {
			for _, row := range overview.Markets {
				if row.Symbol != symbol || row.LastPrice.Ticks() <= 0 {
					continue
				}
				if price, ok := postOnlyBuyPriceFromLastTicks(row.LastPrice.Ticks(), tickSize); ok {
					return price, nil
				}
			}
		}
	}
	return FarBelowMarketPrice(100, pair), nil
}

func postOnlyBuyPriceFromBook(book models.OrderbookData, tickSize string) (string, bool) {
	if len(book.Bids) == 0 {
		return "", false
	}
	tickTicks, err := codecs.ParsePriceTicks(tickSize, "tick_size")
	if err != nil || tickTicks == 0 {
		return "", false
	}
	bidTicks := book.Bids[0].Price.Ticks()
	target := bidTicks - int64(tickTicks)
	if target < int64(tickTicks) {
		target = int64(tickTicks)
	}
	if len(book.Asks) > 0 && book.Asks[0].Price.Ticks() > 0 {
		askTicks := book.Asks[0].Price.Ticks()
		maxPostOnly := askTicks - int64(tickTicks)
		if target > maxPostOnly {
			target = maxPostOnly
		}
	}
	if target < int64(tickTicks) {
		return "", false
	}
	return codecs.FormatPriceTicks(target), true
}

func postOnlyBuyPriceFromLastTicks(lastTicks int64, tickSize string) (string, bool) {
	if lastTicks <= 0 {
		return "", false
	}
	tickTicks, err := codecs.ParsePriceTicks(tickSize, "tick_size")
	if err != nil || tickTicks == 0 {
		return "", false
	}
	target := lastTicks * 995 / 1000
	aligned := (target / int64(tickTicks)) * int64(tickTicks)
	if aligned < int64(tickTicks) {
		aligned = int64(tickTicks)
	}
	return codecs.FormatPriceTicks(aligned), true
}

func formatTickSize(pair map[string]any) string {
	if value := stringField(pair, "tickSize", "tick_size"); value != "" {
		return value
	}
	return "0.01"
}

// PriceFromTicks converts wire price ticks to a decimal price.
func PriceFromTicks(priceTicks string) float64 {
	ticks, err := strconv.ParseFloat(strings.TrimSpace(priceTicks), 64)
	if err != nil {
		return 0
	}
	return ticks / 1_000_000
}

// MarketableLimitPrice returns a limit price slightly through the market for the given side.
func MarketableLimitPrice(side string, lastPrice float64, pair map[string]any) (string, error) {
	step := TickSize(pair)
	switch side {
	case "buy":
		return FormatDecimal(roundToTick(alignToStep(lastPrice*1.001, step, true), step)), nil
	case "sell":
		return FormatDecimal(roundToTick(alignToStep(lastPrice*0.999, step, false), step)), nil
	default:
		return "", fmt.Errorf("side must be 'buy' or 'sell'")
	}
}

// BuyQtyForQuoteCap sizes a buy order within quote balance and safety caps.
func BuyQtyForQuoteCap(availableQuote, maxQuote, price float64, pair map[string]any) (string, error) {
	quoteToUse := math.Min(availableQuote, maxQuote)
	minimum := MinNotionalQuote(pair)
	if quoteToUse < minimum {
		return "", fmt.Errorf(
			"need at least %s quote trading balance; available=%s",
			FormatDecimal(minimum),
			FormatDecimal(availableQuote),
		)
	}
	step := StepSize(pair)
	rawQty := quoteToUse / price
	qty := alignToStep(rawQty, step, false)
	minQty := MinBaseQtyForNotional(pair, price)
	if qty < minQty {
		qty = minQty
	}
	return FormatQty(qty, pair), nil
}

// SellQtyForQuoteCap sizes a sell order within base balance and quote notional cap.
func SellQtyForQuoteCap(availableBase, maxQuote, price float64, pair map[string]any) (string, error) {
	step := StepSize(pair)
	targetBase := math.Min(availableBase, maxQuote/price)
	qty := alignToStep(targetBase, step, false)
	if qty <= 0 {
		return "", fmt.Errorf("no base trading balance is available to sell")
	}
	minimum := MinNotionalQuote(pair)
	if qty*price < minimum {
		return "", fmt.Errorf(
			"sell size is below min notional %s; available_base=%s",
			FormatDecimal(minimum),
			FormatDecimal(availableBase),
		)
	}
	return FormatQty(qty, pair), nil
}

// MinBaseQtyForNotional returns the smallest base qty that satisfies min qty and
// min notional at the given price (aligned up to the pair step).
func MinBaseQtyForNotional(pair map[string]any, price float64) float64 {
	step := StepSize(pair)
	minQty := MinQtyBase(pair)
	minimum := MinNotionalQuote(pair)
	if price <= 0 {
		return minQty
	}
	notionalQty := alignToStep(minimum/price, step, true)
	minQtyAligned := alignToStep(minQty, step, true)
	return math.Max(notionalQty, math.Max(minQtyAligned, step))
}

func alignToStep(value, step float64, roundUp bool) float64 {
	if step <= 0 {
		return value
	}
	units := value / step
	if roundUp {
		units = math.Ceil(units)
	} else {
		units = math.Floor(units)
	}
	if units < 1 && value > 0 && !roundUp {
		return 0
	}
	return units * step
}

func roundToTick(value, step float64) float64 {
	if step <= 0 {
		return value
	}
	stepText := strconv.FormatFloat(step, 'f', -1, 64)
	decimals := 0
	if index := strings.Index(stepText, "."); index >= 0 {
		decimals = len(stepText) - index - 1
	}
	factor := math.Pow(10, float64(decimals))
	return math.Round(value*factor) / factor
}

func stringField(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			if text, ok := raw.(string); ok && text != "" {
				return text
			}
		}
	}
	return ""
}

func parseDecimalField(values map[string]any, defaultValue float64, keys ...string) float64 {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			switch typed := raw.(type) {
			case string:
				if value, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
					return value
				}
			case float64:
				return typed
			case int:
				return float64(typed)
			case int64:
				return float64(typed)
			}
		}
	}
	return defaultValue
}

func uintField(values map[string]any, keys ...string) *uint32 {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			switch typed := raw.(type) {
			case float64:
				id := uint32(typed)
				return &id
			case int:
				id := uint32(typed)
				return &id
			case int64:
				id := uint32(typed)
				return &id
			case uint32:
				return &typed
			}
		}
	}
	return nil
}

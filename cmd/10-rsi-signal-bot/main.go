package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	polyester "github.com/Fabric-Labs/polyester-sdk-go"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
)

const botPrefix = "rsi-bot"

func main() {
	settings := polyesterexamples.LoadSettings()
	requireAuth := settings.EnableTrading
	cfg, err := polyesterexamples.ClientConfigFromEnv(requireAuth)
	if err != nil {
		log.Fatal(err)
	}
	client, err := polyesterexamples.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	if err := polyesterexamples.WaitForCatalogs(ctx, client); err != nil {
		log.Fatal(err)
	}

	spot, err := client.MarketData.GetSpotConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	symbol := polyesterexamples.PickSymbol(spot.Raw, settings.Symbol)
	pair, err := polyesterexamples.PairForSymbol(spot.Raw, symbol)
	if err != nil {
		log.Fatal(err)
	}

	candleLimit := settings.CandleLimit
	if candleLimit < settings.RSIPeriod+2 {
		candleLimit = settings.RSIPeriod + 2
	}
	candles, err := client.MarketData.GetCandles(ctx, &symbol, nil, settings.Timeframe, candleLimit, nil, nil, false)
	if err != nil {
		log.Fatal(err)
	}
	closes := make([]float64, 0, len(candles.Candles))
	for _, candle := range candles.Candles {
		if candle.Close == "" {
			continue
		}
		closePrice, err := strconv.ParseFloat(candle.Close, 64)
		if err != nil {
			continue
		}
		closes = append(closes, closePrice)
	}
	if len(closes) == 0 {
		log.Fatalf("No candles returned for %s", symbol)
	}

	signal, err := polyesterexamples.EvaluateRSISignal(closes, settings.RSIPeriod, settings.RSIOversold, settings.RSIOverbought)
	if err != nil {
		log.Fatal(err)
	}
	lastClose := closes[len(closes)-1]
	fmt.Printf(
		"%s %s: close=%s rsi=%s previous_rsi=%s action=%s reason=%s\n",
		symbol,
		settings.Timeframe,
		polyesterexamples.FormatDecimal(lastClose),
		formatRSI(signal.LatestRSI),
		formatRSI(signal.PreviousRSI),
		signal.Action,
		signal.Reason,
	)

	if signal.Action == "hold" {
		return
	}

	price, err := polyesterexamples.MarketableLimitPrice(signal.Action, lastClose, pair)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Signal order candidate: side=%s price=%s\n", signal.Action, price)

	if !settings.EnableTrading {
		fmt.Println(
			"Dry run only. Set POLYESTER_EXAMPLES_ENABLE_TRADING=1 to allow " +
				"the bot to place and manage this order.",
		)
		return
	}

	if err := polyesterexamples.EnsureNoOpenOrdersWithPrefix(ctx, client, botPrefix); err != nil {
		log.Fatal(err)
	}

	balances, err := client.Balances.List(ctx, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	priceValue, err := strconv.ParseFloat(price, 64)
	if err != nil {
		log.Fatal(err)
	}
	qty, err := qtyForSignal(client, balances, symbol, pair, signal.Action, priceValue, settings.MaxQuote)
	if err != nil {
		log.Fatal(err)
	}

	clientOrderID := polyesterexamples.UniqueClientOrderID(botPrefix)
	fmt.Printf(
		"Placing live limit order: side=%s price=%s qty=%s client_order_id=%s\n",
		signal.Action, price, qty, clientOrderID,
	)

	defer polyesterexamples.CancelAllForSymbol(ctx, client, symbol)

	tif := "gtc"
	created, err := client.Orders.Create(ctx, models.CreateOrderRequest{
		Symbol:        &symbol,
		Side:          signal.Action,
		OrderType:     "limit",
		TIF:           &tif,
		Qty:           models.QtyFromDecimal(qty),
		Price:         priceInputPtr(price),
		ClientOrderID: &clientOrderID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created: status=%s order_id=%s\n", created.Status, created.OrderID)

	finalStatus, err := polyesterexamples.CancelAfterTimeout(
		ctx,
		client,
		clientOrderID,
		symbol,
		&created.OrderID,
		settings.OrderTimeoutSec,
		settings.PollSec,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Final observed status: %s\n", finalStatus)
	if finalStatus == "canceled_after_timeout_unconfirmed" {
		fmt.Println(
			"Cancel was submitted, but open-order reads did not confirm cleanup. " +
				"This can happen on devnet when OMS read indexing lags.",
		)
	}
}

func qtyForSignal(
	client *polyester.Client,
	balances models.BalancesList,
	symbol string,
	pair map[string]any,
	side string,
	price, maxQuote float64,
) (string, error) {
	switch side {
	case "buy":
		quoteAssetID := polyesterexamples.QuoteAssetID(client, pair, symbol)
		if quoteAssetID == nil {
			return "", fmt.Errorf("could not resolve quote asset id for %s", symbol)
		}
		availableQuote := polyesterexamples.AvailableTradingBalance(balances, *quoteAssetID)
		fmt.Printf(
			"Using up to %s %s of trading balance\n",
			polyesterexamples.FormatDecimal(maxQuote),
			polyesterexamples.QuoteAssetSymbol(pair, symbol),
		)
		return polyesterexamples.BuyQtyForQuoteCap(availableQuote, maxQuote, price, pair)
	case "sell":
		baseAssetID := polyesterexamples.BaseAssetID(client, pair, symbol)
		if baseAssetID == nil {
			return "", fmt.Errorf("could not resolve base asset id for %s", symbol)
		}
		availableBase := polyesterexamples.AvailableTradingBalance(balances, *baseAssetID)
		return polyesterexamples.SellQtyForQuoteCap(availableBase, maxQuote, price, pair)
	default:
		return "", fmt.Errorf("side must be 'buy' or 'sell'")
	}
}

func formatRSI(value *float64) string {
	if value == nil {
		return "n/a"
	}
	scaled := math.Round(*value*100) / 100
	return polyesterexamples.FormatDecimal(scaled)
}

func priceInputPtr(s string) *models.PriceInput {
	p := models.PriceFromDecimal(s)
	return &p
}

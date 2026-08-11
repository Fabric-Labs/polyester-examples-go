package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
)

func main() {
	settings := polyesterexamples.LoadSettings()
	cfg, err := polyesterexamples.ClientConfigFromEnv(false)
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

	overview, err := client.MarketOverview.List(ctx, nil, 5, "", false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Markets")
	for _, market := range overview.Markets {
		last := "-"
		index := "-"
		if market.LastPrice.Ticks() > 0 {
			last = market.LastPrice.Format()
		}
		if market.IndexPrice.Ticks() > 0 {
			index = market.IndexPrice.Format()
		}
		fmt.Printf(
			"  %s: symbol_id=%d last_price=%s index_price=%s\n",
			market.Symbol, market.SymbolID, last, index,
		)
	}

	spot, err := client.MarketData.GetSpotConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	symbol := polyesterexamples.PickSymbol(spot.Raw, settings.Symbol)
	fmt.Printf("\nUsing symbol: %s\n", symbol)

	trades, err := client.MarketData.GetTrades(ctx, &symbol, nil, 5, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nRecent trades")
	for _, trade := range trades.Trades {
		fmt.Printf("  %s price.ticks=%d qty.scaled=%d\n", trade.Side, trade.Price.Ticks(), trade.Qty.Scaled())
	}

	candles, err := client.MarketData.GetCandles(ctx, &symbol, nil, settings.Timeframe, 5, nil, nil, false)
	if err != nil {
		log.Fatal(err)
	}
	timeframe := candles.Timeframe
	if timeframe == "" {
		timeframe = settings.Timeframe
	}
	fmt.Printf("\nRecent %s candles\n", timeframe)
	for _, candle := range candles.Candles {
		fmt.Printf(
			"  ts=%d open=%s high=%s low=%s close=%s volume=%s\n",
			candle.TsSec, candle.Open, candle.High, candle.Low, candle.Close, candle.Volume,
		)
	}
}

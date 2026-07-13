package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	"github.com/Fabric-Labs/polyester-sdk-go/services"
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

	spot, err := client.MarketData.GetSpotConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	symbol := polyesterexamples.PickSymbol(spot.Raw, settings.Symbol)

	fmt.Printf(
		"Streaming %d order book updates for %s at depth=%d\n",
		settings.StreamCount, symbol, settings.OrderbookDepth,
	)
	sub, err := client.Orderbook.CreateSubscription(ctx, services.CreateSubscriptionOptions{
		Symbol: symbol,
		Depth:  settings.OrderbookDepth,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Close()

	seen := 0
	timeout := time.After(30 * time.Second)
	for seen < settings.StreamCount {
		select {
		case book, ok := <-sub.Updates():
			if !ok {
				if seen == 0 {
					fmt.Println("Stream closed before any order book updates arrived.")
				}
				return
			}
			bestBid := "-"
			bestAsk := "-"
			if len(book.Bids) > 0 {
				bestBid = book.Bids[0].Price.Format()
			}
			if len(book.Asks) > 0 {
				bestAsk = book.Asks[0].Price.Format()
			}
			fmt.Printf("  seq=%s bid=%s ask=%s\n", book.BookSeq, bestBid, bestAsk)
			seen++
		case <-timeout:
			fmt.Println("No order book updates within 30s. The market may be quiet on devnet.")
			return
		}
	}
}

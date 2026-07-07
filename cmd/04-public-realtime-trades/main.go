package main

import (
	"context"
	"fmt"
	"log"
	"time"

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

	spot, err := client.MarketData.GetSpotConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	symbol := polyesterexamples.PickSymbol(spot.Raw, settings.Symbol)

	fmt.Printf("Streaming %d public trades for %s\n", settings.StreamCount, symbol)
	sub, err := client.MarketData.SubscribeTrades(ctx, &symbol, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Close()

	seen := 0
	timeout := time.After(30 * time.Second)
	for seen < settings.StreamCount {
		select {
		case trade, ok := <-sub.Messages():
			if !ok {
				if seen == 0 {
					fmt.Println("Stream closed before any trades arrived.")
				}
				return
			}
			fmt.Printf(
				"  %s price_ticks=%s qty_scaled=%s match_id=%s\n",
				trade.Side, trade.PriceTicks, trade.QtyScaled, trade.MatchID,
			)
			seen++
		case <-timeout:
			fmt.Println("No trades received within 30s. The market may be quiet on devnet.")
			return
		}
	}
}

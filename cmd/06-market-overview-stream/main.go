package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
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
		"Streaming %d merged market-overview snapshots (highlighting %s)\n",
		settings.StreamCount, symbol,
	)
	sub, err := client.MarketOverview.CreateSubscription(ctx, services.MarketOverviewCreateSubscriptionOptions{
		Symbols: []string{symbol},
		Limit:   10,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Close()

	seen := 0
	timeout := time.After(30 * time.Second)
	for seen < settings.StreamCount {
		select {
		case rows, ok := <-sub.Updates():
			if !ok {
				if seen == 0 {
					fmt.Println("Stream closed before any overview snapshots arrived.")
				}
				return
			}
			focus := rowForSymbol(rows, symbol)
			label := symbol
			price := "-"
			if focus != nil {
				label = focus.Symbol
				if focus.LastPrice.Ticks() > 0 {
					price = focus.LastPrice.Format()
				}
			}
			fmt.Printf("  update=%d rows=%d %s last_price=%s\n", seen+1, len(rows), label, price)
			seen++
		case <-timeout:
			fmt.Println("No overview updates within 30s. The market may be quiet on devnet.")
			return
		}
	}
}

func rowForSymbol(rows []models.MarketOverviewEntry, symbol string) *models.MarketOverviewEntry {
	for index := range rows {
		if rows[index].Symbol == symbol {
			return &rows[index]
		}
	}
	if len(rows) > 0 {
		return &rows[0]
	}
	return nil
}

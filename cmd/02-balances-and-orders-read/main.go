package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	"github.com/Fabric-Labs/polyester-sdk-go/codecs"
)

func main() {
	settings := polyesterexamples.LoadSettings()
	cfg, err := polyesterexamples.ClientConfigFromEnv(true)
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

	balances, err := client.Balances.List(ctx, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Balances")
	for _, balance := range balances.Balances {
		fmt.Printf(
			"  asset_id=%d trading=%s available=%s reserved=%s funding=%s\n",
			balance.AssetID,
			formatLedger(balance.Trading),
			formatLedger(balance.Available),
			formatLedger(balance.Reserved),
			formatLedger(balance.Funding),
		)
	}

	spot, err := client.MarketData.GetSpotConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	symbol := polyesterexamples.PickSymbol(spot.Raw, settings.Symbol)

	openOrders, err := client.Orders.ListOpen(ctx, nil, nil, nil, intPtr(20), false, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nOpen orders (%d)\n", len(openOrders.Orders))
	for _, order := range openOrders.Orders {
		fmt.Printf(
			"  client_order_id=%s order_id=%s side=%s status=%s leaves_qty.scaled=%d\n",
			order.ClientOrderID, order.OrderID, order.Side, order.Status, order.LeavesQty.Scaled(),
		)
	}

	history, err := client.Orders.ListHistory(ctx, nil, nil, &symbol, nil, nil, 10, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nRecent order history for %s (%d)\n", symbol, len(history.Orders))
	for _, order := range history.Orders {
		fmt.Printf(
			"  client_order_id=%s order_id=%s side=%s status=%s cum_qty.scaled=%d\n",
			order.ClientOrderID, order.OrderID, order.Side, order.Status, order.CumQty.Scaled(),
		)
	}
}

func formatLedger(raw string) string {
	formatted, err := codecs.FormatLedgerU128(raw, codecs.LedgerScale)
	if err != nil {
		return raw
	}
	return formatted
}

func intPtr(value int) *int {
	return &value
}

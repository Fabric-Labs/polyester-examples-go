package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
)

func main() {
	settings := polyesterexamples.LoadSettings()
	if err := polyesterexamples.RequireTradingEnabled(settings); err != nil {
		log.Fatal(err)
	}
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

	spot, err := client.MarketData.GetSpotConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	symbol := polyesterexamples.PickSymbol(spot.Raw, settings.Symbol)
	pair, err := polyesterexamples.PairForSymbol(spot.Raw, symbol)
	if err != nil {
		log.Fatal(err)
	}

	price, err := polyesterexamples.ResolvePostOnlyBuyLimitPrice(ctx, client, symbol, pair)
	if err != nil {
		log.Fatal(err)
	}
	priceValue, err := strconv.ParseFloat(price, 64)
	if err != nil {
		log.Fatal(err)
	}

	quoteAssetID := polyesterexamples.QuoteAssetID(client, pair, symbol)
	if quoteAssetID == nil {
		log.Fatalf("Could not resolve quote asset id for %s", symbol)
	}

	balances, err := client.Balances.List(ctx, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	availableQuote := polyesterexamples.AvailableTradingBalance(balances, *quoteAssetID)
	perOrderCap := settings.MaxQuote / 2
	qty, err := polyesterexamples.BuyQtyForQuoteCap(availableQuote, perOrderCap, priceValue, pair)
	if err != nil {
		log.Fatal(err)
	}

	clientOrderIDs := []string{
		polyesterexamples.UniqueClientOrderID("example-batch-a"),
		polyesterexamples.UniqueClientOrderID("example-batch-b"),
	}
	fmt.Printf(
		"Batch creating 2 post-only buy limits: symbol=%s price=%s qty=%s each (max ~%s quote per order)\n",
		symbol, price, qty, polyesterexamples.FormatDecimal(perOrderCap),
	)

	defer polyesterexamples.CancelAllForSymbol(ctx, client, symbol)

	tif := "gtc"
	items := make([]models.CreateOrderRequest, 0, len(clientOrderIDs))
	for _, clientOrderID := range clientOrderIDs {
		id := clientOrderID
		items = append(items, models.CreateOrderRequest{
			Symbol:        &symbol,
			Side:          "buy",
			OrderType:     "limit",
			TIF:           &tif,
			Qty:           qty,
			Price:         &price,
			PostOnly:      true,
			ClientOrderID: &id,
		})
	}

	created, err := client.Orders.BatchCreate(ctx, nil, items, nil, &symbol, nil, false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Batch create: accepted=%d rejected=%d\n", created.AcceptedCount, created.RejectedCount)
	for _, item := range created.Results {
		code := item.Code
		if code == "" {
			code = "-"
		}
		fmt.Printf(
			"  client_order_id=%s status=%s order_id=%s code=%s\n",
			item.ClientOrderID, item.Status, item.OrderID, code,
		)
	}
	if created.AcceptedCount == 0 {
		log.Fatal("No batch orders were accepted")
	}

	for _, clientOrderID := range clientOrderIDs {
		openOrder, err := polyesterexamples.WaitForOpenOrder(
			ctx, client, clientOrderID, 50, settings.OrderTimeoutSec, settings.PollSec,
		)
		if err != nil {
			fmt.Printf("  %s: create accepted but open-order reads lagged (%v)\n", clientOrderID, err)
			continue
		}
		fmt.Printf("Visible in open orders: %s status=%s\n", clientOrderID, openOrder.Status)
	}

	canceled, err := client.Orders.CancelAll(ctx, nil, nil, &symbol, nil, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(
		"cancel_all: status=%s matched_orders=%d submitted_cancels=%d\n",
		canceled.Status, canceled.MatchedOrders, canceled.SubmittedCancels,
	)
}

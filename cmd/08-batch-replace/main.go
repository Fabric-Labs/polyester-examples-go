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
	newPrice, err := polyesterexamples.SlightlyLowerLimitPrice(price, pair)
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

	const cleanupPrefix = "example-bmod"
	clientOrderIDs := []string{
		polyesterexamples.UniqueClientOrderID(cleanupPrefix),
		polyesterexamples.UniqueClientOrderID(cleanupPrefix),
	}
	fmt.Printf(
		"Batch create 2 post-only buys, then batch_replace new_price=%s (was %s)\n",
		newPrice, price,
	)

	defer func() {
		if err := polyesterexamples.CancelOwnedOrdersWithPrefix(ctx, client, cleanupPrefix); err != nil {
			fmt.Printf("Cleanup warning: %v\n", err)
		}
	}()

	tif := "gtc"
	items := make([]models.CreateOrderRequest, 0, len(clientOrderIDs))
	for _, clientOrderID := range clientOrderIDs {
		id := clientOrderID
		items = append(items, models.CreateOrderRequest{
			Symbol:        &symbol,
			Side:          "buy",
			OrderType:     "limit",
			TIF:           &tif,
			Qty:           models.QtyFromDecimal(qty),
			Price:         priceInputPtr(price),
			PostOnly:      true,
			ClientOrderID: &id,
		})
	}

	created, err := client.Orders.BatchCreate(ctx, nil, items, nil, &symbol, nil, false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Batch create: accepted=%d rejected=%d\n", created.AcceptedCount, created.RejectedCount)
	if created.AcceptedCount == 0 {
		log.Fatal("No batch orders were accepted")
	}

	replacePrice := models.PriceFromDecimal(newPrice)
	replaceItems := make([]models.BatchReplaceItem, 0, len(clientOrderIDs))
	for _, clientOrderID := range clientOrderIDs {
		p := replacePrice
		replaceItems = append(replaceItems, models.BatchReplaceItem{
			Key:      models.OrderKeyByClientID(clientOrderID),
			NewPrice: &p,
		})
	}
	replaced, err := client.Orders.BatchReplace(ctx, nil, replaceItems, symbol, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(
		"Batch replace admission: batch_request_id=%s status=%s accepted=%d rejected=%d\n",
		replaced.BatchRequestID, replaced.Status, replaced.AcceptedCount, replaced.RejectedCount,
	)
	for _, item := range replaced.Results {
		code := item.Code
		if code == "" {
			code = "-"
		}
		fmt.Printf(
			"  item=%d status=%s old=%s replacement=%s client_order_id=%s code=%s\n",
			item.ItemIndex, item.Status, item.OldOrderID, item.ReplacementOrderID, item.ClientOrderID, code,
		)
	}
	status, err := client.Orders.GetBatchReplaceStatus(ctx, nil, replaced.BatchRequestID, nil)
	if err != nil {
		// Admission already succeeded; status projection can lag on devnet.
		fmt.Printf("Batch replace status lookup lagged (%v); continuing cleanup\n", err)
	} else {
		fmt.Printf(
			"Batch replace status: admission=%s items=%d accepted=%d rejected=%d\n",
			status.AdmissionStatus, len(status.Items), status.AcceptedCount, status.RejectedCount,
		)
	}

	if err := polyesterexamples.CancelOwnedOrdersWithPrefix(ctx, client, cleanupPrefix); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Targeted cleanup completed for owned batch-replace orders")
}

func priceInputPtr(s string) *models.PriceInput {
	p := models.PriceFromDecimal(s)
	return &p
}

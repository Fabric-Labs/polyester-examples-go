package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

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

	const cleanupPrefix = "example-brepl"
	predecessorClientOrderIDs := []string{
		polyesterexamples.UniqueClientOrderID(cleanupPrefix),
		polyesterexamples.UniqueClientOrderID(cleanupPrefix),
	}
	successorClientOrderIDs := []string{
		polyesterexamples.UniqueClientOrderID(cleanupPrefix),
		polyesterexamples.UniqueClientOrderID(cleanupPrefix),
	}
	cleanupClientOrderIDs := append([]string(nil), predecessorClientOrderIDs...)
	fmt.Printf(
		"Batch create 2 post-only buys, then batch_replace new_price=%s (was %s)\n",
		newPrice, price,
	)

	defer func() {
		for _, clientOrderID := range cleanupClientOrderIDs {
			_, err := client.Orders.Cancel(ctx, nil, models.OrderKeyByClientID(clientOrderID), &symbol, nil, nil)
			if err != nil {
				fmt.Printf("Cleanup warning for %s: %v\n", clientOrderID, err)
			}
		}
		fmt.Println("Targeted cleanup completed for tracked batch-replace orders")
	}()

	tif := "gtc"
	items := make([]models.CreateOrderRequest, 0, len(predecessorClientOrderIDs))
	for _, clientOrderID := range predecessorClientOrderIDs {
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
	replaceItems := make([]models.BatchReplaceItem, 0, len(predecessorClientOrderIDs))
	for index, clientOrderID := range predecessorClientOrderIDs {
		p := replacePrice
		successorClientOrderID := successorClientOrderIDs[index]
		replaceItems = append(replaceItems, models.BatchReplaceItem{
			Key:              models.OrderKeyByClientID(clientOrderID),
			NewPrice:         &p,
			NewClientOrderID: &successorClientOrderID,
		})
	}
	requestID := polyesterexamples.UniqueClientOrderID("example-brepl-request")
	replaced, err := client.Orders.BatchReplace(ctx, nil, replaceItems, symbol, nil, &requestID)
	if err != nil {
		log.Fatal(err)
	}
	// If this request had timed out ambiguously, retry the exact same logical batch
	// with requestID. Do not generate a new request ID for an idempotent retry.
	// replaced, err = client.Orders.BatchReplace(ctx, nil, replaceItems, symbol, nil, &requestID)
	fmt.Printf(
		"Batch replace admission: request_id=%s batch_request_id=%s status=%s accepted=%d rejected=%d\n",
		requestID, replaced.BatchRequestID, replaced.Status, replaced.AcceptedCount, replaced.RejectedCount,
	)
	for _, item := range replaced.Results {
		if item.ItemIndex >= 0 && int(item.ItemIndex) < len(cleanupClientOrderIDs) && item.ReplacementOrderID != "" {
			cleanupClientOrderIDs[item.ItemIndex] = successorClientOrderIDs[item.ItemIndex]
		}
		code := item.Code
		if code == "" {
			code = "-"
		}
		fmt.Printf(
			"  item=%d status=%s predecessor_order_id=%s replacement_order_id=%s successor_client_order_id=%s code=%s\n",
			item.ItemIndex, item.Status, item.OldOrderID, item.ReplacementOrderID, item.ClientOrderID, code,
		)
	}
	fmt.Printf("Predecessor client IDs are stale after admission: %v\n", predecessorClientOrderIDs)
	fmt.Printf("Cleanup and later tracking use successor client IDs: %v\n", successorClientOrderIDs)

	deadline := time.Now().Add(time.Duration(settings.OrderTimeoutSec * float64(time.Second)))
	for {
		status, err := client.Orders.GetBatchReplaceStatus(ctx, nil, replaced.BatchRequestID, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(
			"Batch replace status: admission=%s items=%d accepted=%d rejected=%d settled=%t\n",
			status.AdmissionStatus, len(status.Items), status.AcceptedCount, status.RejectedCount,
			models.IsBatchReplaceSettled(status),
		)
		for _, item := range status.Items {
			fmt.Printf(
				"  item=%d phase=%s predecessor_order_id=%s replacement_order_id=%s order_status=%s\n",
				item.ItemIndex, item.Phase, item.OldOrderID, item.ReplacementOrderID, item.OrderStatus,
			)
		}
		if models.IsBatchReplaceSettled(status) {
			break
		}
		if time.Now().After(deadline) {
			log.Fatalf("Batch replace did not settle within %.1fs", settings.OrderTimeoutSec)
		}
		time.Sleep(time.Duration(settings.PollSec * float64(time.Second)))
	}

}

func priceInputPtr(s string) *models.PriceInput {
	p := models.PriceFromDecimal(s)
	return &p
}

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

	quoteAssetID := polyesterexamples.QuoteAssetID(client, pair, symbol)
	if quoteAssetID == nil {
		log.Fatalf("Could not resolve quote asset id for %s", symbol)
	}

	balances, err := client.Balances.List(ctx, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	availableQuote := polyesterexamples.AvailableTradingBalance(balances, *quoteAssetID)
	priceValue, err := strconv.ParseFloat(price, 64)
	if err != nil {
		log.Fatal(err)
	}
	qty, err := polyesterexamples.BuyQtyForQuoteCap(availableQuote, settings.MaxQuote, priceValue, pair)
	if err != nil {
		log.Fatal(err)
	}

	clientOrderID := polyesterexamples.UniqueClientOrderID("example-limit")
	fmt.Printf(
		"Creating post-only buy limit order: symbol=%s price=%s qty=%s client_order_id=%s\n",
		symbol, price, qty, clientOrderID,
	)

	defer polyesterexamples.CancelAllForSymbol(ctx, client, symbol)

	tif := "gtc"
	created, err := client.Orders.Create(ctx, models.CreateOrderRequest{
		Symbol:        &symbol,
		Side:          "buy",
		OrderType:     "limit",
		TIF:           &tif,
		Qty:           models.QtyFromDecimal(qty),
		Price:         priceInputPtr(price),
		PostOnly:      true,
		ClientOrderID: &clientOrderID,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created: status=%s order_id=%s\n", created.Status, created.OrderID)

	openOrder, err := polyesterexamples.WaitForOpenOrder(
		ctx, client, clientOrderID, 50, settings.OrderTimeoutSec, settings.PollSec,
	)
	if err != nil {
		fmt.Println(
			"Order create was accepted, but open-order reads did not catch up in time. " +
				"This is a known devnet OMS read-indexing issue — canceling by order_id anyway.",
		)
		fmt.Printf("  detail: %v\n", err)
	} else {
		fmt.Printf("Visible in open orders: status=%s\n", openOrder.Status)
	}

	canceled, err := client.Orders.Cancel(ctx, nil, &created.OrderID, &clientOrderID, &symbol, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cancel submitted: status=%s\n", canceled.Status)

	if err := polyesterexamples.WaitForNoOpenOrder(
		ctx, client, clientOrderID, 50, settings.OrderTimeoutSec, settings.PollSec,
	); err != nil {
		fmt.Println(
			"Cancel was submitted, but open-order reads still show nothing. " +
				"Check the Polyester UI or wait for the devnet read path to catch up.",
		)
	} else {
		fmt.Println("Order is no longer open")
	}
}

func priceInputPtr(s string) *models.PriceInput {
	p := models.PriceFromDecimal(s)
	return &p
}

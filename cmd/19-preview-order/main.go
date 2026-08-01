package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	sdkerrors "github.com/Fabric-Labs/polyester-sdk-go/errors"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
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
		qty = polyesterexamples.FormatQty(
			polyesterexamples.MinBaseQtyForNotional(pair, priceValue),
			pair,
		)
		fmt.Printf(
			"Insufficient quote trading balance (available=%s); previewing minimum qty=%s to exercise admission rejection\n",
			polyesterexamples.FormatDecimal(availableQuote),
			qty,
		)
	}

	tif := "gtc"
	priceInput := models.PriceFromDecimal(price)
	fmt.Printf(
		"Previewing post-only buy limit: symbol=%s price=%s qty=%s (no order is created)\n",
		symbol, price, qty,
	)
	preview, err := client.Orders.Preview(ctx, models.CreateOrderRequest{
		Symbol:    &symbol,
		Side:      "buy",
		OrderType: "limit",
		TIF:       &tif,
		Qty:       models.QtyFromDecimal(qty),
		Price:     &priceInput,
		PostOnly:  true,
	}, nil)
	if err != nil {
		var route *sdkerrors.RouteNotFoundError
		var api *sdkerrors.APIError
		if errors.As(err, &route) || (errors.As(err, &api) && (api.Code == "unimplemented" || api.Code == "not_found")) {
			fmt.Printf("PreviewOrder is not available on this API host (%v). Skipping.\n", err)
			return
		}
		log.Fatal(err)
	}

	if preview.Admissible != nil {
		fmt.Printf("  admissible=%v\n", *preview.Admissible)
	} else {
		fmt.Println("  admissible=<unset>")
	}
	fmt.Printf("  resolved_base_qty_scaled=%s\n", preview.ResolvedBaseQtyScaled)
	if preview.ProtectedPriceBound != nil {
		fmt.Printf("  protected_price_bound.ticks=%d\n", preview.ProtectedPriceBound.Ticks())
	}
	fmt.Printf("  evaluated_at_ms=%d\n", preview.EvaluatedAtMs)
	if preview.Rejection != nil {
		fmt.Printf("  rejection.code=%s\n", preview.Rejection.Code)
		for _, violation := range preview.Rejection.Violations {
			fmt.Printf(
				"  violation field=%s rule=%s message=%s\n",
				violation.FieldPath, violation.RuleID, violation.Message,
			)
		}
	}
}

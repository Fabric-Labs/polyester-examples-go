package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	"github.com/Fabric-Labs/polyester-sdk-go/codecs"
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

	limitPrice, err := polyesterexamples.ResolvePostOnlyBuyLimitPrice(ctx, client, symbol, pair)
	if err != nil {
		log.Fatal(err)
	}
	priceValue, err := strconv.ParseFloat(limitPrice, 64)
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
	qty, err := polyesterexamples.BuyQtyForQuoteCap(availableQuote, settings.MaxQuote, priceValue, pair)
	if err != nil {
		log.Fatal(err)
	}

	clientTriggerID := polyesterexamples.UniqueClientOrderID("trg-twap")
	durationMs := int64(600_000)
	sliceMs := int64(300_000)
	lp := models.PriceFromDecimal(limitPrice)

	fmt.Printf(
		"Creating TWAP trigger: symbol=%s side=buy qty=%s limit=%s duration_ms=%d slice_ms=%d\n",
		symbol, qty, limitPrice, durationMs, sliceMs,
	)

	created, err := client.Triggers.Create(
		ctx, nil, symbol, "twap", nil, "buy",
		models.QtyFromDecimal(qty), "limit", &lp, "", "",
		nil, &clientTriggerID, false,
		codecs.CreateTriggerOptions{
			TwapDurationMs:      &durationMs,
			TwapSliceIntervalMs: &sliceMs,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created: trigger_id=%s status=%s\n", created.TriggerID, created.Status)

	listed, err := client.Triggers.List(ctx, nil, nil, &symbol, nil, 50, nil)
	if err != nil {
		fmt.Printf("List warning: %v\n", err)
	} else {
		fmt.Printf("List: %d trigger(s) for %s\n", len(listed.Triggers), symbol)
		for _, t := range listed.Triggers {
			if t.TriggerID == created.TriggerID {
				fmt.Printf(
					"List match: trigger_id=%s type=%s side=%s status=%s parent_order_id=%s\n",
					t.TriggerID, t.TriggerType, t.Side, t.Status, t.ParentOrderID,
				)
				break
			}
		}
	}

	if created.TriggerID != "" {
		got, err := client.Triggers.Get(ctx, nil, created.TriggerID, nil)
		if err != nil {
			fmt.Printf("Get warning: %v\n", err)
		} else if got != nil {
			fmt.Printf(
				"Get: trigger_id=%s type=%s side=%s status=%s parent_order_id=%s\n",
				got.TriggerID, got.TriggerType, got.Side, got.Status, got.ParentOrderID,
			)
		}

		if _, err := client.Triggers.Cancel(ctx, nil, created.TriggerID, nil); err != nil {
			fmt.Printf("Trigger cancel warning: %v\n", err)
		} else {
			fmt.Println("Trigger canceled")
		}
	}

	if _, err := client.Orders.CancelAll(ctx, nil, nil, &symbol, nil, false, nil); err != nil {
		fmt.Printf("cancel_all cleanup warning: %v\n", err)
	} else {
		fmt.Println("Best-effort cancel_all cleanup submitted")
	}
}

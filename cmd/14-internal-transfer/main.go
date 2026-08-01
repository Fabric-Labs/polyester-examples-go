package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
)

func main() {
	settings := polyesterexamples.LoadSettings()
	if err := polyesterexamples.RequireTransfersEnabled(settings); err != nil {
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

	var assetID *uint32
	zipper, zipperErr := client.Zipper.GetDepositWithdrawConfig(ctx)
	if zipperErr == nil {
		if asset, ok := polyesterexamples.PickUSDTZipperAsset(zipper); ok {
			id := asset.LedgerID
			assetID = &id
		}
	}
	if assetID == nil {
		assetID = polyesterexamples.QuoteAssetID(client, pair, symbol)
	}
	if assetID == nil {
		log.Fatal("Could not resolve USDT/ledger or quote asset id for transfer")
	}

	amount := polyesterexamples.FormatDecimal(settings.TransferAmount)
	dest := settings.TransferDestAccountID
	idempotencyKey := polyesterexamples.UniqueClientOrderID("xfer")

	fmt.Printf(
		"Internal transfer: asset_id=%d amount=%s dest_account=%s\n",
		*assetID, amount, dest,
	)

	result, err := client.InternalTransfers.Create(
		ctx, *assetID, models.AssetAmountFromDecimal(amount), idempotencyKey,
		nil, nil, &dest, nil, nil, nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(
		"Transfer accepted: request_id=%s transfer_id=%s quantity.scaled=%v\n",
		result.RequestID, result.TransferID, result.Quantity.Scaled(),
	)
}

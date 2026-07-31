package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	"github.com/Fabric-Labs/polyester-sdk-go/codecs"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
	"github.com/Fabric-Labs/polyester-sdk-go/services"
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

	zipper, err := client.Zipper.GetDepositWithdrawConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	asset, ok := polyesterexamples.PickUSDTZipperAsset(zipper)
	if !ok {
		log.Fatal("USDT / ledger_id=1 not found in zipper deposit-withdraw config")
	}

	amount := polyesterexamples.FormatDecimal(settings.WithdrawAmount)
	idempotencyKey := polyesterexamples.UniqueClientOrderID("wd-prep")

	fmt.Printf(
		"Preparing API-key Trading→Funding withdraw: asset_id=%d amount=%s\n",
		asset.LedgerID, amount,
	)

	prepared, err := client.Withdraw.PrepareAPIKeyToFunding(services.PrepareAPIKeyWithdrawParams{
		AssetID:        asset.LedgerID,
		Amount:         models.AssetAmountFromDecimal(amount),
		IdempotencyKey: idempotencyKey,
		AmountScale:    codecs.LedgerScale,
	})
	if err != nil {
		log.Fatal(err)
	}
	payload := prepared.Payload()
	fmt.Printf(
		"Prepared: asset_id=%d deadline_ts_sec=%d idempotency_key=%s request_bytes=%d\n",
		payload.GetAssetId(), payload.GetDeadlineTsSec(), payload.GetIdempotencyKey(), len(prepared.RequestBytes()),
	)

	if !settings.EnableWithdrawals {
		fmt.Printf(
			"Submit skipped (set %s=1 to call SubmitPrepared)\n",
			polyesterexamples.ExamplesEnableWithdrawalsEnv,
		)
		return
	}

	if err := polyesterexamples.RequireWithdrawalsEnabled(settings); err != nil {
		log.Fatal(err)
	}
	result, err := client.Withdraw.SubmitPrepared(ctx, prepared)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(
		"Submitted: intent_id=%s status=%s flow_id=%s\n",
		result.IntentID, result.Status, result.FlowID,
	)
}

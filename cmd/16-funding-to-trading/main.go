package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	"github.com/Fabric-Labs/polyester-sdk-go/chain"
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

	qty := polyesterexamples.HumanAmountToE18(settings.ChainAmount)
	if qty.Sign() <= 0 {
		log.Fatalf("invalid %s=%v", polyesterexamples.ExamplesChainAmountEnv, settings.ChainAmount)
	}

	call, err := chain.EncodeTradingGatewayDeposit(
		chain.PolyesterTestnetEnvironment.Contracts.TradingGatewayAddress,
		asset.UAssetID,
		qty,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Encoded TradingGateway.deposit: to=%s u_asset_id=%s qty_e18=%s\n",
		call.To, asset.UAssetID, qty.String(),
	)
	fmt.Printf("calldata=0x%s\n", hex.EncodeToString(call.Data))

	if !settings.EnableChainFundingToTrading {
		fmt.Printf(
			"Submit skipped (set %s=1 and %s to SendCalls)\n",
			polyesterexamples.ExamplesEnableChainFundingToTradingEnv,
			polyesterexamples.OwnerPrivateKeyEnv,
		)
		return
	}

	if err := polyesterexamples.RequireChainFundingToTradingEnabled(settings); err != nil {
		log.Fatal(err)
	}
	if err := polyesterexamples.RequireOwnerPrivateKey(settings); err != nil {
		log.Fatal(err)
	}

	account, err := chain.NewSmartAccount(settings.OwnerPrivateKey, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Smart account: %s (owner=%s)\n", account.Address, account.OwnerAddress)

	receipt, err := account.SendCalls([]chain.ChainCall{call}, true, 120*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	if receipt == nil {
		log.Fatal("SendCalls returned nil receipt")
	}
	fmt.Printf(
		"UserOp receipt: success=%v user_operation_hash=%s tx=%s\n",
		receipt.Success, receipt.UserOperationHash, receipt.TransactionHash,
	)
}

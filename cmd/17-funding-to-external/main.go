package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	"github.com/Fabric-Labs/polyester-sdk-go/chain"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
)

func main() {
	settings := polyesterexamples.LoadSettings()
	if settings.ExternalDestination == "" {
		log.Fatalf(
			"%s is required to encode Funding→external withdraw",
			polyesterexamples.ExamplesExternalDestinationEnv,
		)
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

	zipper, err := client.Zipper.GetDepositWithdrawConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	asset, ok := polyesterexamples.PickUSDTZipperAsset(zipper)
	if !ok {
		log.Fatal("USDT / ledger_id=1 not found in zipper deposit-withdraw config")
	}

	chainID := uint16(settings.ExternalChainID)
	if chainID == 0 {
		log.Fatalf("invalid %s=%d", polyesterexamples.ExamplesExternalChainIDEnv, settings.ExternalChainID)
	}

	var variant *models.ZipperAssetChainVariant
	for i := range asset.Variants {
		if asset.Variants[i].ChainID == uint32(chainID) && asset.Variants[i].ZToken.Address != "" {
			variant = &asset.Variants[i]
			break
		}
	}
	if variant == nil {
		log.Fatalf("No USDT z_token variant for withdraw chain_id=%d", chainID)
	}

	caseSensitive := false
	for _, c := range zipper.Chains {
		if c.ChainID == uint32(chainID) {
			caseSensitive = c.IsCaseSensitive
			break
		}
	}

	zAmount := polyesterexamples.HumanAmountToE18(settings.ChainAmount)
	if zAmount.Sign() <= 0 {
		log.Fatalf("invalid %s=%v", polyesterexamples.ExamplesChainAmountEnv, settings.ChainAmount)
	}

	fee, err := chain.QuoteZipperFee(
		chainID,
		variant.ZToken.Address,
		chain.PolyesterTestnetEnvironment.Contracts.ZipperEndpointAddress,
		nil,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	maxFee := new(big.Int).Add(fee.Fee, new(big.Int).Div(fee.Fee, big.NewInt(10)))
	if zAmount.Cmp(maxFee) <= 0 {
		log.Fatalf(
			"chain amount %s must be greater than max_fee %s; raise %s",
			zAmount, maxFee, polyesterexamples.ExamplesChainAmountEnv,
		)
	}

	destBytes := chain.EncodeWithdrawDestination(settings.ExternalDestination, caseSensitive)
	call, err := chain.EncodeFundingWithdrawToChain(
		chain.PolyesterTestnetEnvironment.Contracts.FundingAccountAddress,
		chainID,
		variant.ZToken.Address,
		destBytes,
		zAmount,
		maxFee,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Encoded FundingAccount.withdrawToChain: to=%s chain_id=%d z_token=%s z_amount=%s max_fee=%s dest=%s\n",
		call.To, chainID, variant.ZToken.Address, zAmount.String(), maxFee.String(), settings.ExternalDestination,
	)
	fmt.Printf("calldata=0x%s\n", hex.EncodeToString(call.Data))

	if !settings.EnableChainExternalSubmit {
		fmt.Printf(
			"Submit skipped (set %s=1 and %s to SendCalls)\n",
			polyesterexamples.ExamplesEnableChainExternalSubmitEnv,
			polyesterexamples.OwnerPrivateKeyEnv,
		)
		return
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

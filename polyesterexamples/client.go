package polyesterexamples

import (
	"context"
	"fmt"

	polyester "github.com/Fabric-Labs/polyester-sdk-go"
)

// NewClient creates a Polyester client from explicit example configuration.
func NewClient(cfg ClientConfig) (*polyester.Client, error) {
	clientCfg := polyester.Config{
		APIKeyID:        cfg.APIKeyID,
		APIPrivateKey:   cfg.APIPrivateKey,
		HydrateCatalogs: false,
	}
	if cfg.APIURL != "" {
		clientCfg.APIURL = cfg.APIURL
	}
	if cfg.WSURL != "" {
		clientCfg.WSURL = cfg.WSURL
	}
	if cfg.DefaultAccountID != "" {
		accountID := cfg.DefaultAccountID
		clientCfg.DefaultAccountID = &accountID
	}
	if cfg.DefaultSubAccountID != "" {
		subAccountID := cfg.DefaultSubAccountID
		clientCfg.DefaultSubAccountID = &subAccountID
	}
	return polyester.New(clientCfg)
}

// WaitForCatalogs hydrates spot and zipper catalogs before examples use them.
func WaitForCatalogs(ctx context.Context, client *polyester.Client) error {
	spot, err := client.MarketData.GetSpotConfig(ctx)
	if err != nil {
		return fmt.Errorf("get spot config: %w", err)
	}
	client.Catalogs.HydrateSpotConfig(spot.Raw)
	zipper, err := client.Zipper.GetDepositWithdrawConfig(ctx)
	if err == nil {
		client.Catalogs.HydrateZipperConfig(zipper)
	}
	return nil
}

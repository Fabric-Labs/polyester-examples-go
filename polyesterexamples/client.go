package polyesterexamples

import (
	"context"

	polyester "github.com/Fabric-Labs/polyester-sdk-go"
)

// NewClient creates a Polyester client from explicit example configuration.
func NewClient(cfg ClientConfig) (*polyester.Client, error) {
	clientCfg := polyester.Config{
		APIKeyID:        cfg.APIKeyID,
		APIPrivateKey:   cfg.APIPrivateKey,
		HydrateCatalogs: true,
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

// WaitForCatalogs waits for the SDK's best-effort background catalog hydration.
func WaitForCatalogs(ctx context.Context, client *polyester.Client) error {
	return client.WaitForCatalogs(ctx)
}

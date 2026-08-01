package polyesterexamples_test

import (
	"os"
	"testing"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
)

func TestLoadSettingsReadsExampleEnv(t *testing.T) {
	t.Setenv("POLYESTER_EXAMPLES_SYMBOL", "BTC-USDT")
	t.Setenv("POLYESTER_EXAMPLES_ENABLE_TRADING", "true")
	t.Setenv("POLYESTER_EXAMPLES_MAX_QUOTE", "25.5")
	t.Setenv("POLYESTER_EXAMPLES_ENABLE_TRANSFERS", "1")
	t.Setenv("POLYESTER_EXAMPLES_TRANSFER_DEST_ACCOUNT_ID", "destAccount")
	t.Setenv("POLYESTER_EXAMPLES_TRANSFER_AMOUNT", "0.02")
	t.Setenv("POLYESTER_EXAMPLES_WITHDRAW_AMOUNT", "0.03")
	t.Setenv("POLYESTER_EXAMPLES_CHAIN_AMOUNT", "2")
	t.Setenv("POLYESTER_EXAMPLES_EXTERNAL_CHAIN_ID", "6")
	t.Setenv("POLYESTER_EXAMPLES_EXTERNAL_DESTINATION", "0xabc")
	t.Setenv("POLYESTER_OWNER_PRIVATE_KEY", "0xdead")

	settings := polyesterexamples.LoadSettings()

	if settings.Symbol != "BTC-USDT" {
		t.Fatalf("symbol=%q want BTC-USDT", settings.Symbol)
	}
	if !settings.EnableTrading {
		t.Fatal("expected enable_trading=true")
	}
	if settings.MaxQuote != 25.5 {
		t.Fatalf("max_quote=%v want 25.5", settings.MaxQuote)
	}
	if !settings.EnableTransfers {
		t.Fatal("expected enable_transfers=true")
	}
	if settings.TransferDestAccountID != "destAccount" {
		t.Fatalf("transfer dest=%q", settings.TransferDestAccountID)
	}
	if settings.TransferAmount != 0.02 || settings.WithdrawAmount != 0.03 || settings.ChainAmount != 2 {
		t.Fatalf("amounts transfer=%v withdraw=%v chain=%v", settings.TransferAmount, settings.WithdrawAmount, settings.ChainAmount)
	}
	if settings.ExternalDestination != "0xabc" || settings.ExternalChainID != 6 {
		t.Fatalf("external dest=%q chain=%d", settings.ExternalDestination, settings.ExternalChainID)
	}
	if settings.OwnerPrivateKey != "0xdead" {
		t.Fatalf("owner key=%q", settings.OwnerPrivateKey)
	}
}

func TestRequireTransfersEnabledNeedsDest(t *testing.T) {
	err := polyesterexamples.RequireTransfersEnabled(polyesterexamples.Settings{EnableTransfers: true})
	if err == nil {
		t.Fatal("expected missing dest account error")
	}
	err = polyesterexamples.RequireTransfersEnabled(polyesterexamples.Settings{
		EnableTransfers:       true,
		TransferDestAccountID: "RLxqJGUDg92",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPickSymbolPrefersConfiguredSymbolWhenAvailable(t *testing.T) {
	spot := map[string]any{
		"pairs": []any{
			map[string]any{"symbol": "ETH-USDT"},
			map[string]any{"symbol": "BTC-USDT"},
		},
	}

	if got := polyesterexamples.PickSymbol(spot, "BTC-USDT"); got != "BTC-USDT" {
		t.Fatalf("symbol=%q want BTC-USDT", got)
	}
}

func TestPairForSymbolRaisesForUnknownSymbol(t *testing.T) {
	_, err := polyesterexamples.PairForSymbol(
		map[string]any{"pairs": []any{map[string]any{"symbol": "ETH-USDT"}}},
		"DOGE-USDT",
	)
	if err == nil {
		t.Fatal("expected error for unknown symbol")
	}
}

func TestFormatQtyAlignsToStepSize(t *testing.T) {
	pair := map[string]any{
		"stepSize": "0.00001",
	}
	if got := polyesterexamples.FormatQty(0.00015000000000000001, pair); got != "0.00015" {
		t.Fatalf("qty=%q want 0.00015", got)
	}
}

func TestBuyQtyForQuoteCapSatisfiesMinNotional(t *testing.T) {
	pair := map[string]any{
		"symbol":           "ETH-USDT",
		"tickSize":         "0.01",
		"stepSize":         "0.001",
		"minNotionalQuote": "10",
		"minQtyBase":       "0.001",
	}

	qty, err := polyesterexamples.BuyQtyForQuoteCap(20, 10, 123.45, pair)
	if err != nil {
		t.Fatal(err)
	}
	if qty != "0.082" {
		t.Fatalf("qty=%q want 0.082", qty)
	}
}

func TestBuyQtyForQuoteCapRequiresMinNotional(t *testing.T) {
	pair := map[string]any{
		"stepSize":         "0.001",
		"minNotionalQuote": "10",
	}
	_, err := polyesterexamples.BuyQtyForQuoteCap(5, 10, 100, pair)
	if err == nil {
		t.Fatal("expected min notional error")
	}
}

func TestPriceHelpersAlignToTickSize(t *testing.T) {
	pair := map[string]any{
		"tickSize": "0.01",
		"stepSize": "0.001",
	}

	if got := polyesterexamples.FarBelowMarketPrice(2500.123, pair); got != "2487.62" {
		t.Fatalf("far below=%q want 2487.62", got)
	}
	buy, err := polyesterexamples.MarketableLimitPrice("buy", 100, pair)
	if err != nil || buy != "100.1" {
		t.Fatalf("buy price=%q err=%v want 100.1", buy, err)
	}
	sell, err := polyesterexamples.MarketableLimitPrice("sell", 100, pair)
	if err != nil || sell != "99.9" {
		t.Fatalf("sell price=%q err=%v want 99.9", sell, err)
	}
}

func TestFormatDecimalTrimsTrailingZeroes(t *testing.T) {
	if got := polyesterexamples.FormatDecimal(10.5); got != "10.5" {
		t.Fatalf("format=%q want 10.5", got)
	}
}

func TestClientConfigFromEnvIgnoresPlaceholderCredentials(t *testing.T) {
	t.Setenv(polyesterexamples.APIKeyIDEnv, "ak_your_key_id_here")
	t.Setenv(polyesterexamples.APIPrivateKeyEnv, "your_64_char_ed25519_private_key_hex")
	t.Setenv(polyesterexamples.AccountIDEnv, "your_account_id_here")

	cfg, err := polyesterexamples.ClientConfigFromEnv(false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKeyID != "" || cfg.APIPrivateKey != "" || cfg.DefaultAccountID != "" {
		t.Fatalf("expected placeholder credentials to be ignored: %+v", cfg)
	}
}

func TestClientConfigFromEnvRejectsInvalidPrivateKeyWhenAuthRequired(t *testing.T) {
	t.Setenv(polyesterexamples.APIKeyIDEnv, "ak_test")
	t.Setenv(polyesterexamples.APIPrivateKeyEnv, "not-valid-hex")
	t.Setenv(polyesterexamples.AccountIDEnv, "RLxqJGUDg92")

	_, err := polyesterexamples.ClientConfigFromEnv(true)
	if err == nil {
		t.Fatal("expected invalid private key error")
	}
}

func TestLoadDotenvDoesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	if err := os.WriteFile(path, []byte("POLYESTER_EXAMPLES_SYMBOL=FROM_FILE\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POLYESTER_EXAMPLES_SYMBOL", "FROM_ENV")
	polyesterexamples.LoadDotenv(path)
	if os.Getenv("POLYESTER_EXAMPLES_SYMBOL") != "FROM_ENV" {
		t.Fatalf("env=%q want FROM_ENV", os.Getenv("POLYESTER_EXAMPLES_SYMBOL"))
	}
}

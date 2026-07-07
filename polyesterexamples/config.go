package polyesterexamples

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Fabric-Labs/polyester-sdk-go/auth"
)

const (
	APIKeyIDEnv      = "POLYESTER_API_KEY_ID"
	APIPrivateKeyEnv = "POLYESTER_API_PRIVATE_KEY"
	AccountIDEnv     = "POLYESTER_ACCOUNT_ID"
	SubAccountIDEnv  = "POLYESTER_SUB_ACCOUNT_ID"
	APIURLEnv        = "POLYESTER_API_URL"
	WSURLEnv         = "POLYESTER_WS_URL"

	ExamplesSymbolEnv         = "POLYESTER_EXAMPLES_SYMBOL"
	ExamplesTimeframeEnv      = "POLYESTER_EXAMPLES_TIMEFRAME"
	ExamplesCandleLimitEnv    = "POLYESTER_EXAMPLES_CANDLE_LIMIT"
	ExamplesEnableTradingEnv  = "POLYESTER_EXAMPLES_ENABLE_TRADING"
	ExamplesMaxQuoteEnv       = "POLYESTER_EXAMPLES_MAX_QUOTE"
	ExamplesRSIPeriodEnv      = "POLYESTER_EXAMPLES_RSI_PERIOD"
	ExamplesRSIOversoldEnv    = "POLYESTER_EXAMPLES_RSI_OVERSOLD"
	ExamplesRSIOverboughtEnv  = "POLYESTER_EXAMPLES_RSI_OVERBOUGHT"
	ExamplesOrderTimeoutEnv   = "POLYESTER_EXAMPLES_ORDER_TIMEOUT_SEC"
	ExamplesPollEnv           = "POLYESTER_EXAMPLES_POLL_SEC"
	ExamplesStreamCountEnv    = "POLYESTER_EXAMPLES_STREAM_COUNT"
	ExamplesOrderbookDepthEnv = "POLYESTER_EXAMPLES_ORDERBOOK_DEPTH"
)

// Settings holds example runtime knobs loaded from the environment.
type Settings struct {
	Symbol          string
	Timeframe       string
	CandleLimit     int
	EnableTrading   bool
	MaxQuote        float64
	RSIPeriod       int
	RSIOversold     float64
	RSIOverbought   float64
	OrderTimeoutSec float64
	PollSec         float64
	StreamCount     int
	OrderbookDepth  int
}

// ClientConfig is explicit Polyester client configuration for examples.
type ClientConfig struct {
	APIKeyID            string
	APIPrivateKey       string
	APIURL              string
	WSURL               string
	DefaultAccountID    string
	DefaultSubAccountID string
}

// LoadDotenv loads a simple KEY=VALUE .env file without adding a runtime dependency.
func LoadDotenv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key != "" {
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
	}
}

// LoadSettings reads example settings from the environment.
func LoadSettings() Settings {
	LoadDotenv(".env")
	symbol := strings.TrimSpace(os.Getenv(ExamplesSymbolEnv))
	if symbol == "" {
		symbol = "BTC-USDT"
	}
	timeframe := strings.TrimSpace(os.Getenv(ExamplesTimeframeEnv))
	if timeframe == "" {
		timeframe = "1m"
	}
	return Settings{
		Symbol:          symbol,
		Timeframe:       timeframe,
		CandleLimit:     EnvInt(ExamplesCandleLimitEnv, 100),
		EnableTrading:   EnvBool(ExamplesEnableTradingEnv),
		MaxQuote:        EnvFloat(ExamplesMaxQuoteEnv, 10),
		RSIPeriod:       EnvInt(ExamplesRSIPeriodEnv, 14),
		RSIOversold:     EnvFloat(ExamplesRSIOversoldEnv, 30),
		RSIOverbought:   EnvFloat(ExamplesRSIOverboughtEnv, 70),
		OrderTimeoutSec: EnvFloat(ExamplesOrderTimeoutEnv, 15),
		PollSec:         EnvFloat(ExamplesPollEnv, 0.5),
		StreamCount:     EnvInt(ExamplesStreamCountEnv, 5),
		OrderbookDepth:  EnvInt(ExamplesOrderbookDepthEnv, 50),
	}
}

// ClientConfigFromEnv builds explicit client configuration for examples.
func ClientConfigFromEnv(requireAuth bool) (ClientConfig, error) {
	LoadDotenv(".env")
	cfg := ClientConfig{
		APIURL:              strings.TrimSpace(os.Getenv(APIURLEnv)),
		WSURL:               strings.TrimSpace(os.Getenv(WSURLEnv)),
		DefaultAccountID:    strings.TrimSpace(os.Getenv(AccountIDEnv)),
		DefaultSubAccountID: strings.TrimSpace(os.Getenv(SubAccountIDEnv)),
	}
	keyID := strings.TrimSpace(os.Getenv(APIKeyIDEnv))
	privateKey := strings.TrimSpace(os.Getenv(APIPrivateKeyEnv))
	if keyID != "" && privateKey != "" {
		if _, err := auth.NormalizePrivateKey(privateKey); err != nil {
			if requireAuth {
				return ClientConfig{}, fmt.Errorf(
					"%s is not a valid 64-char Ed25519 hex seed (copy real values from the Polyester app or polyester-sdk-go/.env): %w",
					APIPrivateKeyEnv,
					err,
				)
			}
		} else if !looksLikePlaceholderCredential(keyID) && !looksLikePlaceholderCredential(privateKey) {
			cfg.APIKeyID = keyID
			cfg.APIPrivateKey = privateKey
		}
	}
	if cfg.APIKeyID == "" && cfg.APIPrivateKey == "" && requireAuth {
		return ClientConfig{}, fmt.Errorf(
			"%s and %s are required for this example; copy real values into .env (see .env.example)",
			APIKeyIDEnv,
			APIPrivateKeyEnv,
		)
	}
	if cfg.DefaultAccountID != "" && looksLikePlaceholderCredential(cfg.DefaultAccountID) {
		cfg.DefaultAccountID = ""
	}
	if cfg.DefaultAccountID == "" && requireAuth {
		return ClientConfig{}, fmt.Errorf("%s is required for this example", AccountIDEnv)
	}
	return cfg, nil
}

func looksLikePlaceholderCredential(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	return strings.Contains(lower, "your_") ||
		strings.Contains(lower, "_here") ||
		strings.Contains(lower, "replace_me") ||
		strings.Contains(lower, "changeme")
}

// RequireTradingEnabled ensures live order writes were explicitly opted in.
func RequireTradingEnabled(settings Settings) error {
	if !settings.EnableTrading {
		return fmt.Errorf(
			"live order writes are disabled; set %s=1 after confirming your API key policy and trading balance",
			ExamplesEnableTradingEnv,
		)
	}
	return nil
}

func EnvBool(name string) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func EnvInt(name string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return value
}

func EnvFloat(name string, defaultValue float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

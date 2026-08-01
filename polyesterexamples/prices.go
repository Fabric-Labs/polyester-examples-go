package polyesterexamples

import (
	"context"
	"strconv"

	polyester "github.com/Fabric-Labs/polyester-sdk-go"
)

// LastReferencePrice returns a decimal reference price from recent candles.
func LastReferencePrice(ctx context.Context, client *polyester.Client, symbol, timeframe string) float64 {
	candles, err := client.MarketData.GetCandles(ctx, &symbol, nil, timeframe, 1, nil, nil, true)
	if err == nil && len(candles.Candles) > 0 {
		// Row candles are newest-first; an included open candle is prepended.
		closePrice := candles.Candles[0].Close
		if value, err := strconv.ParseFloat(closePrice, 64); err == nil && value > 0 {
			return value
		}
	}
	return 100
}

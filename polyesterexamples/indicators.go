package polyesterexamples

import "fmt"

// RsiSignal is the output of a simple RSI threshold strategy.
type RsiSignal struct {
	LatestRSI   *float64
	PreviousRSI *float64
	Action      string
	Reason      string
}

// CalculateRSI returns Wilder RSI values aligned to the input close series.
func CalculateRSI(closes []float64, period int) ([]*float64, error) {
	if period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}
	values := make([]*float64, len(closes))
	if len(closes) < period+1 {
		return values, nil
	}

	gains := make([]float64, 0, period)
	losses := make([]float64, 0, period)
	for index := 1; index <= period; index++ {
		change := closes[index] - closes[index-1]
		gains = append(gains, maxFloat(change, 0))
		losses = append(losses, maxFloat(-change, 0))
	}

	avgGain := average(gains)
	avgLoss := average(losses)
	rsi := rsiFromAverages(avgGain, avgLoss)
	values[period] = &rsi

	for index := period + 1; index < len(closes); index++ {
		change := closes[index] - closes[index-1]
		gain := maxFloat(change, 0)
		loss := maxFloat(-change, 0)
		avgGain = ((avgGain * float64(period-1)) + gain) / float64(period)
		avgLoss = ((avgLoss * float64(period-1)) + loss) / float64(period)
		next := rsiFromAverages(avgGain, avgLoss)
		values[index] = &next
	}
	return values, nil
}

// EvaluateRSISignal evaluates threshold crossings on the latest RSI values.
func EvaluateRSISignal(closes []float64, period int, oversold, overbought float64) (RsiSignal, error) {
	values, err := CalculateRSI(closes, period)
	if err != nil {
		return RsiSignal{}, err
	}
	latest := latestNonNil(values)
	previous := previousNonNil(values)
	if latest == nil || previous == nil {
		return RsiSignal{LatestRSI: latest, PreviousRSI: previous, Action: "hold", Reason: "not enough candle history"}, nil
	}
	if *previous <= oversold && *latest > oversold {
		return RsiSignal{LatestRSI: latest, PreviousRSI: previous, Action: "buy", Reason: "RSI crossed up out of oversold"}, nil
	}
	if *previous >= overbought && *latest < overbought {
		return RsiSignal{LatestRSI: latest, PreviousRSI: previous, Action: "sell", Reason: "RSI crossed down out of overbought"}, nil
	}
	return RsiSignal{LatestRSI: latest, PreviousRSI: previous, Action: "hold", Reason: "RSI has not crossed a threshold"}, nil
}

func rsiFromAverages(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		if avgGain == 0 {
			return 50
		}
		return 100
	}
	relativeStrength := avgGain / avgLoss
	return 100 - (100 / (1 + relativeStrength))
}

func latestNonNil(values []*float64) *float64 {
	for index := len(values) - 1; index >= 0; index-- {
		if values[index] != nil {
			return values[index]
		}
	}
	return nil
}

func previousNonNil(values []*float64) *float64 {
	seenLatest := false
	for index := len(values) - 1; index >= 0; index-- {
		if values[index] == nil {
			continue
		}
		if seenLatest {
			return values[index]
		}
		seenLatest = true
	}
	return nil
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

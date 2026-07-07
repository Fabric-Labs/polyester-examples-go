package polyesterexamples_test

import (
	"testing"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
)

func TestCalculateRSIReturnsValuesAlignedToCloses(t *testing.T) {
	closes := []float64{100, 90, 80, 85}

	values, err := polyesterexamples.CalculateRSI(closes, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != len(closes) {
		t.Fatalf("len=%d want %d", len(values), len(closes))
	}
	if values[0] != nil || values[1] != nil {
		t.Fatal("expected nil RSI for first two values")
	}
	if values[2] == nil || *values[2] != 0 {
		t.Fatalf("values[2]=%v want 0", values[2])
	}
	if values[3] == nil || *values[3] <= 33 || *values[3] >= 34 {
		t.Fatalf("values[3]=%v want between 33 and 34", values[3])
	}
}

func TestRsiBuySignalCrossesUpOutOfOversold(t *testing.T) {
	signal, err := polyesterexamples.EvaluateRSISignal(
		[]float64{100, 90, 80, 85},
		2,
		30,
		70,
	)
	if err != nil {
		t.Fatal(err)
	}
	if signal.Action != "buy" {
		t.Fatalf("action=%q want buy", signal.Action)
	}
}

func TestRsiSellSignalCrossesDownOutOfOverbought(t *testing.T) {
	signal, err := polyesterexamples.EvaluateRSISignal(
		[]float64{100, 110, 120, 115},
		2,
		30,
		70,
	)
	if err != nil {
		t.Fatal(err)
	}
	if signal.Action != "sell" {
		t.Fatalf("action=%q want sell", signal.Action)
	}
}

func TestRsiHoldsWithoutEnoughHistory(t *testing.T) {
	signal, err := polyesterexamples.EvaluateRSISignal([]float64{100, 101}, 14, 30, 70)
	if err != nil {
		t.Fatal(err)
	}
	if signal.Action != "hold" {
		t.Fatalf("action=%q want hold", signal.Action)
	}
	if signal.LatestRSI != nil {
		t.Fatal("expected latest RSI to be nil")
	}
}

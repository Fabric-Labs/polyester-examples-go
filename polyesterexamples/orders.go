package polyesterexamples

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	polyester "github.com/Fabric-Labs/polyester-sdk-go"
	sdkerrors "github.com/Fabric-Labs/polyester-sdk-go/errors"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
)

var openOrderStatuses = map[string]struct{}{
	"": {}, "pending": {}, "working": {}, "pending_cancel": {},
}

var terminalOrderStatuses = map[string]struct{}{
	"canceled": {}, "rejected": {}, "filled": {},
}

// UniqueClientOrderID returns a unique client order id for examples (≤36 chars).
func UniqueClientOrderID(prefix string) string {
	if prefix == "" {
		prefix = "example"
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	maxPrefix := 36 - 1 - len(suffix)
	if maxPrefix < 1 {
		if len(suffix) > 36 {
			return suffix[len(suffix)-36:]
		}
		return suffix
	}
	if len(prefix) > maxPrefix {
		prefix = prefix[:maxPrefix]
	}
	return fmt.Sprintf("%s-%s", prefix, suffix)
}

// WaitForOpenOrder polls until an order is visible as open or terminal.
func WaitForOpenOrder(
	ctx context.Context,
	client *polyester.Client,
	clientOrderID string,
	limit int,
	timeoutSec, pollSec float64,
) (models.Order, error) {
	if limit <= 0 {
		limit = 50
	}
	if pollSec <= 0 {
		pollSec = 0.5
	}
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	deadline := time.Now().Add(time.Duration(timeoutSec * float64(time.Second)))
	var lastStatus string
	for time.Now().Before(deadline) {
		order, ok, err := GetOrderOrNone(ctx, client, clientOrderID)
		if err != nil {
			return models.Order{}, err
		}
		if ok {
			status := strings.ToLower(strings.TrimSpace(order.Status))
			if _, isOpen := openOrderStatuses[status]; isOpen {
				return order, nil
			}
			lastStatus = status
			if _, terminal := terminalOrderStatuses[status]; terminal {
				return order, nil
			}
		}
		openOrders, err := client.Orders.ListOpen(ctx, nil, nil, nil, &limit, false, false)
		if err == nil {
			for _, openOrder := range openOrders.Orders {
				if openOrder.ClientOrderID == clientOrderID {
					return openOrder, nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return models.Order{}, ctx.Err()
		case <-time.After(time.Duration(pollSec * float64(time.Second))):
		}
	}
	msg := fmt.Sprintf("order %s was not visible as open within %.0fs", clientOrderID, timeoutSec)
	if lastStatus != "" {
		msg += fmt.Sprintf(" (last status=%s)", lastStatus)
	}
	return models.Order{}, errors.New(msg)
}

// WaitForNoOpenOrder polls until an order is no longer open.
func WaitForNoOpenOrder(
	ctx context.Context,
	client *polyester.Client,
	clientOrderID string,
	limit int,
	timeoutSec, pollSec float64,
) error {
	if limit <= 0 {
		limit = 50
	}
	if pollSec <= 0 {
		pollSec = 0.5
	}
	if timeoutSec <= 0 {
		timeoutSec = 10
	}
	deadline := time.Now().Add(time.Duration(timeoutSec * float64(time.Second)))
	for time.Now().Before(deadline) {
		openOrders, err := client.Orders.ListOpen(ctx, nil, nil, nil, &limit, false, false)
		if err == nil {
			stillOpen := false
			for _, order := range openOrders.Orders {
				if order.ClientOrderID == clientOrderID {
					stillOpen = true
					break
				}
			}
			if !stillOpen {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(pollSec * float64(time.Second))):
		}
	}
	return fmt.Errorf("order %s was still open after %.0fs", clientOrderID, timeoutSec)
}

// GetOrderOrNone fetches an order by client order id, returning false when not found.
func GetOrderOrNone(ctx context.Context, client *polyester.Client, clientOrderID string) (models.Order, bool, error) {
	detail, err := client.Orders.Get(ctx, nil, models.OrderKeyByClientID(clientOrderID), nil, false, false)
	if err != nil {
		if isNotFound(err) {
			return models.Order{}, false, nil
		}
		return models.Order{}, false, err
	}
	if detail.Order == nil {
		return models.Order{}, false, nil
	}
	return *detail.Order, true, nil
}

// OpenOrdersWithPrefix returns open orders whose client order id starts with prefix.
func OpenOrdersWithPrefix(ctx context.Context, client *polyester.Client, prefix string, limit int) ([]models.Order, error) {
	if limit <= 0 {
		limit = 100
	}
	openOrders, err := client.Orders.ListOpen(ctx, nil, nil, nil, &limit, false, false)
	if err != nil {
		return nil, err
	}
	out := make([]models.Order, 0)
	for _, order := range openOrders.Orders {
		if strings.HasPrefix(order.ClientOrderID, prefix) {
			out = append(out, order)
		}
	}
	return out, nil
}

// EnsureNoOpenOrdersWithPrefix refuses to proceed when bot orders are still open.
func EnsureNoOpenOrdersWithPrefix(ctx context.Context, client *polyester.Client, prefix string) error {
	matches, err := OpenOrdersWithPrefix(ctx, client, prefix, 100)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return nil
	}
	ids := make([]string, 0, len(matches))
	for _, order := range matches {
		ids = append(ids, order.ClientOrderID)
	}
	return fmt.Errorf("refusing to place a new order; open bot orders exist: %s", strings.Join(ids, ", "))
}

// CancelAfterTimeout cancels an order when it has not reached a terminal state in time.
func CancelAfterTimeout(
	ctx context.Context,
	client *polyester.Client,
	clientOrderID, symbol string,
	timeoutSec, pollSec float64,
) (string, error) {
	if pollSec <= 0 {
		pollSec = 0.5
	}
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	deadline := time.Now().Add(time.Duration(timeoutSec * float64(time.Second)))
	for time.Now().Before(deadline) {
		order, ok, err := GetOrderOrNone(ctx, client, clientOrderID)
		if err != nil {
			return "", err
		}
		if ok {
			status := strings.ToLower(strings.TrimSpace(order.Status))
			if _, terminal := terminalOrderStatuses[status]; terminal {
				return status, nil
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(pollSec * float64(time.Second))):
		}
	}

	_, err := client.Orders.Cancel(ctx, nil, models.OrderKeyByClientID(clientOrderID), &symbol, nil, nil)
	if err != nil {
		return "", err
	}
	confirmTimeout := mathMax(5, pollSec*4)
	if err := WaitForNoOpenOrder(ctx, client, clientOrderID, 50, confirmTimeout, pollSec); err != nil {
		return "canceled_after_timeout_unconfirmed", nil
	}
	return "canceled_after_timeout", nil
}

// CancelOwnedOrdersWithPrefix cancels only open orders owned by the named bot
// prefix. A concurrent fill/cancel that makes an order not_found is successful
// cleanup; other cancellation failures are returned.
func CancelOwnedOrdersWithPrefix(ctx context.Context, client *polyester.Client, prefix string) error {
	if strings.TrimSpace(prefix) == "" {
		return errors.New("cleanup prefix must not be empty")
	}
	matches, err := OpenOrdersWithPrefix(ctx, client, prefix, 100)
	if err != nil {
		return err
	}
	var cleanupErrs []error
	for _, order := range matches {
		clientOrderID := order.ClientOrderID
		symbolID := order.SymbolID
		_, err := client.Orders.Cancel(ctx, nil, models.OrderKeyByClientID(clientOrderID), nil, &symbolID, nil)
		if err == nil || isNotFound(err) {
			continue
		}
		cleanupErrs = append(cleanupErrs, fmt.Errorf("cancel owned order %q: %w", clientOrderID, err))
	}
	return errors.Join(cleanupErrs...)
}

func isNotFound(err error) bool {
	var api *sdkerrors.APIError
	if errors.As(err, &api) {
		return strings.EqualFold(strings.TrimSpace(api.Code), "not_found")
	}
	return false
}

func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
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

	if client.DefaultAccountID == nil || *client.DefaultAccountID == "" {
		log.Fatal("POLYESTER_ACCOUNT_ID is required for private realtime")
	}
	accountID := *client.DefaultAccountID

	fmt.Printf(
		"Subscribing to private orders + balances for %s (up to %d events each, 30s timeout)\n",
		accountID, settings.StreamCount,
	)

	// Pass the account id string explicitly. A nil interface / *string default can
	// stringify as a pointer address in ResolveAccountID (SDK quirk).
	ordersSub, err := client.Orders.Subscribe(ctx, accountID)
	if err != nil {
		log.Fatal(err)
	}
	defer ordersSub.Close()

	balancesSub, err := client.Balances.Subscribe(ctx, accountID)
	if err != nil {
		log.Fatal(err)
	}
	defer balancesSub.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		seen := 0
		timeout := time.After(30 * time.Second)
		for seen < settings.StreamCount {
			select {
			case order, ok := <-ordersSub.Messages():
				if !ok {
					fmt.Println("Orders stream closed")
					return
				}
				fmt.Printf(
					"[orders] client_order_id=%s status=%s order_id=%s side=%s\n",
					order.ClientOrderID, order.Status, order.OrderID, order.Side,
				)
				seen++
			case <-timeout:
				fmt.Printf("Orders stream: %d event(s) within 30s\n", seen)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		seen := 0
		timeout := time.After(30 * time.Second)
		for seen < settings.StreamCount {
			select {
			case bal, ok := <-balancesSub.Messages():
				if !ok {
					fmt.Println("Balances stream closed")
					return
				}
				fmt.Printf(
					"[balances] asset_id=%d available=%s trading=%s funding=%s\n",
					bal.AssetID, bal.Available, bal.Trading, bal.Funding,
				)
				seen++
			case <-timeout:
				fmt.Printf("Balances stream: %d event(s) within 30s\n", seen)
				return
			}
		}
	}()

	wg.Wait()
	fmt.Println("Private realtime example finished")
}

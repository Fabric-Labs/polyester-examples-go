package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	sdkerrors "github.com/Fabric-Labs/polyester-sdk-go/errors"
	"github.com/Fabric-Labs/polyester-sdk-go/models"
)

const lifecycleTxHashEnv = "POLYESTER_EXAMPLES_LIFECYCLE_TX_HASH"

func main() {
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
	scope := "all"
	flows, err := client.Lifecycle.ListFlows(ctx, 20, nil, &scope, nil, nil, false)
	if err != nil {
		var route *sdkerrors.RouteNotFoundError
		var api *sdkerrors.APIError
		if errors.As(err, &route) || (errors.As(err, &api) && (api.Code == "unimplemented" || api.Code == "not_found")) {
			fmt.Printf("Lifecycle list unavailable on this host: %v\n", err)
			return
		}
		log.Fatal(err)
	}

	fmt.Printf("Lifecycle flows (%d)\n", len(flows.Flows))
	for _, flow := range flows.Flows {
		fmt.Printf(
			"  intent_id=%s kind=%s step=%s open=%v terminal=%v reason=%s\n",
			flow.IntentID, flow.FlowKind, flow.LatestStep, flow.IsOpen, flow.IsTerminal, flow.LifecycleReason,
		)
		if flow.ZipperReason != nil {
			fmt.Printf(
				"    zipper_reason reason_id=%q message=%q code=%d\n",
				flow.ZipperReason.ReasonID, flow.ZipperReason.Message, flow.ZipperReason.Code,
			)
		}
	}

	txHash := strings.TrimSpace(os.Getenv(lifecycleTxHashEnv))
	if txHash == "" {
		fmt.Printf("Set %s to demonstrate paginated transaction lookup.\n", lifecycleTxHashEnv)
		return
	}

	page, err := client.Lifecycle.ListFlowsByTx(ctx, txHash, "any", 50, nil)
	if err != nil {
		log.Fatal(err)
	}
	matches := append([]string(nil), flowIDs(page.Flows)...)
	for page.NextPageToken != "" {
		token := page.NextPageToken
		page, err = client.Lifecycle.ListFlowsByTx(ctx, txHash, "any", 50, &token)
		if err != nil {
			log.Fatal(err)
		}
		matches = append(matches, flowIDs(page.Flows)...)
	}

	fmt.Printf("Transaction matches (%d) for %s\n", len(matches), txHash)
	for _, flowID := range matches {
		fmt.Printf("  intent_id=%s\n", flowID)
	}
}

func flowIDs(flows []models.LifecycleFlowSummary) []string {
	ids := make([]string, 0, len(flows))
	for _, flow := range flows {
		ids = append(ids, flow.IntentID)
	}
	return ids
}

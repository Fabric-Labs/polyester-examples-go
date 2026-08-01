package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	sdkerrors "github.com/Fabric-Labs/polyester-sdk-go/errors"
)

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
}

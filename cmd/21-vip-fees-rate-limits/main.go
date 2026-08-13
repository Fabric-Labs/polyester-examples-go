package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Fabric-Labs/polyester-examples-go/polyesterexamples"
	sdkerrors "github.com/Fabric-Labs/polyester-sdk-go/errors"
)

func unavailable(err error) bool {
	var route *sdkerrors.RouteNotFoundError
	var api *sdkerrors.APIError
	if errors.As(err, &route) {
		return true
	}
	return errors.As(err, &api) && (api.Code == "unimplemented" || api.Code == "not_found")
}

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

	tiers, err := client.VIP.ListVIPTiers(ctx)
	if err != nil {
		if !unavailable(err) {
			log.Fatal(err)
		}
		fmt.Printf("VIP catalog unavailable on this host: %v\n", err)
	} else {
		fmt.Printf(
			"VIP catalog policy_version=%d tiers=%d retention_bp=%d\n",
			tiers.PolicyVersion, len(tiers.Tiers), tiers.RetentionThresholdBp,
		)
		for _, row := range tiers.Tiers {
			aop := "omitted"
			if row.AopThresholdUsd != nil {
				aop = *row.AopThresholdUsd
			}
			fmt.Printf(
				"  VIP%d volume_usd=%s aop_usd=%s maker=%s%% taker=%s%%\n",
				row.Tier, row.VolumeThresholdUsd, aop, row.MakerFeeRatePercent, row.TakerFeeRatePercent,
			)
		}
	}

	status, err := client.VIP.GetVIPStatus(ctx)
	if err != nil {
		if !unavailable(err) {
			log.Fatal(err)
		}
		fmt.Printf("VIP status unavailable on this host: %v\n", err)
	} else {
		vol := "omitted"
		if status.SettledVolume30DUsd != nil {
			vol = *status.SettledVolume30DUsd
		}
		aop := "omitted"
		if status.AverageAop30DUsd != nil {
			aop = *status.AverageAop30DUsd
		}
		fmt.Printf(
			"VIP status tier=%d volume_tier=%d aop_tier=%d volume_30d=%s aop_30d=%s\n",
			status.Tier, status.VolumeTier, status.AopTier, vol, aop,
		)
		if status.NextTierThresholds != nil {
			nxt := status.NextTierThresholds
			fmt.Printf(
				"  next VIP%d volume_usd=%s aop_usd=%s\n",
				nxt.Tier, nxt.VolumeThresholdUsd, nxt.AopThresholdUsd,
			)
		}
	}

	fees, err := client.Fees.GetSpotFeeRates(ctx, nil, nil, nil)
	if err != nil {
		if !unavailable(err) {
			log.Fatal(err)
		}
		fmt.Printf("spot fee rates unavailable on this host: %v\n", err)
	} else {
		fmt.Printf("Spot fee rates (%d)\n", len(fees.FeeRates))
		limit := min(8, len(fees.FeeRates))
		for _, row := range fees.FeeRates[:limit] {
			fmt.Printf(
				"  %s maker=%s%% taker=%s%% vip=%d\n",
				row.Symbol, row.MakerFeeRatePercent, row.TakerFeeRatePercent, row.VIPTier,
			)
		}
		if extra := len(fees.FeeRates) - limit; extra > 0 {
			fmt.Printf("  ... %d more\n", extra)
		}
	}

	catalog, err := client.RateLimits.GetRateLimitConfig(ctx)
	if err != nil {
		if !unavailable(err) {
			log.Fatal(err)
		}
		fmt.Printf("rate-limit catalog unavailable on this host: %v\n", err)
	} else {
		fmt.Printf(
			"Rate-limit catalog policy_version=%d rules=%d\n",
			catalog.PolicyVersion, len(catalog.Rules),
		)
		limit := min(6, len(catalog.Rules))
		for _, rule := range catalog.Rules[:limit] {
			fmt.Printf(
				"  %s VIP%d quota=%d/%dms burst=%d\n",
				rule.PolicyClass, rule.Tier, rule.QuotaWeight, rule.PeriodMs, rule.BurstWeight,
			)
		}
	}

	limits, err := client.RateLimits.GetTradingRateLimits(ctx, nil, nil)
	if err != nil {
		if !unavailable(err) {
			log.Fatal(err)
		}
		fmt.Printf("trading rate limits unavailable on this host: %v\n", err)
		return
	}
	fmt.Printf(
		"Trading rate limits policy_version=%d rules=%d api_key_rules=%d\n",
		limits.PolicyVersion, len(limits.Rules), len(limits.APIKeyRules),
	)
}

package metrics

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler/pkg/config"
)

// Evaluator resolves endpoints, scrapes Prometheus, and aggregates contract metrics.
type Evaluator struct {
	Resolver *EndpointResolver
	Scraper  *Scraper
	Logger   *slog.Logger
}

// Result is the aggregated metric value for a ScaledObject poll.
type Result struct {
	Value       float64
	PodCount    int
	ScrapeErrors int
}

// Evaluate runs the metric contract for the given trigger metadata.
func (e *Evaluator) Evaluate(ctx context.Context, meta *config.TriggerMetadata) (*Result, error) {
	if e.Resolver == nil || e.Scraper == nil {
		return nil, fmt.Errorf("evaluator not configured")
	}
	urls, err := e.Resolver.ScrapeURLs(ctx, meta.BackendNamespace, meta.BackendService, meta.PrometheusPort, meta.PrometheusPath)
	if err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return &Result{Value: 0, PodCount: 0}, nil
	}

	samples, scrapeErrs := e.Scraper.ScrapeMany(ctx, urls)
	if e.Logger != nil {
		for _, se := range scrapeErrs {
			e.Logger.Warn("scrape failed", "err", se)
		}
	}
	if len(samples) == 0 {
		return &Result{Value: 0, PodCount: 0, ScrapeErrors: len(scrapeErrs)}, nil
	}

	values := make([]float64, 0, len(samples))
	for _, s := range samples {
		v, err := EvaluateContract(meta.Metric, s)
		if err != nil {
			return nil, err
		}
		values = append(values, SanitizeFinite(v))
	}
	agg, err := Aggregate(values, meta.Aggregation)
	if err != nil {
		return nil, err
	}
	return &Result{
		Value:        SanitizeFinite(agg),
		PodCount:     len(samples),
		ScrapeErrors: len(scrapeErrs),
	}, nil
}

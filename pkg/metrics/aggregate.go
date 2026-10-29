package metrics

import (
	"fmt"
	"math"

	"github.com/hubtel/keda-otel-collector-fleet-scaler/pkg/config"
)

// Aggregate combines per-pod contract values.
func Aggregate(values []float64, aggregation string) (float64, error) {
	if len(values) == 0 {
		return 0, nil
	}
	switch aggregation {
	case config.AggregationAvg:
		var sum float64
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values)), nil
	case config.AggregationMax:
		max := values[0]
		for _, v := range values[1:] {
			if v > max {
				max = v
			}
		}
		return max, nil
	case config.AggregationSum:
		var sum float64
		for _, v := range values {
			sum += v
		}
		return sum, nil
	default:
		return 0, fmt.Errorf("unknown aggregation %q", aggregation)
	}
}

// EvaluateContract computes a single pod's contract value from scraped samples.
func EvaluateContract(metric string, samples map[string]float64) (float64, error) {
	switch metric {
	case config.MetricMemoryRSSBytes:
		return samples["otelcol_process_memory_rss"], nil
	case config.MetricExporterQueueSize:
		return samples["otelcol_exporter_queue_size"], nil
	case config.MetricExporterQueueUtilization:
		size := samples["otelcol_exporter_queue_size"]
		cap := samples["otelcol_exporter_queue_capacity"]
		if cap <= 0 {
			return 0, nil
		}
		return size / cap, nil
	case config.MetricExporterEnqueueFailed:
		return samples["otelcol_exporter_enqueue_failed_spans"], nil
	case config.MetricReceiverRefusedSpans:
		return samples["otelcol_receiver_refused_spans"], nil
	default:
		return 0, fmt.Errorf("unsupported metric %q", metric)
	}
}

// SanitizeFinite replaces NaN/Inf with 0.
func SanitizeFinite(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

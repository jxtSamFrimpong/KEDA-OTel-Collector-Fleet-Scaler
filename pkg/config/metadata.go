package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Supported contract metric names (see docs/metric-contract.md).
const (
	MetricMemoryRSSBytes           = "memory_rss_bytes"
	MetricExporterQueueUtilization = "exporter_queue_utilization"
	MetricExporterQueueSize        = "exporter_queue_size"
	MetricExporterEnqueueFailed    = "exporter_enqueue_failed_spans"
	MetricReceiverRefusedSpans     = "receiver_refused_spans"
)

const (
	AggregationAvg = "avg"
	AggregationMax = "max"
	AggregationSum = "sum"
)

// TriggerMetadata is parsed from ScaledObject trigger metadata.
type TriggerMetadata struct {
	Metric              string
	TargetValue         float64
	Aggregation         string
	PrometheusPort      string
	PrometheusPath      string
	BackendService      string
	BackendNamespace    string
	MetricName          string
	ActivationThreshold float64
	ScrapeTimeout       time.Duration
}

// ParseTriggerMetadata validates and normalizes scaler metadata.
func ParseTriggerMetadata(meta map[string]string, scaledObjectNamespace string) (*TriggerMetadata, error) {
	if meta == nil {
		return nil, fmt.Errorf("scaler metadata is required")
	}

	metric := strings.TrimSpace(meta["metric"])
	if metric == "" {
		return nil, fmt.Errorf("metadata.metric is required")
	}
	if !ValidMetric(metric) {
		return nil, fmt.Errorf("unsupported metadata.metric %q", metric)
	}

	targetRaw := strings.TrimSpace(meta["targetValue"])
	if targetRaw == "" {
		return nil, fmt.Errorf("metadata.targetValue is required")
	}
	target, err := strconv.ParseFloat(targetRaw, 64)
	if err != nil {
		return nil, fmt.Errorf("metadata.targetValue: %w", err)
	}

	backendService := strings.TrimSpace(meta["backendService"])
	if backendService == "" {
		return nil, fmt.Errorf("metadata.backendService is required")
	}

	agg := strings.ToLower(strings.TrimSpace(meta["aggregation"]))
	if agg == "" {
		agg = AggregationAvg
	}
	switch agg {
	case AggregationAvg, AggregationMax, AggregationSum:
	default:
		return nil, fmt.Errorf("metadata.aggregation must be avg, max, or sum")
	}

	ns := strings.TrimSpace(meta["backendNamespace"])
	if ns == "" {
		ns = scaledObjectNamespace
	}

	port := strings.TrimSpace(meta["prometheusPort"])
	if port == "" {
		port = "8888"
	}

	path := strings.TrimSpace(meta["prometheusPath"])
	if path == "" {
		path = "/metrics"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	metricName := strings.TrimSpace(meta["metricName"])
	if metricName == "" {
		metricName = "otelcol_backend_" + metric
	}

	activation := 0.0
	if v := strings.TrimSpace(meta["activationThreshold"]); v != "" {
		activation, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("metadata.activationThreshold: %w", err)
		}
	}

	timeout := 2 * time.Second
	if v := strings.TrimSpace(meta["scrapeTimeout"]); v != "" {
		timeout, err = time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("metadata.scrapeTimeout: %w", err)
		}
	}

	return &TriggerMetadata{
		Metric:              metric,
		TargetValue:         target,
		Aggregation:         agg,
		PrometheusPort:      port,
		PrometheusPath:      path,
		BackendService:      backendService,
		BackendNamespace:    ns,
		MetricName:          metricName,
		ActivationThreshold: activation,
		ScrapeTimeout:       timeout,
	}, nil
}

// ValidMetric reports whether name is a supported contract metric.
func ValidMetric(name string) bool {
	switch name {
	case MetricMemoryRSSBytes,
		MetricExporterQueueUtilization,
		MetricExporterQueueSize,
		MetricExporterEnqueueFailed,
		MetricReceiverRefusedSpans:
		return true
	default:
		return false
	}
}

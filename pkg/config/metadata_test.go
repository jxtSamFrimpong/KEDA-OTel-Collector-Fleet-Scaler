package config

import (
	"testing"
	"time"
)

func TestParseTriggerMetadataDefaults(t *testing.T) {
	meta, err := ParseTriggerMetadata(map[string]string{
		"metric":         MetricExporterQueueUtilization,
		"targetValue":    "0.65",
		"backendService": "otel-backends",
	}, "observability")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Aggregation != AggregationAvg {
		t.Fatalf("aggregation=%s", meta.Aggregation)
	}
	if meta.BackendNamespace != "observability" {
		t.Fatalf("namespace=%s", meta.BackendNamespace)
	}
	if meta.PrometheusPort != "8888" || meta.PrometheusPath != "/metrics" {
		t.Fatalf("prometheus=%s%s", meta.PrometheusPort, meta.PrometheusPath)
	}
	if meta.MetricName != "otelcol_backend_exporter_queue_utilization" {
		t.Fatalf("metricName=%s", meta.MetricName)
	}
	if meta.ScrapeTimeout != 2*time.Second {
		t.Fatalf("timeout=%s", meta.ScrapeTimeout)
	}
}

func TestParseTriggerMetadataValidation(t *testing.T) {
	_, err := ParseTriggerMetadata(map[string]string{
		"metric":      "not_a_metric",
		"targetValue": "1",
	}, "ns")
	if err == nil {
		t.Fatal("expected error for bad metric")
	}

	_, err = ParseTriggerMetadata(map[string]string{
		"metric":         MetricMemoryRSSBytes,
		"targetValue":    "abc",
		"backendService": "svc",
	}, "ns")
	if err == nil {
		t.Fatal("expected error for bad target")
	}

	_, err = ParseTriggerMetadata(map[string]string{
		"metric":         MetricMemoryRSSBytes,
		"targetValue":    "1",
		"backendService": "svc",
		"aggregation":    "p99",
	}, "ns")
	if err == nil {
		t.Fatal("expected error for bad aggregation")
	}
}

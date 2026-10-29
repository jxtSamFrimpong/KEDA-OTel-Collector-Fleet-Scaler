package metrics

import (
	"strings"
	"testing"
)

func TestParsePrometheus(t *testing.T) {
	body := `
# HELP otelcol_exporter_queue_size Current size of the sending queue
# TYPE otelcol_exporter_queue_size gauge
otelcol_exporter_queue_size{exporter="otlp"} 40
otelcol_exporter_queue_size{exporter="datadog"} 20
otelcol_exporter_queue_capacity{exporter="otlp"} 100
otelcol_exporter_queue_capacity{exporter="datadog"} 100
otelcol_process_memory_rss 123456789
otelcol_ignored_metric 1
`
	samples, err := ParsePrometheus(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if samples["otelcol_exporter_queue_size"] != 60 {
		t.Fatalf("queue_size=%v", samples["otelcol_exporter_queue_size"])
	}
	if samples["otelcol_exporter_queue_capacity"] != 200 {
		t.Fatalf("queue_capacity=%v", samples["otelcol_exporter_queue_capacity"])
	}
	if samples["otelcol_process_memory_rss"] != 123456789 {
		t.Fatalf("rss=%v", samples["otelcol_process_memory_rss"])
	}
	if _, ok := samples["otelcol_ignored_metric"]; ok {
		t.Fatal("ignored metric should not be present")
	}
}

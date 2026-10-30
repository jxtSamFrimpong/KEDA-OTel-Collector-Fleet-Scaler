package metrics

import (
	"testing"

	"github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler/pkg/config"
)

func TestAggregate(t *testing.T) {
	avg, err := Aggregate([]float64{0.2, 0.4, 0.6}, config.AggregationAvg)
	if err != nil || avg < 0.399 || avg > 0.401 {
		t.Fatalf("avg=%v err=%v", avg, err)
	}
	max, err := Aggregate([]float64{0.2, 0.9, 0.1}, config.AggregationMax)
	if err != nil || max != 0.9 {
		t.Fatalf("max=%v err=%v", max, err)
	}
	sum, err := Aggregate([]float64{1, 2, 3}, config.AggregationSum)
	if err != nil || sum != 6 {
		t.Fatalf("sum=%v err=%v", sum, err)
	}
	empty, err := Aggregate(nil, config.AggregationAvg)
	if err != nil || empty != 0 {
		t.Fatalf("empty=%v err=%v", empty, err)
	}
}

func TestEvaluateContractUtilization(t *testing.T) {
	v, err := EvaluateContract(config.MetricExporterQueueUtilization, map[string]float64{
		"otelcol_exporter_queue_size":     65,
		"otelcol_exporter_queue_capacity": 100,
	})
	if err != nil || v != 0.65 {
		t.Fatalf("got=%v err=%v", v, err)
	}
	v, err = EvaluateContract(config.MetricExporterQueueUtilization, map[string]float64{
		"otelcol_exporter_queue_size":     10,
		"otelcol_exporter_queue_capacity": 0,
	})
	if err != nil || v != 0 {
		t.Fatalf("cap0 got=%v err=%v", v, err)
	}
}

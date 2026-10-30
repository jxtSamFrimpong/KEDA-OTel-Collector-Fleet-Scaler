package scaler

import (
	"context"
	"testing"

	pb "github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler/api/externalscaler"
	"github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler/pkg/config"
	"github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler/pkg/metrics"
)

func TestGetMetricSpec(t *testing.T) {
	srv := &Server{Evaluator: &metrics.Evaluator{}}
	resp, err := srv.GetMetricSpec(context.Background(), &pb.ScaledObjectRef{
		Name:      "otel-backends",
		Namespace: "observability",
		ScalerMetadata: map[string]string{
			"metric":         config.MetricExporterQueueUtilization,
			"targetValue":    "0.65",
			"backendService": "otel-backends",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.MetricSpecs) != 1 {
		t.Fatalf("specs=%d", len(resp.MetricSpecs))
	}
	if resp.MetricSpecs[0].TargetSizeFloat != 0.65 {
		t.Fatalf("target=%v", resp.MetricSpecs[0].TargetSizeFloat)
	}
	if resp.MetricSpecs[0].MetricName != "otelcol_backend_exporter_queue_utilization" {
		t.Fatalf("name=%s", resp.MetricSpecs[0].MetricName)
	}
}

func TestGetMetricSpecMissingMetadata(t *testing.T) {
	srv := &Server{}
	_, err := srv.GetMetricSpec(context.Background(), &pb.ScaledObjectRef{
		Name:           "x",
		Namespace:      "ns",
		ScalerMetadata: map[string]string{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

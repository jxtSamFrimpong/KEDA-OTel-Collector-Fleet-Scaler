package scaler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/hubtel/keda-otel-collector-fleet-scaler/api/externalscaler"
	"github.com/hubtel/keda-otel-collector-fleet-scaler/pkg/config"
	"github.com/hubtel/keda-otel-collector-fleet-scaler/pkg/metrics"
)

// Server implements the KEDA ExternalScaler gRPC service.
type Server struct {
	pb.UnimplementedExternalScalerServer
	Evaluator *metrics.Evaluator
	Logger    *slog.Logger
}

func (s *Server) log() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

// IsActive reports whether the metric is above the activation threshold.
func (s *Server) IsActive(ctx context.Context, so *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {
	meta, err := config.ParseTriggerMetadata(so.GetScalerMetadata(), so.GetNamespace())
	if err != nil {
		return nil, err
	}
	res, err := s.Evaluator.Evaluate(ctx, meta)
	if err != nil {
		return nil, err
	}
	active := res.Value > meta.ActivationThreshold
	s.log().Debug("IsActive",
		"scaledObject", so.GetName(),
		"namespace", so.GetNamespace(),
		"value", res.Value,
		"active", active,
		"pods", res.PodCount,
	)
	return &pb.IsActiveResponse{Result: active}, nil
}

// StreamIsActive periodically pushes IsActive for external-push triggers.
func (s *Server) StreamIsActive(so *pb.ScaledObjectRef, stream pb.ExternalScaler_StreamIsActiveServer) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		resp, err := s.IsActive(stream.Context(), so)
		if err != nil {
			return err
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
		}
	}
}

// GetMetricSpec returns the HPA metric name and target.
func (s *Server) GetMetricSpec(_ context.Context, so *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {
	meta, err := config.ParseTriggerMetadata(so.GetScalerMetadata(), so.GetNamespace())
	if err != nil {
		return nil, err
	}
	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{
			{
				MetricName:      meta.MetricName,
				TargetSize:      int64(meta.TargetValue),
				TargetSizeFloat: meta.TargetValue,
			},
		},
	}, nil
}

// GetMetrics returns the current aggregated metric value.
func (s *Server) GetMetrics(ctx context.Context, req *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	so := req.GetScaledObjectRef()
	if so == nil {
		return nil, fmt.Errorf("scaledObjectRef is required")
	}
	meta, err := config.ParseTriggerMetadata(so.GetScalerMetadata(), so.GetNamespace())
	if err != nil {
		return nil, err
	}
	metricName := req.GetMetricName()
	if metricName == "" {
		metricName = meta.MetricName
	}
	res, err := s.Evaluator.Evaluate(ctx, meta)
	if err != nil {
		return nil, err
	}
	s.log().Debug("GetMetrics",
		"scaledObject", so.GetName(),
		"metric", metricName,
		"value", res.Value,
		"pods", res.PodCount,
		"scrapeErrors", res.ScrapeErrors,
	)
	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{
			{
				MetricName:       metricName,
				MetricValue:      int64(res.Value),
				MetricValueFloat: res.Value,
			},
		},
	}, nil
}

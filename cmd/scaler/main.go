package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	pb "github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler/api/externalscaler"
	"github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler/pkg/metrics"
	"github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler/pkg/scaler"
)

func main() {
	var (
		grpcAddr   = flag.String("grpc-addr", ":9090", "gRPC listen address for KEDA ExternalScaler")
		httpAddr   = flag.String("http-addr", ":8080", "HTTP listen address for health endpoints")
		kubeconfig = flag.String("kubeconfig", "", "optional kubeconfig path (out-of-cluster)")
		logLevel   = flag.String("log-level", "info", "log level: debug, info, warn, error")
	)
	flag.Parse()

	logger := newLogger(*logLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	kubeClient, err := newKubeClient(*kubeconfig)
	if err != nil {
		logger.Error("kubernetes client", "err", err)
		os.Exit(1)
	}

	eval := &metrics.Evaluator{
		Resolver: &metrics.EndpointResolver{Client: kubeClient},
		Scraper:  metrics.NewScraper(2 * time.Second),
		Logger:   logger,
	}
	srv := &scaler.Server{Evaluator: eval, Logger: logger}

	grpcServer := grpc.NewServer()
	pb.RegisterExternalScalerServer(grpcServer, srv)
	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		logger.Error("listen grpc", "addr", *grpcAddr, "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	httpServer := &http.Server{Addr: *httpAddr, Handler: mux}

	go func() {
		logger.Info("http listening", "addr", *httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server", "err", err)
			stop()
		}
	}()

	go func() {
		logger.Info("grpc listening", "addr", *grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	grpcServer.GracefulStop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func newLogger(level string) *slog.Logger {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
}

func newKubeClient(kubeconfig string) (*kubernetes.Clientset, error) {
	var cfg *rest.Config
	var err error
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig for local development.
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				loadingRules, &clientcmd.ConfigOverrides{},
			).ClientConfig()
		}
	}
	if err != nil {
		return nil, fmt.Errorf("kube config: %w", err)
	}
	return kubernetes.NewForConfig(cfg)
}

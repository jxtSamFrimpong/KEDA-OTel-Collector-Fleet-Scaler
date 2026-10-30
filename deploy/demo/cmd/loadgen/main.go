package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	mode := flag.String("mode", "affinity", "affinity|load")
	endpoint := flag.String("endpoint", envOr("OTLP_ENDPOINT", "otel-gateway.observability.svc:4317"), "OTLP gRPC endpoint")
	spansPerTrace := flag.Int("spans", 5, "spans per trace")
	gapMs := flag.Int("gap-ms", 400, "gap between spans in a trace (affinity)")
	duration := flag.Duration("duration", 90*time.Second, "load duration")
	rate := flag.Float64("rate", 100, "traces per second (load)")
	flag.Parse()

	ctx := context.Background()
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(*endpoint),
		otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		log.Fatalf("exporter: %v", err)
	}
	defer func() { _ = exp.Shutdown(ctx) }()

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName("orbstack-otlp-demo")),
	)
	if err != nil {
		log.Fatalf("resource: %v", err)
	}

	switch *mode {
	case "affinity":
		runAffinity(ctx, exp, res, *spansPerTrace, time.Duration(*gapMs)*time.Millisecond)
	case "load":
		runLoad(ctx, exp, res, *duration, *rate, *spansPerTrace)
	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}

func runAffinity(ctx context.Context, exp sdktrace.SpanExporter, res *resource.Resource, spansPerTrace int, gap time.Duration) {
	traceIDs := []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"cccccccccccccccccccccccccccccccc",
		"dddddddddddddddddddddddddddddddd",
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	}
	log.Printf("affinity endpoint spans=%d gap=%s traces=%d", spansPerTrace, gap, len(traceIDs))

	// Use a synchronous exporter path via manual span stubs exported directly
	for _, tidHex := range traceIDs {
		tid, err := trace.TraceIDFromHex(tidHex)
		if err != nil {
			log.Fatalf("trace id: %v", err)
		}
		log.Printf("sending trace_id=%s", tidHex)
		for i := 0; i < spansPerTrace; i++ {
			sid := mustSpanID(i + 1)
			start := time.Now().Add(-10 * time.Millisecond)
			end := time.Now()
			stub := tracetest.SpanStub{
				Name:      fmt.Sprintf("affinity-span-%d", i),
				SpanKind:  trace.SpanKindInternal,
				StartTime: start,
				EndTime:   end,
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    tid,
					SpanID:     sid,
					TraceFlags: trace.FlagsSampled,
				}),
				Resource: res,
				Attributes: []attribute.KeyValue{
					attribute.String("demo.trace_id", tidHex),
					attribute.Int("demo.span_index", i),
				},
			}
			if err := exp.ExportSpans(ctx, tracetest.SpanStubs{stub}.Snapshots()); err != nil {
				log.Fatalf("export: %v", err)
			}
			time.Sleep(gap)
		}
	}
	log.Printf("affinity send complete")
}

func runLoad(ctx context.Context, exp sdktrace.SpanExporter, res *resource.Resource, duration time.Duration, rate float64, spansPerTrace int) {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(500*time.Millisecond)),
		sdktrace.WithResource(res),
	)
	defer func() { _ = tp.Shutdown(ctx) }()
	tr := tp.Tracer("load-demo")

	interval := time.Duration(float64(time.Second) / rate)
	end := time.Now().Add(duration)
	sent := 0
	log.Printf("load rate=%.1f/s duration=%s spans=%d", rate, duration, spansPerTrace)
	for time.Now().Before(end) {
		_, span := tr.Start(ctx, "load-root", trace.WithAttributes(attribute.Bool("demo.load", true)))
		for i := 0; i < spansPerTrace-1; i++ {
			_, c := tr.Start(ctx, fmt.Sprintf("load-child-%d", i))
			c.End()
		}
		span.End()
		sent++
		if sent%200 == 0 {
			log.Printf("sent_traces=%d", sent)
		}
		time.Sleep(interval)
	}
	log.Printf("load complete sent_traces=%d", sent)
}

func mustSpanID(n int) trace.SpanID {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(n)+0x1000)
	// mix in randomness so repeated runs differ at span level
	var r [2]byte
	_, _ = rand.Read(r[:])
	b[6], b[7] = r[0], r[1]
	sid, err := trace.SpanIDFromHex(hex.EncodeToString(b[:]))
	if err != nil {
		log.Fatalf("span id: %v", err)
	}
	return sid
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

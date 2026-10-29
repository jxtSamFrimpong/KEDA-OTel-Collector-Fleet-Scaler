# KEDA OTel Collector Fleet Scaler

KEDA external scaler + reference fleet for autoscaling **OpenTelemetry Collector backends** behind a TraceID-aware gateway.

This project does **not** reinvent TraceID routing. It reuses the upstream OpenTelemetry [`load_balancing`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/loadbalancingexporter) exporter (k8s/DNS discovery) and focuses on:

1. Scaling backends from **collector self-telemetry** (Prometheus scrape)
2. Safe join/leave of the consistent-hash ring (`forward` / `stateful` fleet profiles)
3. Packaging that fits the **KEDA external scaler** ecosystem

```text
Apps → Gateway Collectors (HA, load_balancing/TraceID)
         → Headless Service / EndpointSlices
           → Backend Collectors (KEDA-scaled) → Datadog / other
                 ↑ Prometheus :8888
           keda-otel-collector-fleet-scaler (gRPC :9090) ← KEDA
```

## Why this exists

| Pain | Approach here |
|---|---|
| Single collector SPOF + memory pressure | HA gateway + autoscaled backends |
| TraceID routing + membership | Upstream `load_balancing` + EndpointSlice watch |
| Autoscaling collectors | KEDA `external` scaler on queue/memory metrics |
| Stateful vs forward configs | Declared **fleet profiles** (not config.yaml parsing) |

Not the same as [Kedify otel-add-on](https://github.com/kedify/otel-add-on) (that scales **apps** from OTel metrics).

## Quick start

### 1. Build / install the scaler

```bash
make test
make build
# or
docker build -t ghcr.io/hubtel/keda-otel-collector-fleet-scaler:dev .
helm install keda-otel-collector-fleet-scaler charts/keda-otel-collector-fleet-scaler -n observability --create-namespace
```

### 2. Deploy a reference fleet

Requires KEDA in the cluster and (for Datadog export) a Secret `datadog-secret`.

```bash
kubectl apply -f deploy/scaler/
kubectl apply -k deploy/examples/forward   # or deploy/examples/stateful
```

### 3. Point apps at the gateway

OTLP endpoint: `otel-gateway.observability.svc:4317` (gRPC).

## ScaledObject metadata

```yaml
triggers:
  - type: external
    metadata:
      scalerAddress: keda-otel-collector-fleet-scaler.observability.svc:9090
      metric: exporter_queue_utilization   # or memory_rss_bytes
      targetValue: "0.65"
      aggregation: avg
      backendService: otel-backends
      backendNamespace: observability
```

See [docs/metric-contract.md](docs/metric-contract.md).

## Fleet profiles

| Profile | Workload | Drain | Use when |
|---|---|---|---|
| `forward` | Deployment | 15s | Stateless forward to Datadog |
| `stateful` | StatefulSet | 35s+ | Tail sampling / in-memory hold |

Platform teams own collector `config.yaml`; they declare the profile to match. See [docs/fleet-profiles.md](docs/fleet-profiles.md) and [docs/scale-safety.md](docs/scale-safety.md).

## Documentation

- [Architecture](docs/architecture.md)
- [Metric contract](docs/metric-contract.md)
- [Fleet profiles](docs/fleet-profiles.md)
- [Scale safety](docs/scale-safety.md)
- [Community packaging](docs/community-packaging.md)

## Development

```bash
make test
make build
# Regenerate protobuf (requires protoc + plugins):
make generate
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).

# Metric Contract

This document defines the **collector self-telemetry** the KEDA OTel Collector Fleet Scaler
consumes to decide backend replica count.

## Ingestion path (v1)

**Prometheus scrape** of each backend collector’s internal metrics endpoint.

| Setting | Default |
|---|---|
| Path | `/metrics` |
| Port | `8888` |
| Discovery | Kubernetes endpoints of the backend Service (same headless Service used by `load_balancing`) |

Collectors must expose Prometheus pull telemetry:

```yaml
service:
  telemetry:
    metrics:
      readers:
        - pull:
            exporter:
              prometheus:
                host: "0.0.0.0"
                port: 8888
```

Rationale: OpenTelemetry Collector already exposes these metrics by default; scrape
avoids a second OTLP pipeline and matches how most platform teams already monitor
collectors. OTLP push ingestion is a non-goal for v1.

## Metric families

The scaler aggregates across **Ready** backend pod scrapes, then returns one value
to KEDA per configured signal.

### Primary signals

| Contract name | Source metric(s) | Aggregation | Unit | Scale-up when |
|---|---|---|---|---|
| `memory_rss_bytes` | `otelcol_process_memory_rss` | `avg` (default) or `max` | bytes | avg/max RSS above `targetValue` |
| `exporter_queue_utilization` | `otelcol_exporter_queue_size` / `otelcol_exporter_queue_capacity` | `avg` or `max` of per-pod ratio | ratio 0–1 | utilization above `targetValue` (recommend `0.6`–`0.7`) |
| `exporter_queue_size` | `otelcol_exporter_queue_size` | `avg` or `max` | batches | absolute queue depth above target |

### Secondary / activation signals

| Contract name | Source metric(s) | Notes |
|---|---|---|
| `exporter_enqueue_failed_spans` | `otelcol_exporter_enqueue_failed_spans` | Counter; scaler may expose rate over scrape window. Prefer queue utilization for steady-state scaling. |
| `receiver_refused_spans` | `otelcol_receiver_refused_spans` | Indicates memory_limiter or backpressure; useful as activation / alert, not sole HPA target. |

## ScaledObject metadata

```yaml
triggers:
  - type: external
    metadata:
      scalerAddress: keda-otel-collector-fleet-scaler.observability.svc:9090
      metric: exporter_queue_utilization   # contract name
      targetValue: "0.65"
      aggregation: avg                     # avg | max | sum
      prometheusPort: "8888"
      prometheusPath: /metrics
      backendService: otel-backends
      backendNamespace: observability      # optional; defaults to ScaledObject namespace
      metricName: otelcol_backend_queue_utilization  # HPA metric name exposed to KEDA
      activationThreshold: "0.05"          # IsActive when aggregated value > threshold
```

### Metadata reference

| Key | Required | Description |
|---|---|---|
| `scalerAddress` | yes (KEDA) | gRPC address of this external scaler |
| `metric` | yes | Contract name from the tables above |
| `targetValue` | yes | HPA target (float). For utilization use `0.65`. For memory use bytes (e.g. `2147483648`). |
| `aggregation` | no | `avg` (default), `max`, or `sum` across scraped pods |
| `prometheusPort` | no | Default `8888` |
| `prometheusPath` | no | Default `/metrics` |
| `backendService` | yes | Service selecting backend collector pods |
| `backendNamespace` | no | Defaults to ScaledObject namespace |
| `metricName` | no | Name returned in `GetMetricSpec`; default derived from `metric` |
| `activationThreshold` | no | `IsActive` is true when value > threshold; default `0` |
| `scrapeTimeout` | no | Per-pod scrape timeout; default `2s` |
| `unsafeSsl` | no | Reserved; scrapes are plain HTTP in v1 |

## Aggregation semantics

1. Resolve Ready endpoints for `backendService` (EndpointSlice `ready != false`).
2. Scrape each endpoint’s Prometheus path in parallel.
3. For each pod, compute the contract value (e.g. queue size / capacity; if capacity is 0, treat utilization as 0).
4. Aggregate with `avg` / `max` / `sum`.
5. Return that float to KEDA as `metricValueFloat`.

Empty endpoint set → metric value `0`, `IsActive` false (unless `activationThreshold` is negative).

## Relationship to collector config

Platform teams own `config.yaml`. This contract only requires:

1. Prometheus metrics on the configured port.
2. Exporters that emit queue metrics when using `exporter_queue_*` signals (standard for OTLP exporters with `sending_queue`).

Memory scaling works regardless of processors. Queue scaling is most meaningful when backends export to Datadog (or another remote backend) with a sending queue enabled.

## Non-goals

- Scraping application APM metrics (use Kedify OTel add-on or Prometheus scaler for that).
- Inferring which processors are configured.
- Push-based OTLP metric ingestion in v1.

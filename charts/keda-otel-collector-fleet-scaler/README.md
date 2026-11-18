# KEDA OTel Collector Fleet Scaler

Helm chart for a **KEDA external scaler** that autoscales OpenTelemetry Collector **backends** behind a TraceID-aware gateway.

The chart ships the scaler by default. Optionally (`fleet.enabled=true`) it also deploys a reference fleet: HA gateway collectors using the upstream OpenTelemetry [`load_balancing`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/loadbalancingexporter) exporter, headless backends, gateway RBAC, and a ScaledObject.

```text
Apps → Gateway (load_balancing / TraceID)
         → Backend collectors (KEDA-scaled)
                 ↑ Prometheus :8888
           this scaler (gRPC :9090) ← KEDA
```

| | |
|---|---|
| Chart version | **0.2.2** |
| Scaler image (`appVersion`) | `ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:`**0.1.1** |
| OCI | `oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler` |
| Source / docs | [GitHub](https://github.com/jxtSamFrimpong/KEDA-OTel-Collector-Fleet-Scaler) |

This is **not** the Kedify OTel add-on (that scales applications from OTel metrics).

## Prerequisites

- Kubernetes cluster
- [Helm](https://helm.sh/) 3
- [KEDA](https://keda.sh/) installed (ScaledObject CRDs) when using the in-chart ScaledObject or your own external trigger
- Optional: [AWS Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/) for BYO NLB via `TargetGroupBinding`
- Optional: Secret `datadog-secret` when fleet backends use the stock Datadog exporter

## Install — scaler only

```bash
helm upgrade --install keda-otel-collector-fleet-scaler \
  oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler \
  --version 0.2.2 \
  -n observability --create-namespace
```

ScaledObject trigger address (default release name / namespace):

```text
keda-otel-collector-fleet-scaler.observability.svc:9090
```

## Install — with reference fleet

```bash
helm upgrade --install keda-otel-collector-fleet-scaler \
  oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler \
  --version 0.2.2 \
  -n observability --create-namespace \
  --set fleet.enabled=true
```

Demo backends without Datadog: `--set fleet.backend.exporter=debug`.

In-cluster OTLP: `otel-gateway.<namespace>.svc:4317` (gRPC) / `:4318` (HTTP). Do **not** expose scaler `:9090` externally.

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

## External OTLP (BYO vs dedicated LB)

Gateway Service defaults to **ClusterIP** (BYO-compatible).

| Mode | Values sketch |
|---|---|
| **BYO NLB** (preferred) | `fleet.gateway.service.type=ClusterIP` + `fleet.gateway.targetGroupBinding.enabled=true` + TG ARNs |
| **Dedicated LB** | `fleet.gateway.service.type=LoadBalancer` + cloud annotations |

Full BYO walkthrough: [docs/byo-load-balancer.md](https://github.com/jxtSamFrimpong/KEDA-OTel-Collector-Fleet-Scaler/blob/main/docs/byo-load-balancer.md).

## Configuration highlights

| Key | Default | Meaning |
|---|---|---|
| `fleet.enabled` | `false` | Deploy gateway + backends + ScaledObject |
| `fleet.profile` | `forward` | `forward` or `stateful` drain/cooldown profile |
| `fleet.backend.exporter` | `datadog` | Stock backend exporter; `debug` for local demos |
| `fleet.scaledObject.enabled` | `true` | Create ScaledObject when fleet is on (needs KEDA) |
| `fleet.gateway.targetGroupBinding.enabled` | `false` | AWS LBC TargetGroupBinding for existing NLB |
| `image.tag` | *(empty → appVersion)* | Scaler image tag |

Inspect all values:

```bash
helm show values oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler --version 0.2.2
```

## Documentation

- [Quickstart](https://github.com/jxtSamFrimpong/KEDA-OTel-Collector-Fleet-Scaler/blob/main/quickstart.md) — install and verify step-by-step
- [Architecture](https://github.com/jxtSamFrimpong/KEDA-OTel-Collector-Fleet-Scaler/blob/main/docs/architecture.md)
- [Metric contract](https://github.com/jxtSamFrimpong/KEDA-OTel-Collector-Fleet-Scaler/blob/main/docs/metric-contract.md)
- [Fleet profiles](https://github.com/jxtSamFrimpong/KEDA-OTel-Collector-Fleet-Scaler/blob/main/docs/fleet-profiles.md)
- [BYO load balancer](https://github.com/jxtSamFrimpong/KEDA-OTel-Collector-Fleet-Scaler/blob/main/docs/byo-load-balancer.md)
- [Project README](https://github.com/jxtSamFrimpong/KEDA-OTel-Collector-Fleet-Scaler/blob/main/README.md)

## License

Apache License 2.0.

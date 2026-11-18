# Quickstart

Install and use **KEDA OTel Collector Fleet Scaler** — a KEDA external scaler for OpenTelemetry collector fleets that reuse the upstream TraceID `load_balancing` gateway pattern.

```text
Apps → Gateway Collectors (HA, load_balancing/TraceID)
         → Headless Service / EndpointSlices
           → Backend Collectors (KEDA-scaled)
                 ↑ Prometheus :8888
           keda-otel-collector-fleet-scaler (gRPC :9090) ← KEDA
```

This project scales **collector backends** from collector self-telemetry. It is not the Kedify OTel add-on (which scales apps from OTel metrics).

## Prerequisites

| Requirement | Notes |
|---|---|
| Kubernetes cluster | Any recent cluster with workloads in a target namespace |
| [KEDA](https://keda.sh/) | Required for ScaledObject / external triggers (`fleet.scaledObject.enabled=true` by default when fleet is on) |
| Helm 3 | OCI install from GHCR |
| Optional: [AWS Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/) | Only for BYO NLB via `TargetGroupBinding` |
| Optional: Secret `datadog-secret` | When fleet backends use the stock Datadog exporter (`fleet.backend.exporter=datadog`) |

Chart version in the examples below: **0.2.2**. Scaler image / `appVersion`: **0.1.1**.

## Install scaler only

Default chart mode: scaler Deployment + Service on gRPC `:9090`. No gateway or backends.

```bash
helm upgrade --install keda-otel-collector-fleet-scaler \
  oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler \
  --version 0.2.2 \
  -n observability --create-namespace
```

Confirm the Service and pods:

```bash
kubectl -n observability get deploy,svc -l app.kubernetes.io/name=keda-otel-collector-fleet-scaler
kubectl -n observability get endpoints keda-otel-collector-fleet-scaler
```

gRPC address for ScaledObject triggers (default release/name):

```text
keda-otel-collector-fleet-scaler.observability.svc:9090
```

## Install with in-chart fleet

Turnkey gateway + backends + ScaledObject in the same release (`fleet.enabled=true`). Chart ≥ **0.2.0** for the fleet; ≥ **0.2.1** for in-chart BYO `TargetGroupBinding`.

```bash
helm upgrade --install keda-otel-collector-fleet-scaler \
  oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler \
  --version 0.2.2 \
  -n observability --create-namespace \
  --set fleet.enabled=true
```

Useful overrides:

```bash
# Local / demo backends (no Datadog Secret)
--set fleet.backend.exporter=debug

# Stateful profile (longer drain / cooldown)
--set fleet.profile=stateful
```

In-cluster OTLP endpoints:

```text
otel-gateway.observability.svc:4317   # gRPC
otel-gateway.observability.svc:4318   # HTTP
```

Never expose the scaler port `:9090` on an external load balancer — OTLP **4317** / **4318** only.

## Point a ScaledObject at the scaler

When the fleet is enabled, the chart creates a ScaledObject by default. For a custom fleet (scaler-only install), create your own:

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: otel-backends
  namespace: observability
spec:
  scaleTargetRef:
    name: otel-backends
  minReplicaCount: 2
  maxReplicaCount: 20
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

Verify KEDA can reach the scaler (pod logs / ScaledObject status):

```bash
kubectl -n observability get scaledobject
kubectl -n observability describe scaledobject otel-backends
kubectl -n observability logs -l app.kubernetes.io/name=keda-otel-collector-fleet-scaler --tail=50
```

Metric names and scrape contract: [docs/metric-contract.md](docs/metric-contract.md).

## Bring-your-own load balancer

Prefer keeping the gateway Service as **ClusterIP** and attaching pods to an existing NLB with AWS LBC `TargetGroupBinding`. Full walkthrough:

→ [docs/byo-load-balancer.md](docs/byo-load-balancer.md)

Minimal values sketch:

```yaml
fleet:
  enabled: true
  gateway:
    service:
      type: ClusterIP
    targetGroupBinding:
      enabled: true
      targetType: ip
      otlpGrpcTargetGroupARN: arn:aws:elasticloadbalancing:...
      otlpHttpTargetGroupARN: arn:aws:elasticloadbalancing:...
```

Dedicated LB (chart provisions a new NLB): set `fleet.gateway.service.type=LoadBalancer` plus cloud annotations — see the root [README](README.md) and BYO doc comparison table.

## Next reading

| Doc | Why |
|---|---|
| [docs/architecture.md](docs/architecture.md) | Gateway / backend / scaler layout |
| [docs/metric-contract.md](docs/metric-contract.md) | Prometheus scrape + ScaledObject metadata |
| [docs/fleet-profiles.md](docs/fleet-profiles.md) | `forward` vs `stateful` |
| [docs/scale-safety.md](docs/scale-safety.md) | Join/leave and ring behavior |
| [docs/byo-load-balancer.md](docs/byo-load-balancer.md) | External OTLP via existing NLB |
| [README.md](README.md) | Full project overview, Artifact Hub, release notes |

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

### 1. Pull / install the scaler

Published images (multi-arch `linux/amd64` + `linux/arm64`):

```text
ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:latest
ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:0.1.0
ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:sha-<short>
```

```bash
# Install from the OCI Helm chart (recommended)
helm install keda-otel-collector-fleet-scaler \
  oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler \
  --version 0.1.0 \
  -n observability --create-namespace

# Or pull the image and apply raw manifests
docker pull ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:latest
kubectl apply -f deploy/scaler/
```

GHCR packages created by CI start as private. After the first successful publish, open the package on GitHub → **Package settings** → set visibility to **Public** (one-time).

Local development:

```bash
make test
make build
docker build -t keda-otel-collector-fleet-scaler:dev .
helm install keda-otel-collector-fleet-scaler charts/keda-otel-collector-fleet-scaler -n observability --create-namespace
```

### 2. Deploy a reference fleet

Requires KEDA in the cluster and (for Datadog export) a Secret `datadog-secret`.

```bash
kubectl apply -f deploy/scaler/          # if not using Helm for the scaler
kubectl apply -k deploy/examples/forward   # or deploy/examples/stateful
```

### 3. Point apps at the gateway

In-cluster OTLP (default ClusterIP Service):

```text
otel-gateway.<namespace>.svc:4317   # gRPC
otel-gateway.<namespace>.svc:4318   # HTTP
```

Example: `otel-gateway.observability.svc:4317`.

### External clients

The gateway Service stays **ClusterIP** by default. For optional L4 LoadBalancer access (OTLP **4317** / **4318** only — never the scaler):

```bash
kubectl apply -k deploy/examples/external-lb
kubectl -n observability get svc otel-gateway   # EXTERNAL-IP / hostname
```

Point SDKs at the LB address:

```bash
# gRPC
export OTEL_EXPORTER_OTLP_ENDPOINT=http://<EXTERNAL-IP>:4317

# HTTP
export OTEL_EXPORTER_OTLP_ENDPOINT=http://<EXTERNAL-IP>:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
```

Uncomment cloud annotations / `loadBalancerSourceRanges` in the overlay (or base Service comments) for AWS NLB, GCP, Azure, or source IP allowlists. No Ingress in this path.

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

## Helm (OCI) and Artifact Hub

Artifact Hub indexes public chart repositories; it does not host chart tarballs. This project publishes the chart to GHCR as an OCI artifact:

```bash
helm install keda-otel-collector-fleet-scaler \
  oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler \
  --version 0.1.0 \
  -n observability --create-namespace

helm show values oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler --version 0.1.0
```

### List the chart on Artifact Hub (one-time)

1. Ensure the GHCR packages below are **Public**:
   - `ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler` (image)
   - `ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler` (Helm OCI chart)
2. Sign in at [artifacthub.io](https://artifacthub.io) → **Control Panel** → **Add repository**.
3. Kind: **Helm charts**. URL (OCI — one Artifact Hub repo per chart):

   ```text
   oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler
   ```

4. After Artifact Hub assigns a `repositoryID`, put it in [`artifacthub-repo.yml`](artifacthub-repo.yml) and let CI re-push the `artifacthub.io` OCI tag (or push manually with [oras](https://oras.land/) as described in the [Artifact Hub OCI docs](https://artifacthub.io/docs/topics/repositories/helm-charts/#oci-support)). That enables ownership claim / verified publisher.

Official docs: [Helm charts on Artifact Hub](https://artifacthub.io/docs/topics/repositories/helm-charts/) · [OCI support](https://artifacthub.io/docs/topics/repositories/helm-charts/#oci-support) · [Helm annotations](https://artifacthub.io/docs/topics/annotations/helm/).

## Documentation

- [Architecture](docs/architecture.md)
- [Scale metrics](docs/scale-metrics.md) (plain-language guide)
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

CI workflows:

| Workflow | File | Trigger |
|---|---|---|
| Go tests | `.github/workflows/ci.yml` | PR / push to `main` |
| Container image → GHCR | `.github/workflows/publish-image.yml` | `main`, tags `v*` (PRs build only) |
| Helm chart → GHCR OCI | `.github/workflows/publish-chart.yml` | `main`, tags `v*` |

## License

Apache License 2.0 — see [LICENSE](LICENSE).

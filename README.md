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
ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:0.1.1
ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:sha-<short>
```

```bash
# Install from the OCI Helm chart (recommended) — scaler only
helm install keda-otel-collector-fleet-scaler \
  oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler \
  --version 0.2.0 \
  -n observability --create-namespace

# Turnkey fleet (gateway + backends + ScaledObject) in the same release
helm install keda-otel-collector-fleet-scaler \
  oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler \
  --version 0.2.0 \
  -n observability --create-namespace \
  --set fleet.enabled=true

# Or pull the image and apply raw manifests
docker pull ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:latest
kubectl apply -f deploy/scaler/
```

GHCR packages created by CI start as **private**. Artifact Hub can only scan public images; a private package produces `error scanning image …: image not found`. After the first successful publish, make **both** packages Public (one-time each):

1. Open [github.com/jxtSamFrimpong?tab=packages](https://github.com/jxtSamFrimpong?tab=packages)
2. Click **`keda-otel-collector-fleet-scaler`** (app image) → **Package settings** (right sidebar) → **Change visibility** → **Public** → confirm
3. Click **`charts/keda-otel-collector-fleet-scaler`** (Helm OCI chart) → **Package settings** → **Change visibility** → **Public** → confirm

Verify anonymously (must work **without** `docker login`):

```bash
docker logout ghcr.io
docker manifest inspect ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:0.1.1
helm show chart oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler --version 0.1.1
```

**Artifact Hub `image not found` is not fixed by making packages public alone.** Chart `0.1.0` still points at an image published as an OCI index (+ provenance). AH’s scanner omits OCI `Accept` headers; GHCR then returns `MANIFEST_UNKNOWN`, which AH maps to `image not found`. That failure sticks on the scanned chart version: same-tag image overwrite and Control Panel refresh do **not** clear it. Fix by publishing a **new** chart version (`0.1.1+`) whose `artifacthub.io/images` tag matches an image built with Docker media types and no provenance/SBOM (`publish-image.yml`: `provenance: false`, `sbom: false`, `oci-mediatypes=false`), then wait for AH to index the new version.

Local development:

```bash
make test
make build
docker build -t keda-otel-collector-fleet-scaler:dev .
helm install keda-otel-collector-fleet-scaler charts/keda-otel-collector-fleet-scaler \
  -n observability --create-namespace --set fleet.enabled=true
```

### 2. Deploy a reference fleet

**Preferred:** Helm with `fleet.enabled=true` (chart ≥ 0.2.0). Requires KEDA CRDs when `fleet.scaledObject.enabled=true` (default), and for Datadog export a Secret `datadog-secret`.

Optional kustomize reference (not required for production Helm installs):

```bash
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

The gateway Service stays **ClusterIP** by default. For optional L4 LoadBalancer access (OTLP **4317** / **4318** only — never the scaler), set Helm values:

```yaml
fleet:
  enabled: true
  gateway:
    service:
      type: LoadBalancer
      annotations:
        service.beta.kubernetes.io/aws-load-balancer-type: "external"
        service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: "ip"
        service.beta.kubernetes.io/aws-load-balancer-scheme: "internal"
```

```bash
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

`deploy/examples/external-lb` remains an optional kustomize reference. No Ingress in this path.

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
  --version 0.2.0 \
  -n observability --create-namespace \
  --set fleet.enabled=true

helm show values oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler --version 0.2.0
```

### List the chart on Artifact Hub (one-time)

1. Ensure **both** GHCR packages are **Public** (exact clicks above):
   - `ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler` (app image — this is what AH Trivy scans via `artifacthub.io/images`)
   - `ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler` (Helm OCI chart)
2. Sign in at [artifacthub.io](https://artifacthub.io) → **Control Panel** → **Add repository**.
3. Kind: **Helm charts**. URL (OCI — one Artifact Hub repo per chart):

   ```text
   oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler
   ```

4. After Artifact Hub assigns a `repositoryID`, put it in [`artifacthub-repo.yml`](artifacthub-repo.yml) and let CI re-push the `artifacthub.io` OCI tag (or push manually with [oras](https://oras.land/) as described in the [Artifact Hub OCI docs](https://artifacthub.io/docs/topics/repositories/helm-charts/#oci-support)). That enables ownership claim / verified publisher.

**Scan errors / sticky `image not found`:** Public packages are necessary but not sufficient. Chart `0.1.0` failed because the image was an OCI index (+ provenance); AH’s scanner hit `MANIFEST_UNKNOWN` without OCI Accept headers. Same-tag overwrite and Control Panel refresh do **not** clear that sticky failure. Publish a new chart version (e.g. `0.1.1`) with `artifacthub.io/images` pointing at a Docker-mediatype image (`provenance: false`, `sbom: false`, `oci-mediatypes=false` in `publish-image.yml`). Do **not** whitelist the image unless you intentionally want to skip the security report.

Official docs: [Helm charts on Artifact Hub](https://artifacthub.io/docs/topics/repositories/helm-charts/) · [OCI support](https://artifacthub.io/docs/topics/repositories/helm-charts/#oci-support) · [Helm annotations](https://artifacthub.io/docs/topics/annotations/helm/) · [Security reports](https://artifacthub.io/docs/topics/security_report/).

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

## Releasing

A release is a **git tag** `vX.Y.Z` on `main`. Pushing that tag publishes both the container image and the Helm chart.

1. Bump chart `version` and `appVersion` together in `charts/keda-otel-collector-fleet-scaler/Chart.yaml` (keep `appVersion` aligned with the image semver Artifact Hub will scan). Leave `values.yaml` `image.tag` empty so the Deployment uses `Chart.AppVersion`.
2. Update `artifacthub.io/images` in `Chart.yaml` if the image reference changed.
3. Merge to `main`, then tag and push:

```bash
git tag v0.1.0
git push origin v0.1.0
```

4. `publish-image.yml` tags the multi-arch image as `X.Y.Z`, `X.Y`, plus `latest` / `sha-*` as configured. It must keep Docker media types (`provenance: false`, `sbom: false`, `oci-mediatypes=false`) so Artifact Hub can resolve the tag. `publish-chart.yml` packages and pushes the OCI chart at the chart `version`.
5. Confirm GHCR packages stay **Public**. AH indexes the **new** chart version; do not expect same-tag image overwrite or Control Panel refresh to clear a prior sticky scan failure on an older chart version.

Do **not** rely on `main`-only image publishes for Artifact Hub: pushes to `main` produce `latest` / `sha-*`, not the semver tag the chart’s `artifacthub.io/images` annotation declares (`…:0.1.1`). Keep owner paths **lowercase** (`jxtsamfrimpong`); GHCR rejects mixed-case refs.

## License

Apache License 2.0 — see [LICENSE](LICENSE).

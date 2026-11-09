# Community Packaging

## Repository layout

```text
.
├── api/externalscaler/     # KEDA ExternalScaler protobuf + generated Go
├── cmd/scaler/             # Scaler process entrypoint
├── pkg/
│   ├── config/             # Trigger metadata parsing + fleet profile presets
│   ├── metrics/            # Prometheus scrape + contract evaluation
│   └── scaler/             # gRPC ExternalScaler implementation
├── config/collector/       # Reference gateway/backend collector configs
├── deploy/
│   ├── scaler/             # Scaler Deployment/Service/RBAC manifests
│   └── examples/
│       ├── forward/        # Stateless Datadog-forward example
│       ├── external-lb/    # Optional LoadBalancer overlay for gateway OTLP
│       └── stateful/       # Tail-sampling example
├── charts/keda-otel-collector-fleet-scaler/# Helm chart (scaler + optional fleet templates)
├── docs/                   # Architecture and contracts
├── hack/                   # Codegen scripts
└── test/                   # Unit + e2e placeholders
```

## KEDA ecosystem fit

This project is an **external scaler** (trigger `type: external`), not a built-in
scaler merged into `kedacore/keda`. That matches how community scalers are usually
published first:

1. Standalone repo with clear gRPC contract
2. Example ScaledObjects and Helm chart
3. Optional listing under [KEDA external scalers](https://keda.sh/docs/latest/scalers/)
   / samples after the API stabilizes

Differentiate from [Kedify otel-add-on](https://github.com/kedify/otel-add-on):

| Project | Scales |
|---|---|
| Kedify OTel add-on | Applications, using app metrics pushed over OTLP |
| **This project** | OTel **collector fleets**, using collector self-telemetry + TraceID gateway pattern |

## Module / image

| Item | Value |
|---|---|
| Go module | `github.com/jxtSamFrimpong/keda-otel-collector-fleet-scaler` |
| Container image | `ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler` |
| Helm chart (OCI) | `oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler` |
| Default gRPC port | `9090` |
| Health HTTP port | `8080` (`/healthz`, `/readyz`) |

After the first CI publish, set each GHCR package visibility to **Public** in the GitHub UI (packages start private).

## Contribution checklist

- [ ] Keep TraceID routing in upstream `load_balancing` (do not fork)
- [ ] Document metric contract changes with ScaledObject metadata examples
- [ ] Preserve `forward` / `stateful` profile presets when changing defaults
- [ ] Add unit tests for metadata parsing and metric aggregation
- [ ] Run `make test` and `make build` before PR
- [ ] Update `docs/` when changing scaler behavior

## Release shape

1. Bump chart `version` and `appVersion` together before release (`appVersion` must match the image tag Artifact Hub scans)
2. Tag `vX.Y.Z` on `main` and push the tag (see README **Releasing**) — this is what publishes semver image tags (`X.Y.Z`, `X.Y`)
3. `publish-image.yml` builds multi-arch images to GHCR; `main` alone only pushes `latest` / `sha-*`
4. `publish-chart.yml` packages and pushes the Helm chart as OCI to `oci://ghcr.io/jxtsamfrimpong/charts`
5. Keep GHCR image + chart packages **Public**; refresh Artifact Hub after the first semver image lands
6. Register the OCI chart URL on [Artifact Hub](https://artifacthub.io) (one-time; see README)
7. Announce as a community external scaler for OTel collector fleets

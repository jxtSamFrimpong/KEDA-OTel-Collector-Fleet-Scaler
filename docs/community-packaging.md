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
| Container image | `keda-otel-collector-fleet-scaler` (override in Helm) |
| Default gRPC port | `9090` |
| Health HTTP port | `8080` (`/healthz`, `/readyz`) |

## Contribution checklist

- [ ] Keep TraceID routing in upstream `load_balancing` (do not fork)
- [ ] Document metric contract changes with ScaledObject metadata examples
- [ ] Preserve `forward` / `stateful` profile presets when changing defaults
- [ ] Add unit tests for metadata parsing and metric aggregation
- [ ] Run `make test` and `make build` before PR
- [ ] Update `docs/` when changing scaler behavior

## Release shape

1. Tag `v0.x` while the metric contract may still change
2. Publish multi-arch container images
3. Helm chart version aligned with appVersion
4. Announce as a community external scaler for OTel collector fleets

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

After the first CI publish, set each GHCR package visibility to **Public** in the GitHub UI (packages start private):

1. [Packages](https://github.com/jxtSamFrimpong?tab=packages) → `keda-otel-collector-fleet-scaler` → **Package settings** → **Change visibility** → **Public**
2. Same for `charts/keda-otel-collector-fleet-scaler`

Anonymous check: `docker logout ghcr.io` then `docker manifest inspect ghcr.io/jxtsamfrimpong/keda-otel-collector-fleet-scaler:0.1.1`.

**Artifact Hub sticky scan:** Chart `0.1.0` failed because the image was an OCI index (+ provenance). AH’s scanner omits OCI Accept headers → GHCR `MANIFEST_UNKNOWN` → AH `image not found`. Making packages Public, same-tag overwrite, Control Panel refresh, or re-running **Publish chart** for `0.1.0` alone does **not** clear it. Publish a new chart version (`0.1.1+`) with `artifacthub.io/images` on a Docker-mediatype image (`publish-image.yml`: `provenance: false`, `sbom: false`, `oci-mediatypes=false`).

## Contribution checklist

- [ ] Keep TraceID routing in upstream `load_balancing` (do not fork)
- [ ] Document metric contract changes with ScaledObject metadata examples
- [ ] Preserve `forward` / `stateful` profile presets when changing defaults
- [ ] Add unit tests for metadata parsing and metric aggregation
- [ ] Run `make test` and `make build` before PR
- [ ] Update `docs/` when changing scaler behavior

## Release shape

1. Bump chart `version` and `appVersion` together before release (`appVersion` must match the image tag Artifact Hub scans)
2. Keep `artifacthub.io/images` on that same semver tag; AH sticky failures on an older chart version need a **new** chart version, not same-tag overwrite
3. Tag `vX.Y.Z` on `main` and push the tag (see README **Releasing**) — this is what publishes semver image tags (`X.Y.Z`, `X.Y`)
4. `publish-image.yml` builds multi-arch images to GHCR with Docker media types (`provenance: false`, `sbom: false`, `oci-mediatypes=false`); `main` alone only pushes `latest` / `sha-*`
5. `publish-chart.yml` packages and pushes the Helm chart as OCI to `oci://ghcr.io/jxtsamfrimpong/charts`
6. Keep GHCR image + chart packages **Public**; wait for AH to index the new chart version (refresh alone does not clear sticky `image not found`)
7. Register the OCI chart URL on [Artifact Hub](https://artifacthub.io) (one-time; see README)
8. Announce as a community external scaler for OTel collector fleets

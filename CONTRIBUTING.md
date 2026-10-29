# Contributing

## Scope

Keep this project focused on **scaling OTel collector fleets** with KEDA.
TraceID routing stays in upstream `load_balancing` — do not fork or reimplement it here.

## Workflow

1. Read [docs/architecture.md](docs/architecture.md) and the metric / profile / scale-safety docs.
2. Run `make test` and `make build`.
3. If changing the ExternalScaler protobuf, run `make generate` (requires `protoc`).
4. Update docs when changing ScaledObject metadata or profile defaults.

## PR checklist

- [ ] Unit tests for metadata / aggregation changes
- [ ] Example manifests still match docs
- [ ] `forward` / `stateful` presets stay coherent with drain guidance

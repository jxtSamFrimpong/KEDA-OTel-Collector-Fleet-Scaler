# Fleet Profiles

Platform teams decide whether backends are stateful via their collector
`config.yaml`. This project does **not** parse that file. Instead, declare a
**fleet profile** that sets drain, HPA behavior, and workload defaults.

## Profiles

| Profile | When to use | Workload default | Drain (`preStop`) | Scale-in stabilization |
|---|---|---|---|---|
| `forward` | Stateless forward to Datadog/OTLP; no tail sampling / groupbytrace holding spans | Deployment | `15s` | `60s` |
| `stateful` | Tail sampling, spanmetrics with affinity needs, `groupbytrace`, or other in-memory hold | StatefulSet | `≥ decision_wait` (default `35s` for 30s wait) | `300s` |

Both profiles share:

- Gateway Deployment (≥2 replicas) with `load_balancing` + k8s resolver
- Headless backend Service as membership source
- Same KEDA external scaler and metric contract

## Helm / values

```yaml
fleet:
  profile: forward   # or stateful

backend:
  # overridden by profile defaults unless set explicitly
  kind: ""           # Deployment | StatefulSet
  preStopSeconds: 0
  minReplicas: 2
  maxReplicas: 20

keda:
  pollingInterval: 15
  cooldownPeriod: 60
  advanced:
    horizontalPodAutoscalerConfig:
      behavior: {}   # filled from profile presets if empty
```

## Presets

### `forward`

```yaml
backend:
  kind: Deployment
  preStopSeconds: 15
  terminationGracePeriodSeconds: 30

keda:
  cooldownPeriod: 60
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          stabilizationWindowSeconds: 30
          policies:
            - type: Pods
              value: 2
              periodSeconds: 60
        scaleDown:
          stabilizationWindowSeconds: 60
          policies:
            - type: Pods
              value: 1
              periodSeconds: 60
```

### `stateful`

```yaml
backend:
  kind: StatefulSet
  preStopSeconds: 35
  terminationGracePeriodSeconds: 60
  # Headless Service must be StatefulSet .spec.serviceName when using return_hostnames

keda:
  cooldownPeriod: 300
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleUp:
          stabilizationWindowSeconds: 60
          policies:
            - type: Pods
              value: 1
              periodSeconds: 120
        scaleDown:
          stabilizationWindowSeconds: 300
          policies:
            - type: Pods
              value: 1
              periodSeconds: 180
```

Tune `preStopSeconds` to at least the longest in-memory hold in your config
(e.g. `tail_sampling::decision_wait`, `groupbytrace` wait).

## ScaledObject annotation

Examples also label the ScaledObject:

```yaml
metadata:
  annotations:
    keda.otel.collector-fleet.scaler/fleet-profile: forward
```

The scaler binary does not require this annotation; it is documentation and a
hook for future admission/validation controllers.

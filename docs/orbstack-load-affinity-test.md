# OrbStack load + TraceID affinity test

Always use `--context orbstack`. Do not change the default kubectl context.

Scaling signals are collector self-telemetry (`exporter_queue_utilization`,
`memory_rss_bytes`, …) — see [`metric-contract.md`](metric-contract.md).

## Method

1. Backends export with **debug / detailed** so TraceIDs appear in pod logs.
2. Job `otlp-affinity-test` sends 5 known TraceIDs × 5 staggered spans to `otel-gateway.observability.svc:4317`.
3. Grep each backend’s logs for `Trace ID       : <id>` — each TraceID must appear on **exactly one** backend.
4. Lower ScaledObject `memory_rss_bytes` target (demo-only) and/or run `otlp-load-test` so HPA scales backends **out** above min.
5. Delete load Jobs, restore the calmer `exporter_queue_utilization` target (`0.65`), and wait for **scale-in** to `minReplicaCount` (2) under the `forward` profile windows (60s scale-down stabilization, 1 pod / 60s).

Manifests: [`deploy/demo/affinity-loadgen.yaml`](../deploy/demo/affinity-loadgen.yaml)

## Re-run (affinity + scale-out)

```bash
# Ensure detailed debug backends (local demo)
kubectl --context orbstack apply -f deploy/demo/affinity-loadgen.yaml

# Optional: force scale-out with a low memory target
kubectl --context orbstack -n observability patch scaledobject otel-backends --type=merge -p '{
  "spec":{"triggers":[{"type":"external","metadata":{
    "scalerAddress":"keda-otel-collector-fleet-scaler.observability.svc:9090",
    "metric":"memory_rss_bytes","targetValue":"20000000","aggregation":"avg",
    "backendService":"otel-backends","backendNamespace":"observability",
    "prometheusPort":"8888","prometheusPath":"/metrics","activationThreshold":"0",
    "metricName":"otelcol_backend_memory_rss_bytes"
  }}]}
}'

kubectl --context orbstack -n observability delete job otlp-affinity-test otlp-load-test --ignore-not-found
kubectl --context orbstack apply -f deploy/demo/affinity-loadgen.yaml

# Affinity matrix
for tid in aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb \
           cccccccccccccccccccccccccccccccc dddddddddddddddddddddddddddddddd \
           eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee; do
  echo "== $tid =="
  for p in $(kubectl --context orbstack -n observability get pods -l app.kubernetes.io/name=otel-backends -o jsonpath='{.items[*].metadata.name}'); do
    c=$(kubectl --context orbstack -n observability logs "$p" --since=30m | grep -c "Trace ID       : $tid" || true)
    echo "  $p $c"
  done
done

kubectl --context orbstack -n observability get hpa,deploy/otel-backends,pods -l app.kubernetes.io/name=otel-backends
```

## Scale-in

After scale-out (backends above min), restore pressure and wait for orderly leave:

```bash
# Stop load generators
kubectl --context orbstack -n observability delete job otlp-affinity-test otlp-load-test --ignore-not-found

# Restore forward-profile queue utilization target
kubectl --context orbstack -n observability patch scaledobject otel-backends --type=merge -p '{
  "spec":{
    "minReplicaCount":2,"maxReplicaCount":6,"cooldownPeriod":60,"pollingInterval":15,
    "advanced":{"horizontalPodAutoscalerConfig":{"behavior":{
      "scaleUp":{"stabilizationWindowSeconds":30,"policies":[{"type":"Pods","value":2,"periodSeconds":60}]},
      "scaleDown":{"stabilizationWindowSeconds":60,"policies":[{"type":"Pods","value":1,"periodSeconds":60}]}
    }}},
    "triggers":[{"type":"external","metadata":{
      "scalerAddress":"keda-otel-collector-fleet-scaler.observability.svc:9090",
      "metric":"exporter_queue_utilization","targetValue":"0.65","aggregation":"avg",
      "backendService":"otel-backends","backendNamespace":"observability",
      "prometheusPort":"8888","prometheusPath":"/metrics","activationThreshold":"0.05",
      "metricName":"otelcol_backend_exporter_queue_utilization"
    }}]
  }
}'

# Watch HPA step down 1 pod / ~60s toward min=2; EndpointSlices should shrink with Ready endpoints
watch -n 10 'kubectl --context orbstack -n observability get hpa keda-hpa-otel-backends deploy/otel-backends; \
  kubectl --context orbstack -n observability get pods -l app.kubernetes.io/name=otel-backends; \
  kubectl --context orbstack -n observability get endpointslices -l kubernetes.io/service-name=otel-backends'

# Confirm gateway stays Ready with no restarts during leave
kubectl --context orbstack -n observability get pods -l app.kubernetes.io/name=otel-gateway
```

Expect: backends return to 2 Ready pods; terminating pods keep `preStop` sleep (`15s` on the `forward` profile) before exit; gateway pods remain Ready with restart count unchanged.

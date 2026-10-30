# OrbStack load + TraceID affinity test

Always use `--context orbstack`. Do not change the default kubectl context.

## Method

1. Backends export with **debug / detailed** so TraceIDs appear in pod logs.
2. Job `otlp-affinity-test` sends 5 known TraceIDs × 5 staggered spans to `otel-gateway.observability.svc:4317`.
3. Grep each backend’s logs for `Trace ID       : <id>` — each TraceID must appear on **exactly one** backend.
4. Lower ScaledObject `memory_rss_bytes` target (demo-only) and/or run `otlp-load-test` so HPA scales backends above min.

Manifests: [`deploy/demo/affinity-loadgen.yaml`](../deploy/demo/affinity-loadgen.yaml)

## Re-run

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

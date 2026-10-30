# OrbStack local test

Always target OrbStack explicitly. Do **not** switch the default kubectl context.

```bash
CTX=orbstack   # use: kubectl --context "$CTX" ...
               # helm --kube-context "$CTX" ...
```

## One-time setup

```bash
# Enable Kubernetes in OrbStack if needed
orbctl config set k8s.enable true
orbctl stop && orbctl start

# Restore your default context if OrbStack flipped it
kubectl config use-context <your-eks-or-other-context>

kubectl --context orbstack get nodes
```

## Install KEDA

```bash
helm repo add kedacore https://kedacore.github.io/charts
helm upgrade --install keda kedacore/keda \
  --kube-context orbstack -n keda --create-namespace --wait
```

If image pulls are slow, pre-pull into OrbStack Docker:

```bash
docker pull ghcr.io/kedacore/keda:2.21.0
docker pull ghcr.io/kedacore/keda-metrics-apiserver:2.21.0
docker pull ghcr.io/kedacore/keda-admission-webhooks:2.21.0
```

## Build / load scaler image

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/scaler-linux ./cmd/scaler
docker build -f Dockerfile.local -t keda-otel-collector-fleet-scaler:dev .
```

## Deploy

```bash
kubectl --context orbstack apply -f deploy/scaler/
kubectl --context orbstack -n observability patch deploy keda-otel-collector-fleet-scaler \
  --type='json' -p='[{"op":"replace","path":"/spec/template/spec/containers/0/imagePullPolicy","value":"IfNotPresent"}]'

kubectl --context orbstack apply -k deploy/examples/forward
```

The forward example uses a `debug` backend exporter for local runs (see `deploy/examples/forward/config/backend-debug.yaml`). Gateway uses `loadbalancing` + k8s resolver and needs Endpoints + EndpointSlice list/watch RBAC.

## Verify

```bash
kubectl --context orbstack get pods,svc,scaledobject,hpa -n observability
kubectl --context orbstack get pods -n keda

# Backend self-metrics
BP=$(kubectl --context orbstack -n observability get pod -l app.kubernetes.io/name=otel-backends -o jsonpath='{.items[0].metadata.name}')
kubectl --context orbstack -n observability port-forward pod/$BP 18888:8888
curl -s localhost:18888/metrics | grep otelcol_process_memory_rss
```

Idle fleets stay `ACTIVE=False` until the trigger exceeds `activationThreshold`. HPA needs `keda-operator-metrics-apiserver` Ready.

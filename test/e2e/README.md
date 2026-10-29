# End-to-end tests

Placeholder for cluster e2e covering:

1. Scale-out: Ready endpoint joins ring before traffic
2. Scale-in: EndpointSlice removal during `preStop` sleep
3. Scaler `GetMetrics` reflects queue utilization across backends

Run unit tests locally with `make test` until a Kind/k3d harness is added.

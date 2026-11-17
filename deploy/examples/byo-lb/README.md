# BYO LoadBalancer (reuse existing NLB)

Keep the gateway Service as **ClusterIP** and register pods into target groups
on an **existing** Network Load Balancer via AWS Load Balancer Controller
`TargetGroupBinding`.

## When to use

| Mode | How | Use when |
|------|-----|----------|
| **BYO** (this overlay) | ClusterIP + TargetGroupBinding | Platform already owns an NLB / TGs |
| **Dedicated** | `deploy/examples/external-lb` or Helm `type: LoadBalancer` | Chart should provision a new NLB |

## Apply

1. Create (or reuse) an internal NLB with TCP listeners on **4317** and **4318**.
2. Create target groups (`target-type: ip`) and attach them to those listeners.
3. Replace `REPLACE_WITH_*` ARNs in `targetgroupbinding.yaml`.
4. Ensure security groups allow NLB → node/pod ENIs on 4317/4318.
5. Apply:

```bash
kubectl apply -k deploy/examples/byo-lb
```

## Helm equivalent

```yaml
fleet:
  enabled: true
  gateway:
    service:
      type: ClusterIP
    targetGroupBinding:
      enabled: true
      targetType: ip
      otlpGrpcTargetGroupARN: arn:aws:elasticloadbalancing:...:targetgroup/...
      otlpHttpTargetGroupARN: arn:aws:elasticloadbalancing:...:targetgroup/...
```

## Notes

- Do **not** combine BYO TGB with `type: LoadBalancer` (that creates a second LB).
- Ingress / ALB is a poor fit for raw OTLP L4 — prefer NLB.
- Scaler gRPC (`:9090`) stays ClusterIP-only; never put it on the NLB.

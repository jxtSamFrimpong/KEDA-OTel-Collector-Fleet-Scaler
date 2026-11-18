# Bring-your-own (BYO) Load Balancer

Prefer **BYO** for production: keep the gateway Service as **ClusterIP** and attach
pods to an existing (or platform-owned) Network Load Balancer via AWS Load
Balancer Controller `TargetGroupBinding`. Do **not** put scaler port `:9090` on
any external LB.

## Why BYO

| Concern | BYO (ClusterIP + TGB) | Chart `type: LoadBalancer` |
|---------|------------------------|----------------------------|
| Security groups / client whitelist | Owned in AWS/Terraform | Split across annotations + controller |
| Subnets / scheme (internal) | Explicit LB config | Annotation-driven discovery |
| LB lifecycle | Platform owns NLB independently of Helm | LB created/destroyed with the Service |
| Sticky chart churn | Chart upgrades do not recreate the NLB | Service annotation drift can recreate LB |
| Multi-tenant / shared NLB | Reuse one NLB + extra listeners/TGs | One LB per Service |

Ingress / ALB is a poor fit for raw OTLP L4 (gRPC/HTTP on fixed ports **4317** /
**4318**). Use an NLB (or equivalent L4 LB).

## Modes at a glance

| Mode | Gateway Service | Who creates the LB |
|------|-----------------|--------------------|
| **In-cluster only** | `ClusterIP` | None |
| **BYO** (preferred) | `ClusterIP` + `TargetGroupBinding` | You / Terraform / platform |
| **Dedicated** | `type: LoadBalancer` + annotations | Cloud LB controller via the Service |

`fleet.enabled=false` (scaler-only) is unchanged — no gateway Service or TGB.

## BYO steps (AWS + EKS)

Prerequisites: **AWS Load Balancer Controller** installed with IRSA, and its
`TargetGroupBinding` CRDs available on the cluster.

1. **Create (or reuse) an internal NLB** in the cluster VPC (private subnets tagged
   for internal ELB if you rely on auto-discovery elsewhere).
2. **Attach a security group** to the NLB (NLB SG support) and whitelist OTLP
   clients on TCP **4317** and **4318** (CIDRs and/or peer SGs).
3. **Create two target groups** (`target-type: ip` for EKS pod ENIs):
   - OTLP gRPC → port **4317**, TCP health check
   - OTLP HTTP → port **4318**, TCP health check
4. **Create listeners** on the NLB: TCP 4317 → gRPC TG, TCP 4318 → HTTP TG.
5. **Allow NLB → nodes/pods**: node security group ingress from the NLB SG on
   4317/4318 (required when the controller does not manage backend SG rules).
6. **Deploy the chart** with ClusterIP + TargetGroupBinding ARNs (below), **or**
   apply the kustomize reference after filling ARNs.
7. **Point clients** at the NLB DNS name: `:4317` (gRPC) / `:4318` (HTTP).

## Helm values (BYO)

```yaml
fleet:
  enabled: true
  gateway:
    service:
      type: ClusterIP          # required for BYO — do not set LoadBalancer
      annotations: {}
    targetGroupBinding:
      enabled: true
      targetType: ip           # must match the target groups
      otlpGrpcTargetGroupARN: arn:aws:elasticloadbalancing:REGION:ACCOUNT:targetgroup/otel-grpc/...
      otlpHttpTargetGroupARN: arn:aws:elasticloadbalancing:REGION:ACCOUNT:targetgroup/otel-http/...
```

Install (example):

```bash
helm upgrade --install keda-otel-collector-fleet-scaler \
  oci://ghcr.io/jxtsamfrimpong/charts/keda-otel-collector-fleet-scaler \
  --version 0.2.2 \
  -n observability --create-namespace \
  -f values-byo.yaml
```

Requires chart **≥ 0.2.1** for in-chart `fleet.gateway.targetGroupBinding`.

## Example TargetGroupBinding YAML

Same shape as the optional chart templates and
[`deploy/examples/byo-lb/targetgroupbinding.yaml`](../deploy/examples/byo-lb/targetgroupbinding.yaml):

```yaml
apiVersion: elbv2.k8s.aws/v1beta1
kind: TargetGroupBinding
metadata:
  name: otel-gateway-otlp-grpc
  namespace: observability
spec:
  serviceRef:
    name: otel-gateway
    port: 4317
  targetGroupARN: REPLACE_WITH_OTLP_GRPC_TARGET_GROUP_ARN
  targetType: ip
---
apiVersion: elbv2.k8s.aws/v1beta1
kind: TargetGroupBinding
metadata:
  name: otel-gateway-otlp-http
  namespace: observability
spec:
  serviceRef:
    name: otel-gateway
    port: 4318
  targetGroupARN: REPLACE_WITH_OTLP_HTTP_TARGET_GROUP_ARN
  targetType: ip
```

Kustomize apply (after replacing ARNs):

```bash
kubectl apply -k deploy/examples/byo-lb
```

## Contrast: dedicated `Service type: LoadBalancer`

Use when you want the chart/controller to provision a **new** NLB for this
release only:

```yaml
fleet:
  enabled: true
  gateway:
    service:
      type: LoadBalancer
      annotations:
        service.beta.kubernetes.io/aws-load-balancer-type: "external"
        service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: "ip"
        service.beta.kubernetes.io/aws-load-balancer-scheme: "internal"
        # optional: pin SG / subnets
        # service.beta.kubernetes.io/aws-load-balancer-security-groups: "sg-..."
        # service.beta.kubernetes.io/aws-load-balancer-subnets: "subnet-a,subnet-b"
```

Kustomize reference: [`deploy/examples/external-lb`](../deploy/examples/external-lb).

**Do not** combine dedicated `LoadBalancer` with BYO `TargetGroupBinding` — that
registers pods on two LBs.

## Checklist

- [ ] LBC + TargetGroupBinding CRDs present
- [ ] NLB + listeners 4317/4318 + TGs (`ip`)
- [ ] NLB SG client whitelist set
- [ ] Node SG allows NLB SG → 4317/4318
- [ ] Gateway Service remains `ClusterIP`
- [ ] TGB ARNs match the TGs (and `targetType`)
- [ ] Clients use NLB DNS, not the scaler Service
- [ ] Scaler `:9090` is not on the NLB

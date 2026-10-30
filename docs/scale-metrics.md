# Scale Metrics (Plain Language)

This page explains **what the fleet scales on**, in everyday terms. For the full
ScaledObject metadata reference (keys, defaults, units), see
[metric-contract.md](metric-contract.md).

## What the fleet scales on

KEDA asks this project’s external scaler for a number. That number comes from
**collector self-telemetry**: Prometheus metrics scraped from each **backend**
collector on port **8888** (`/metrics`).

It is **not** application APM. The scaler does not look at your apps’ traces,
latency, or request rates. It only looks at how healthy and busy the **backend
collectors** themselves are (queue depth, memory, refused spans, and so on).

```text
Apps → Gateway → Backend collectors (:8888 /metrics)
                      ↑ scrape Ready pods only
                 fleet scaler → KEDA → more/fewer backends
```

## Choosing a metric

| Contract name | Best when… | Typical `targetValue` |
|---|---|---|
| `exporter_queue_utilization` | You export with a `sending_queue` (most forward fleets) | `"0.65"` |
| `exporter_queue_size` | You care about absolute queue depth, not fill ratio | `"500"` (tune to `queue_size`) |
| `memory_rss_bytes` | Backends hold spans in memory (tail sampling, groupbytrace) | `"2147483648"` (2 GiB) |
| `exporter_enqueue_failed_spans` | Activation / alert when the queue rejects work | rate threshold (e.g. `"1"`) |
| `receiver_refused_spans` | Activation / alert for memory_limiter backpressure | rate threshold (e.g. `"1"`) |

Most people pick **one primary HPA signal** (`exporter_queue_utilization` or
`memory_rss_bytes`) and keep the refused/failed counters for alerts or
`activationThreshold`, not as the only scale target.

---

### `exporter_queue_utilization`

**What it measures.** How full each backend’s exporter send queue is, as a
ratio from **0 to 1**: `queue_size / queue_capacity` (from
`otelcol_exporter_queue_*`). The scaler aggregates that per-pod ratio across
Ready backends.

**When to choose it.** Default choice for **forward** fleets that push to
Datadog (or another remote) with `sending_queue` enabled. Queue fill tracks
export pressure before the pod OOMs.

**What “too high” means.** The queue is filling faster than exporters can drain
it. Sustained values near **1.0** mean you are about to drop or fail enqueues.
Scale out before that—commonly when average utilization sits around
**0.6–0.7**.

**Example**

```yaml
metric: exporter_queue_utilization
targetValue: "0.65"
aggregation: avg
```

---

### `exporter_queue_size`

**What it measures.** Absolute exporter queue depth (batches / items in the
send queue), not the fill ratio.

**When to choose it.** You already know a meaningful absolute depth relative to
`sending_queue.queue_size` in `config.yaml`, or capacity is fixed and you prefer
raw counts over ratios.

**What “too high” means.** Depth is climbing toward your configured
`queue_size`. If capacity is 1000 and you sit at 800+ for long, you are close to
overflow—same story as high utilization, expressed in absolute terms.

**Example**

```yaml
metric: exporter_queue_size
targetValue: "500"
aggregation: avg
```

---

### `memory_rss_bytes`

**What it measures.** Process RSS of the collector (`otelcol_process_memory_rss`),
in bytes—how much memory each Ready backend is actually using.

**When to choose it.** **Stateful** fleets that hold spans in memory
(`tail_sampling`, `groupbytrace`, large in-process buffers). Queue metrics alone
may stay calm while RSS climbs toward the pod limit or `memory_limiter`.

**What “too high” means.** Backends are approaching the memory budget you sized
for (pod limit and/or `memory_limiter.limit_mib`). Scale out so TraceIDs spread
across more pods before the limiter starts refusing data or the kubelet OOMs
the pod.

**Example** (target ~2 GiB average RSS):

```yaml
metric: memory_rss_bytes
targetValue: "2147483648"
aggregation: avg
```

---

### `exporter_enqueue_failed_spans`

**What it measures.** Spans that failed to enter the exporter send queue
(counter; the scaler may expose a rate over the scrape window).

**When to choose it.** As a **secondary / activation** signal or alert—not as
the only HPA target. Prefer `exporter_queue_utilization` for steady-state
scaling; use this when you want “scale or wake when the queue is already
rejecting work.”

**What “too high” means.** The queue is full or broken enough that new spans
cannot enqueue. That is already a failure mode; scaling helps recovery, but you
should have scaled earlier on utilization.

**Example**

```yaml
metric: exporter_enqueue_failed_spans
targetValue: "1"
activationThreshold: "0"
```

---

### `receiver_refused_spans`

**What it measures.** Spans refused at the receiver—often because
`memory_limiter` (or other backpressure) is protecting the process.

**When to choose it.** Activation or alerting when backends are under memory
pressure. Not ideal as the sole scale target: by the time refuses climb, you are
already shedding load.

**What “too high” means.** Collectors are rejecting inbound spans to stay
alive. Add capacity (and check `limit_mib` / pod requests) so traffic is
accepted again.

**Example**

```yaml
metric: receiver_refused_spans
targetValue: "1"
activationThreshold: "0"
```

---

## How aggregation works (`avg` / `max` / `sum`)

The scaler only scrapes **Ready** backend pods for the Service you name in
`backendService`. Then:

1. Compute the contract value **per pod** (e.g. that pod’s queue utilization).
2. Combine pods with `aggregation`:
   - **`avg`** (default) — typical fleet pressure. Good for smooth HPA behavior.
   - **`max`** — scale when **any** hot pod is bad. More aggressive; useful if
     load is skewed.
   - **`sum`** — total across pods. Rare for ratios; sometimes used for absolute
     counters or sizes when you want fleet-wide volume.

Empty Ready set → value `0` and inactive (unless you set a negative
`activationThreshold`). Details:
[Aggregation semantics](metric-contract.md#aggregation-semantics).

## Relation to `memory_limiter` and `sending_queue`

Platform teams own collector `config.yaml`. Two knobs matter for these metrics:

**`processors.memory_limiter`** — Soft memory guard. When usage crosses the
configured limit, the collector refuses or drops data rather than OOM. That
shows up as `receiver_refused_spans` (and related pressure). Scaling on
`memory_rss_bytes` aims to add pods **before** the limiter has to refuse.

**`exporters.*.sending_queue`** — Outbound buffer toward Datadog/OTLP. With the
queue enabled, the collector emits `otelcol_exporter_queue_size` and
`otelcol_exporter_queue_capacity`, which power `exporter_queue_utilization` and
`exporter_queue_size`. Without a sending queue, queue-based scale signals are
weak or meaningless.

Queue scaling answers “can we export fast enough?” Memory scaling answers “are
we holding too much in-process?” Many fleets need both ideas operationally even
if only one metric drives the HPA.

## Forward vs stateful: what people usually pick

| Fleet profile | Common primary metric | Why |
|---|---|---|
| **`forward`** | `exporter_queue_utilization` | Stateless pass-through; pressure shows up in the export queue first |
| **`stateful`** | `memory_rss_bytes` | Tail sampling / in-memory hold grows RSS even when the export queue looks fine |

Same scaler and metric contract for both profiles—only the **chosen signal** and
drain/HPA presets differ. See [fleet-profiles.md](fleet-profiles.md).

## Full metadata reference

ScaledObject keys (`metric`, `targetValue`, `aggregation`, `backendService`,
ports, activation, and more) are defined in
[metric-contract.md](metric-contract.md).

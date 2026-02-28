# GPU Economic Telemetry Agent — Detailed Technical Specification

This document defines the complete technical specification for the GPU Economic Telemetry Agent: requirements, data models, configuration schema, OpenTelemetry contracts, error handling, and compliance rules.

**Implementation language:** The service is implemented in **Go (Golang)**.

---

## 1. Overview

### 1.1 Purpose

The GPU Economic Telemetry Agent is a **node-level infrastructure daemon** written in Go that:

- Collects GPU telemetry (power, utilization, memory, processes) from NVIDIA GPUs via NVML
- Aggregates energy usage over configurable time windows
- Performs cost estimation using electricity rate and PUE (Power Usage Effectiveness)
- Exports OpenTelemetry metrics and traces via OTLP (gRPC)
- Runs as a long-lived, stateless background service

### 1.2 Design Principles

| Principle | Description |
|-----------|-------------|
| **Stateless** | No persistent storage; all aggregation in-memory only |
| **Configuration-driven** | Behavior fully controlled via YAML config; no runtime code loading |
| **Observability-first** | Exports OTel metrics/traces; no custom protocols |
| **Minimal privilege** | Non-root preferred; least privilege filesystem and network access |
| **Deterministic** | Bounded allocations; no dynamic plugins; predictable polling |

### 1.3 Supported Deployment Environments

- **Bare-metal Linux** — systemd service, static binary
- **Kubernetes** — containerized, DaemonSet, with `/dev/nvidia*` and `/proc` mounts
- **VM-based GPU clusters** — same binary; NVML available inside VM

### 1.4 Service Classification

| Attribute | Value |
|-----------|--------|
| Language | Go (Golang) |
| Type | Infrastructure Agent |
| Runtime | Long-lived daemon |
| Execution model | Background service; no interactive CLI |
| State | In-memory aggregation only |
| Privilege | Requires NVML access and read-only `/proc` (and optionally cgroups) |

---

## 2. Functional Requirements

### 2.1 Telemetry Collection

#### 2.1.1 Per-GPU Metrics (Required)

| Metric | Source | Unit | Type | Notes |
|--------|--------|------|------|--------|
| Instantaneous power draw | NVML | Watts (W) | float64 | Current power consumption |
| SM utilization | NVML | Percent (0–100) | float64 | Streaming multiprocessor utilization |
| Memory utilization | NVML | Percent (0–100) | float64 | Used / total GPU memory |
| GPU temperature | NVML | Celsius | float64 | Current temperature |
| Power limit | NVML | Watts (W) | float64 | Configured power cap |

**Collection interval:** Default **5 seconds**; configurable (min 1s, max 300s). Same interval used for all GPUs on the node.

#### 2.1.2 Per-Process Metrics (Optional)

When `process_attribution.enabled: true`:

| Attribute | Source | Type | Notes |
|-----------|--------|------|--------|
| PID | Process table / NVML | int64 | Process identifier |
| GPU memory usage | NVML | bytes (uint64) | Per-process GPU memory |
| GPU utilization | NVML (if available) | percent | Per-process utilization |

Process attribution may be skipped safely if process exits between poll and mapping (no crash).

#### 2.1.3 Collection Semantics

- **Polling model:** Synchronous poll at fixed interval; no event-driven NVML.
- **Missing values:** If a GPU or metric is unavailable (e.g., driver reset), emit nothing for that GPU/metric for that interval; do not extrapolate.
- **Multi-GPU:** Enumerate all GPUs visible to NVML on the node; collect per device.

### 2.2 Energy Aggregation (Riemann Sum Approximation)

Energy is computed using a **trapezoidal (Riemann sum) approximation** over each polling interval: the average of previous and current power multiplied by the interval duration. This is the correct production formula for accuracy.

#### 2.2.1 Per-Interval Energy (Trapezoidal Rule)

For each interval between two consecutive power samples:

```
E_interval = ((P_prev + P_current) / 2) * delta_t
```

- **P_prev:** Power (Watts) at the start of the interval (previous sample).
- **P_current:** Power (Watts) at the end of the interval (current sample).
- **delta_t:** Elapsed time in seconds (monotonic or wall clock; see Edge Cases).
- **E_interval:** Energy in Watt-seconds (Joules) for that interval.

#### 2.2.2 Accumulation and Conversion to kWh

Sum all interval energies, then convert to kilowatt-hours:

```
E_total = sum(E_interval)   [Watt-seconds]
E_kWh   = E_total / 3,600,000
```

So cumulative energy in kWh is: **E_kWh = E_total / 3,600,000**.

#### 2.2.3 Aggregation Windows

- **Rolling window (metrics):** Energy (and thus cost) is accumulated in-memory per GPU and exported as counters (cumulative since process start or since last reset). No sliding window required for v1.
- **Job-level (traces):** When job lifecycle is known (e.g., future K8s integration), accumulate E_interval (and thus E_kWh and cost) per job and emit a single span on job end.

#### 2.2.4 Data Types

- All energy values: **float64**, unit **kWh** (after conversion).
- All power values: **float64**, unit **Watts**.
- All monetary cost values: **float64**; currency implied by config (e.g., USD); no currency code in v1.

#### 2.2.5 Idle vs Active Energy

Energy is classified per interval by utilization: if SM utilization is below a configurable threshold (`cost.idle_utilization_threshold_percent`, default 10%), the interval energy is counted as **idle**; otherwise as **active**. Both are exported as cumulative counters (`gpu.energy.kwh.idle`, `gpu.energy.kwh.active`) so backends and dashboards can report idle vs active economic insight. Total energy remains `gpu.energy.kwh.total` (sum of both).

#### 2.2.6 Power Validation and Counter Reset

- **Power sanity checks:** The agent evaluates per-GPU readings and exposes anomaly gauges: e.g. suspiciously low power (&lt; 5W for a discrete GPU, e.g. power limit max >= 50W) and NVML reporting power limit zero. These are exposed as `gpu.anomaly.*` gauges (0 or 1) for observability.
- **Counter reset:** On daemon restart, energy and cost counters reset to zero. The agent emits `agent.restart.count` (value 1 per process start) so backends can detect restarts and avoid misinterpreting counter drops.

### 2.3 Cost Engine and Full Economic Pipeline

After energy aggregation (E_kWh per GPU or per job), the cost engine applies PUE and electricity rate. The **full economic pipeline** is:

1. **Per interval (Riemann sum):**  
   `E_interval = (P_prev + P_current) / 2 * delta_t`
2. **Accumulate:**  
   `E_total = sum(E_interval)`
3. **Convert to kWh:**  
   `E_kWh = E_total / 3,600,000`
4. **Apply PUE:**  
   `E_effective = E_kWh * PUE`
5. **Cost:**  
   `Cost = E_effective * cost_per_kWh`

#### 2.3.1 Inputs (Configuration)

| Parameter | Type | Unit | Description |
|-----------|------|------|-------------|
| `electricity_cost_per_kwh` | float64 | Currency per kWh (e.g., USD/kWh) | Price of electricity |
| `pue` | float64 | Dimensionless | Power Usage Effectiveness (datacenter) |

#### 2.3.2 Calculations

```
E_effective = E_kWh * PUE
Cost        = E_effective * electricity_cost_per_kwh
```

- **E_kWh:** Energy from aggregation (see 2.2); uses Riemann sum approximation.
- **PUE:** Applied to reflect datacenter overhead (cooling, PDU, etc.).
- **Cost:** Stored and exported as **float64**; no rounding specified (implementation may round for display only).

#### 2.3.3 Outputs

- **cost.estimated** (or equivalent): Cost in configured currency, attached to metrics (counters) and to job summary spans.

---

## 3. Data Models

### 3.1 Internal Telemetry Snapshot (Per Poll)

Logical structure produced by the collector and consumed by aggregator/cost engine:

```go
// GPUSample represents one GPU at one point in time.
type GPUSample struct {
    GPUIndex     int     // 0-based index
    PowerWatts   float64
    SMUtilPct    float64
    MemUtilPct   float64
    TemperatureC float64
    PowerLimitW  float64
}

// NodeSample is the full snapshot for the node at one poll.
type NodeSample struct {
    Timestamp   time.Time   // Monotonic or wall clock per config
    GPUs        []GPUSample
    ProcessInfo []ProcessGPUUsage // If process_attribution enabled
}
```

### 3.2 Aggregated State (In-Memory)

- **Per GPU:** E_total (Watt-seconds) accumulated via `E_interval = (P_prev + P_current) / 2 * delta_t` (Riemann sum); then E_kWh = E_total / 3,600,000.
- **Cumulative energy per GPU:** `map[gpuID]E_kWh` (float64).
- **Cumulative cost per GPU:** `map[gpuID]cost` (float64), where cost = E_kWh * PUE * cost_per_kWh.
- **Optional:** Rolling window of recent power samples for histogram (if implemented).

No persistence; state is lost on restart (counters reset).

### 3.3 Process Attribution (Optional)

```go
// ProcessGPUUsage links a process to GPU usage.
type ProcessGPUUsage struct {
    PID           int64
    GPUIndex      int
    MemoryBytes   uint64
    UtilizationPct float64 // If available
}
```

---

## 4. Configuration Specification

### 4.1 Configuration File (YAML)

Single file path provided via CLI flag or env (e.g., `--config /etc/gpu-econ-agent/config.yaml`).

### 4.2 Schema (Logical)

```yaml
# Cluster and node identity (used in OTel resource attributes).
cluster_id: string                    # Required. E.g. "research-cluster-01"
node_id: string                       # Required. "auto" = hostname or node name from env

# Telemetry collection.
collection_interval_seconds: int      # Default 5. Min 1, max 300.

# OpenTelemetry export.
otel:
  endpoint: string                    # Required. E.g. "otel-collector:4317"
  insecure: bool                      # Default false (use TLS)
  tls_cert_path: string               # Optional. Path to CA cert for server verification
  # Optional: client cert/key for mTLS
  # timeout_seconds: int              # Optional. Export timeout

# Cost engine.
cost:
  electricity_cost_per_kwh: float64   # Required if cost export enabled
  pue: float64                        # Default 1.0. Must be >= 1.0

# Process attribution (optional).
process_attribution:
  enabled: bool                       # Default false

# Optional: logging level, log path (no write except optional log file).
# Optional: resource attributes (environment, etc.)
```

### 4.3 Validation Rules

- `cluster_id`: Non-empty.
- `node_id`: Non-empty; if `"auto"`, resolve to hostname or `NODE_NAME` (K8s).
- `collection_interval_seconds`: 1 <= value <= 300.
- `otel.endpoint`: Non-empty; valid host:port.
- `cost.pue`: >= 1.0.
- `cost.electricity_cost_per_kwh`: >= 0.

Invalid config: log error and exit at startup (no fallback defaults that change semantics).

### 4.4 Config Reload

Spec: "Config reload must not restart process." Implication: if reload is supported, apply new collection interval and cost/OTel settings without full restart; no requirement to implement reload in v1 (can be "no reload" for v1).

---

## 5. OpenTelemetry Contract

### 5.1 Protocol and Endpoint

- **Protocol:** OTLP over gRPC.
- **Endpoint:** Single endpoint from config (e.g., `otel.endpoint`).
- **Inbound:** None. Agent only initiates outbound connections.

### 5.2 Resource Attributes (Common)

Attached to all metrics and traces:

| Attribute (OTel semantic) | Example / Source |
|---------------------------|------------------|
| `service.name` | `"gpu-econ-agent"` |
| `service.version` | Build version (e.g., git tag) |
| `host.name` | From `node_id` (resolved) |
| `cluster.id` | From config `cluster_id` |
| `node.id` | From config `node_id` (resolved) |
| `environment` | Optional; from config |

Custom attributes must use consistent namespaces: `gpu.*`, `infra.*`, `cost.*`.

### 5.3 Metrics

#### 5.3.1 Metric Types and Names

| Name | Type | Unit | Description |
|------|------|------|--------------|
| `gpu.power.watts` | Gauge | W | Instantaneous power per GPU |
| `gpu.utilization.percent` | Gauge | 1 | SM utilization 0–100 |
| `gpu.memory.utilization.percent` | Gauge | 1 | Memory utilization 0–100 |
| `gpu.temperature.celsius` | Gauge | C | GPU temperature |
| `gpu.power.limit.watts` | Gauge | W | Power limit per GPU |
| `gpu.energy.kwh.total` | Counter | kWh | Cumulative energy per GPU |
| `gpu.cost.total` | Counter | (currency) | Cumulative cost per GPU |
| `gpu.power.distribution` | Histogram | W | Optional; power distribution over time |

#### 5.3.2 Allowed Labels (Cardinality Control)

- **gpu.id** — Bounded (number of GPUs on node).
- **node.id** — One per node (already in resource attributes; can be repeated in metric attributes if needed).
- **cluster.id** — One per cluster (prefer resource attributes).

**Disallowed in metrics:** `job_id`, `tenant_id`, raw `PID`, raw `container_id`, timestamp, or any unbounded identifier.

#### 5.3.3 Agent Self-Observability Metrics

Exposed via same OTel pipeline:

| Name | Type | Description |
|------|------|-------------|
| `agent.cpu.usage` | Gauge | Agent process CPU usage |
| `agent.memory.usage` | Gauge | Agent process memory (e.g., RSS) |
| `agent.export.errors` | Counter | OTLP export failures |
| `agent.poll.duration` | Histogram or Gauge | Duration of one poll cycle |
| `agent.queue.size` | Gauge | Size of in-memory retry queue (if any) |

#### 5.3.4 Error Metric

| Name | Type | Description |
|------|------|-------------|
| `gpu.nvml.errors` | Counter | NVML errors (e.g., driver unavailable, device lost) |

### 5.4 Traces

#### 5.4.1 Span: Job Summary

- **Span name:** `gpu.job.summary`
- **Span kind:** INTERNAL
- **Emission:** One span per job lifecycle (when job boundary is known); no per-interval traces.

#### 5.4.2 Span Attributes (Bounded)

| Attribute | Type | Description |
|-----------|------|-------------|
| `job.id` | string | Job identifier |
| `tenant.id` | string | Optional |
| `model.name` | string | Optional |
| `gpu.count` | int64 | Number of GPUs used |
| `job.duration.seconds` | float64 | Wall-clock duration |
| `job.energy.kwh` | float64 | Total energy for job |
| `job.cost.estimated` | float64 | Estimated cost |
| `avg.gpu.utilization` | float64 | Average utilization over job |
| `avg.gpu.power` | float64 | Average power (W) |

#### 5.4.3 Sampling

- **Job summary spans:** AlwaysOn (no sampling drop for these spans).
- No trace emission for every polling interval.

### 5.5 Compliance Rules

- Use official **OpenTelemetry Go SDK**.
- Follow OTel semantic conventions where applicable; do not invent conflicting attribute names.
- Custom attributes: consistent namespaces `gpu.*`, `infra.*`, `cost.*`.
- Prefer **monotonic counters** for cumulative energy and cost.
- **Gauges** for instantaneous readings.
- No dynamic label keys; strict cardinality budget (see 5.3.2).

---

## 6. Error Handling and Edge Cases

### 6.1 NVML Unavailable

- **Scenarios:** Driver crash, NVML library missing, GPU reset.
- **Behavior:** Increment `gpu.nvml.errors`; continue retry loop; do **not** crash daemon.
- **Retry:** Backoff and re-init NVML (implementation-defined interval).

### 6.2 GPU Hot Reset During Job

- **Scenarios:** GPU power cycle; device disappears temporarily.
- **Behavior:** Close current aggregation window for that GPU; emit partial metrics; re-enumerate devices and re-open handles cleanly.

### 6.3 OTLP Collector Unreachable

- **Scenarios:** Network outage, collector restart.
- **Behavior:** In-memory retry queue with exponential backoff; if memory cap exceeded, drop oldest telemetry; **never** block the telemetry polling loop.

### 6.4 High Cardinality (Many Short-Lived Jobs)

- **Mitigation:** Disable job-level tracing via config; or apply sampling; cap concurrent job span count.

### 6.5 Clock Skew

- **Mitigation:** Prefer **monotonic clock** for energy aggregation (delta time); use **wall clock** only for span timestamps and export.

### 6.6 Process Attribution Race

- **Scenario:** Process exits between poll and mapping.
- **Behavior:** Skip attribution for that process; do not crash.

### 6.7 Shared GPU (MIG or Multi-Process)

- **Requirement:** Attribute proportional energy/cost using utilization ratio and/or memory ratio when possible.
- If accurate attribution not possible: emit shared-mode flag; mark cost as `estimated_shared`.

### 6.8 Resource Starvation

- **Scenario:** Node under heavy CPU pressure.
- **Behavior:** Skip interval if necessary; log warning; avoid aggressive spinning.

---

## 7. Security and Isolation

### 7.1 Communication Boundaries

- **Inbound:** None. No network ports, no remote commands, no IPC listeners.
- **Outbound:** Only to configured OTLP endpoint (and optional DNS for that endpoint).

### 7.2 Filesystem

- **Read-only:** `/proc`, `/dev/nvidia*`, cgroup paths (if attribution enabled); config file.
- **Write:** Optional structured logs only; no local database; no state persistence.

### 7.3 Privilege

- **Preferred:** Run as non-root with minimal capabilities.
- **If root required:** Drop unneeded capabilities; use minimal capabilities (e.g., CAP_SYS_ADMIN only if unavoidable).
- **Kubernetes:** `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false`, `runAsNonRoot: true` when possible.

### 7.4 TLS/mTLS

- Support TLS for OTLP (server verification via CA cert).
- Optional client certificate for mTLS (config-driven).

---

## 8. Performance and Stability

### 8.1 Performance Targets

| Metric | Target |
|--------|--------|
| CPU overhead | < 1% of node CPU |
| Memory usage | < 50 MB |
| Collection interval jitter | < 50 ms |
| Export latency | < 1 s (best effort) |

### 8.2 Stability and Determinism

- No dynamic plugin loading.
- No runtime code generation.
- Deterministic polling interval (fixed duration).
- All memory allocations bounded (no unbounded queues without drop policy).
- Config reload (if implemented) must not require process restart.

---

## 9. Non-Goals (v1)

- Centralized billing engine
- Multi-cluster aggregation
- Predictive cost modeling
- Energy optimization scheduler
- HTTP health endpoint (unless explicitly enabled)

---

## 10. Future Extensions (Out of Scope for v1)

- Carbon intensity integration
- Energy efficiency scoring
- Kubernetes job auto-attribution (job boundaries)
- Multi-tenant cost dashboards
- Billing export (CSV / API)

---

## 11. Threat Model Summary

- **Trusted:** Node, collector endpoint.
- **Defend against:** Misconfiguration, collector unavailability, driver instability, process churn.
- **Out of scope:** Authentication of workloads, policy enforcement, modifying GPU behavior. The agent is **strictly observational**.

---

*End of detailed specification.*

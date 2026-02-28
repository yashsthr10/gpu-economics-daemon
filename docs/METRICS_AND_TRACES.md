# GPU Economic Telemetry Agent — Metrics and Traces Reference

This document is the authoritative reference for OpenTelemetry metrics and traces emitted by the GPU Economic Telemetry Agent (implemented in Go). Use it for building dashboards, alerts, and trace queries.

---

## 1. Resource Attributes (Common)

All metrics and traces carry the following resource attributes (when configured):

| Attribute | Type | Example | Description |
|-----------|------|---------|-------------|
| `service.name` | string | `gpu-econ-agent` | Fixed service name |
| `service.version` | string | `1.0.0` | Build/version |
| `host.name` | string | `gpu-node-42` | Resolved node hostname |
| `cluster.id` | string | `research-cluster-01` | From config |
| `node.id` | string | `gpu-node-42` | From config or auto |
| `environment` | string | `production` | Optional |

---

## 2. Metrics

### 2.1 GPU Telemetry (Gauges)

Instantaneous values per GPU; sampled at `collection_interval_seconds`.

| Metric Name | Type | Unit | Description | Attributes |
|-------------|------|------|-------------|------------|
| `gpu.power.watts` | Gauge | W | Instantaneous power draw | `gpu.id` |
| `gpu.utilization.percent` | Gauge | 1 | SM utilization (0–100) | `gpu.id` |
| `gpu.memory.utilization.percent` | Gauge | 1 | Memory utilization (0–100) | `gpu.id` |
| `gpu.temperature.celsius` | Gauge | C | GPU temperature | `gpu.id` |
| `gpu.power.limit.watts` | Gauge | W | Power limit (cap) | `gpu.id` |

**Attribute:**

- `gpu.id`: GPU index or stable identifier (e.g., `0`, `1`); cardinality = number of GPUs per node.

### 2.2 Energy and Cost (Counters)

Cumulative since agent start (or since last counter reset). Monotonic.

| Metric Name | Type | Unit | Description | Attributes |
|-------------|------|------|-------------|------------|
| `gpu.energy.kwh.total` | Counter | kWh | Cumulative energy per GPU | `gpu.id` |
| `gpu.cost.total` | Counter | (currency) | Cumulative cost per GPU | `gpu.id` |

Currency is implied by configuration (e.g., USD); no currency dimension in v1.

### 2.3 Optional Histogram

| Metric Name | Type | Unit | Description | Attributes |
|-------------|------|------|-------------|------------|
| `gpu.power.distribution` | Histogram | W | Distribution of power samples over time | `gpu.id` |

Bucket boundaries are implementation-defined (e.g., 0, 50, 100, 150, 200, 250, 300, +Inf for Watts).

### 2.4 Error Metric

| Metric Name | Type | Description | Attributes |
|-------------|------|-------------|------------|
| `gpu.nvml.errors` | Counter | NVML errors (driver/device unavailable, reset) | — |

Increment once per error event; no high-cardinality labels.

### 2.5 Agent Self-Observability

| Metric Name | Type | Unit | Description |
|-------------|------|------|-------------|
| `agent.cpu.usage` | Gauge | 1 (ratio or percent) | Agent process CPU usage |
| `agent.memory.usage` | Gauge | By | Agent process memory (e.g., RSS) |
| `agent.export.errors` | Counter | 1 | OTLP export failures |
| `agent.poll.duration` | Histogram or Gauge | s | Duration of one poll cycle |
| `agent.queue.size` | Gauge | 1 | In-memory retry queue size (if applicable) |

---

## 3. Traces

### 3.1 Job Summary Span

Emitted once per job when job lifecycle is known (e.g., future K8s integration). Not emitted every polling interval.

| Field | Value |
|-------|--------|
| Span name | `gpu.job.summary` |
| Span kind | INTERNAL |
| Sampling | AlwaysOn for job summaries |

### 3.2 Span Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `job.id` | string | Job identifier |
| `tenant.id` | string | Optional tenant |
| `model.name` | string | Optional model name |
| `gpu.count` | int64 | Number of GPUs used |
| `job.duration.seconds` | float64 | Wall-clock duration (s) |
| `job.energy.kwh` | float64 | Total energy (kWh) for job |
| `job.cost.estimated` | float64 | Estimated cost (currency) |
| `avg.gpu.utilization` | float64 | Average GPU utilization (%) |
| `avg.gpu.power` | float64 | Average power (W) |

---

## 4. Cardinality and Naming Rules

- **Allowed metric/trace attributes:** `gpu.id`, `node.id`, `cluster.id` (prefer resource attributes for node/cluster).
- **Disallowed:** `job_id`, `tenant_id`, raw `PID`, raw `container_id`, timestamp, or any unbounded identifier in **metrics**.
- **Namespaces:** Custom attributes use `gpu.*`, `infra.*`, `cost.*` for consistency with OTel conventions.

---

## 5. Example Queries (Prometheus-style)

Assuming metrics are exported to Prometheus (via OTLP collector):

- **Total energy per node (sum over GPUs):**  
  `sum by (node_id, cluster_id) (gpu_energy_kwh_total)`

- **Average GPU utilization per node:**  
  `avg by (node_id) (gpu_utilization_percent)`

- **Agent export errors (alert):**  
  `increase(agent_export_errors[5m]) > 0`

- **NVML errors (alert):**  
  `increase(gpu_nvml_errors[5m]) > 0`

*(Exact metric names depend on OTLP-to-Prometheus naming; typically snake_case and suffix like `_total` for counters.)*

---

*End of metrics and traces reference.*

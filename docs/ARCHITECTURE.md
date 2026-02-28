# GPU Economic Telemetry Agent — Architecture

This document describes the system architecture of the GPU Economic Telemetry Agent: components, data flow, concurrency model, deployment, and integration points. The agent is implemented in **Go (Golang)**.

---

## 1. High-Level Architecture

### 1.1 System Context

The agent runs on each node that has one or more NVIDIA GPUs. It collects GPU telemetry, aggregates energy and cost in memory, and exports OpenTelemetry data to an OTLP collector. The collector forwards metrics and traces to backends (e.g., Prometheus, Tempo, or vendor backends).

```
                    +------------------------------------------+
                    |                  Node                     |
                    |  +--------------------------------------+ |
                    |  |       GPU Economic Telemetry Agent    | |
                    |  |  (Collector -> Aggregator -> Cost     | |
                    |  |   -> OTel Exporter)                   | |
                    |  +------------------+-------------------+ |
                    |                     | OTLP (gRPC)        | |
                    |  +------------------v-------------------+ |
                    |  |  NVML / /proc (read-only)            | |
                    |  +--------------------------------------+ |
                    +-------------------+-----------------------+
                                        |
                                        | Outbound only
                                        v
                    +------------------------------------------+
                    |  OTLP Collector (e.g., OpenTelemetry     |
                    |  Collector, Grafana Agent)               |
                    +-------------------+-----------------------+
                                        |
                    +-------------------+-----------------------+
                    |  Backends: Prometheus, Tempo, etc.       |
                    +------------------------------------------+
```

### 1.2 Logical Layers

| Layer | Responsibility |
|-------|----------------|
| **Collector** | Poll NVML and process table at fixed interval; produce raw telemetry snapshots |
| **Aggregator** | Compute energy per interval via Riemann sum (trapezoidal: (P_prev+P_current)/2 * delta_t), accumulate E_total, convert to kWh; maintain cumulative state per GPU |
| **Cost Engine** | Apply PUE to E_kWh (E_effective = E_kWh * PUE), then cost = E_effective * cost_per_kWh |
| **OTel Export** | Map internal state to OTel metrics/traces; send via OTLP gRPC |

---

## 2. Component Architecture

### 2.1 Component Diagram

```
+------------------------------------------------------------------+
|                         GPU Econ Agent Process                    |
|                                                                   |
|  +-------------+   +-------------+   +-------------+   +---------+|
|  |   Config    |   |  Collector  |   | Aggregator  |   |  Cost   ||
|  |   Loader    |-->|   (NVML +   |-->|  (windows,  |-->| Engine  ||
|  |             |   |   process)  |   |  energy)    |   | (PUE,   ||
|  +-------------+   +-------------+   +-------------+   |  rate)  ||
|         |                |                  |          +----+----+|
|         v                v                  v               |    |
|  +-------------+   +-------------+   +------------------+     |    |
|  |   Runtime   |   |  Channels  |   |  OTel Exporter   |<----+    |
|  |  (daemon,   |   |  (samples, |   |  (metrics +      |          |
|  |   shutdown) |   |   state)   |   |   traces, OTLP)  |          |
|  +-------------+   +-------------+   +--------+--------+          |
|                                               |                   |
+-----------------------------------------------+-------------------+
                                                |
                                                v
                                    +----------------------+
                                    |  OTLP Collector      |
                                    +----------------------+
```

### 2.2 Component Descriptions

#### 2.2.1 Config Loader

- **Input:** File path (CLI or env).
- **Output:** Validated configuration struct (cluster_id, node_id, intervals, OTel endpoint, cost params, process_attribution).
- **Behavior:** Load and validate once at startup; optional reload without process restart (if implemented).

#### 2.2.2 Collector

- **Input:** Config (collection_interval_seconds, process_attribution.enabled).
- **Output:** Stream of `NodeSample` (timestamp, per-GPU metrics, optional per-process).
- **Dependencies:** NVML library (CGO or cgo wrapper), read-only `/proc` (and optionally cgroups).
- **Behavior:** Loop: sleep(interval), enumerate GPUs, read power/util/memory/temp/limit per GPU, optionally map processes to GPUs, send snapshot to channel. On NVML error: increment error metric, retry, do not exit.

#### 2.2.3 Aggregator

- **Input:** Stream of `NodeSample` from collector.
- **Output:** Updated cumulative state (energy per GPU in kWh, optional rolling windows) and/or events for exporter.
- **Behavior:** For each sample, compute delta_t since last sample; for each GPU, use Riemann sum (trapezoidal): E_interval = (P_prev + P_current) / 2 * delta_t; accumulate E_total, then E_kWh = E_total / 3,600,000; maintain cumulative counters; handle GPU disappear/reappear (reset or partial emit for that GPU).

#### 2.2.4 Cost Engine

- **Input:** Aggregated E_kWh per GPU (from aggregator); config (electricity_cost_per_kwh, pue).
- **Output:** Cost per GPU: E_effective = E_kWh * pue; cost = E_effective * electricity_cost_per_kwh.
- **Behavior:** Stateless calculation; can be invoked by aggregator or exporter when producing OTel data.

#### 2.2.5 OTel Exporter

- **Input:** Current gauges (power, utilization, etc.), cumulative counters (energy, cost), optional job summary events.
- **Output:** OTLP gRPC requests (metrics and traces) to configured endpoint.
- **Behavior:** Map internal state to OTel Metrics API; optionally create spans for job summaries; batch and send; on failure, queue with backoff and drop policy when over cap.

#### 2.2.6 Runtime / Daemon

- **Responsibility:** Start all goroutines (collector, aggregator, exporter); handle signals (SIGTERM/SIGINT); propagate context cancellation for graceful shutdown; coordinate cleanup (flush exporter, close NVML).

---

## 3. Data Flow

### 3.1 Telemetry Pipeline (Metrics)

```
[NVML + /proc] --> Collector --> NodeSample --> Aggregator --> Cumulative state
                                                                      |
                                                                      v
Config (cost) --------------------------------------------------> Cost Engine
                                                                      |
                                                                      v
                                                            OTel Exporter
                                                                      |
                                                                      v
                                                            OTLP Collector
```

- **Collector** produces `NodeSample` at fixed intervals.
- **Aggregator** consumes samples, computes energy per interval via Riemann sum (trapezoidal), accumulates E_total, converts to E_kWh; exposes current state (gauges + counters) to exporter.
- **Cost Engine** reads E_kWh and config, computes E_effective = E_kWh * PUE and cost = E_effective * cost_per_kWh; can be called by Aggregator or Exporter.
- **OTel Exporter** periodically (or on timer) reads state and cost, builds OTel Metric protobuf, sends via OTLP.

### 3.2 Full Economic Pipeline (Formula)

The end-to-end calculation uses trapezoidal (Riemann sum) approximation:

1. **Per interval:** `E_interval = (P_prev + P_current) / 2 * delta_t`
2. **Accumulate:** `E_total = sum(E_interval)`
3. **Convert to kWh:** `E_kWh = E_total / 3,600,000`
4. **Apply PUE:** `E_effective = E_kWh * PUE`
5. **Cost:** `Cost = E_effective * cost_per_kWh`

This is the full economic pipeline from power samples to cost.

### 3.3 Trace Flow (Job Summary)

When job boundaries are known (e.g., future K8s integration):

- Aggregator or a dedicated **job tracker** accumulates energy/cost per job.
- On job end: emit one span `gpu.job.summary` with attributes (job.id, energy, cost, duration, avg utilization/power).
- Exporter sends trace via same OTLP connection (traces and metrics can share endpoint).

### 3.4 Sequence (Simplified)

```
  Collector        Aggregator       Cost Engine      Exporter        OTLP
      |                 |                 |              |             |
      |-- NodeSample -->|                 |              |             |
      |                 |-- energy ----->|              |             |
      |                 |<-- cost -------|              |             |
      |                 |-- state --------------------->|             |
      |                 |                 |              |-- OTLP --->|
      | (every interval)                  |              | (periodic)  |
```

---

## 4. Concurrency Model

### 4.1 Goroutines

| Goroutine | Role |
|-----------|------|
| Main | Load config, create context, start collector/aggregator/exporter, wait for shutdown signal |
| Collector | Poll loop: sleep(interval), sample, send to channel |
| Aggregator | Read from channel, update state (may use mutex or single goroutine for state) |
| Exporter | Timer or trigger: read state, build OTLP payload, send; optional retry queue processor |

### 4.2 Synchronization

- **Channels:** Bounded channel from Collector to Aggregator (e.g., cap 1 or 2); avoid blocking collector if aggregator is slow.
- **State:** Aggregator owns cumulative state; Exporter reads either under mutex or via copy/snapshot to avoid long locks.
- **Context:** Root `context.Context` cancelled on shutdown; all loops and RPCs respect it (graceful stop).

### 4.3 No Shared Mutable State Without Sync

- All shared state (e.g., cumulative energy map) protected by mutex or confined to one goroutine.
- Config: read-only after init or updated under mutex if reload is implemented.

---

## 5. Package Layout (Go)

```
cmd/
  agent/
    main.go              # Entrypoint: parse flags, load config, run daemon

internal/
  config/
    config.go            # Config struct, validation, load from YAML
  collector/
    nvml.go              # NVML wrapper: init, enumerate devices, read metrics
    process_mapper.go    # Map PIDs to GPU usage (optional)
    collector.go         # Poll loop, emit NodeSample
  aggregator/
    window.go            # Rolling/cumulative state per GPU
    aggregator.go        # Consume NodeSample, update state
  cost/
    engine.go            # Cost = f(energy, pue, electricity_cost_per_kwh)
  otel/
    metrics.go           # Build OTel metrics from state
    tracing.go           # Build job summary spans
  exporter/
    otlp.go              # OTLP gRPC client, batch, send, retry
  runtime/
    daemon.go            # Start/stop goroutines, signal handling, shutdown
```

Optional: `pkg/` for types shared with tests or future libraries (e.g., `NodeSample`, `GPUSample`).

---

## 6. Deployment Architecture

### 6.1 Bare Metal

- **Artifact:** Static binary (e.g., `gpu-econ-agent`).
- **Process manager:** systemd.
- **Config:** `/etc/gpu-econ-agent/config.yaml` (or similar).
- **Permissions:** Binary with capability to access `/dev/nvidia*` and read `/proc`; run as non-root if possible.
- **Network:** Outbound only to OTLP endpoint.

### 6.2 Kubernetes (DaemonSet)

- **Schedule:** One pod per node (DaemonSet); nodeSelector or taint/toleration for GPU nodes.
- **Container:** Single container image; entrypoint = agent binary.
- **Mounts:**
  - `/dev/nvidia*` (device access for NVML).
  - `/proc` (read-only) for process attribution.
  - Config via ConfigMap or mounted secret.
- **Env:** Optional `NODE_NAME` for `node_id: auto`.
- **Resource limits:** CPU/memory per spec (e.g., 50 Mi memory, 0.1 CPU).
- **SecurityContext:** readOnlyRootFilesystem, allowPrivilegeEscalation: false, runAsNonRoot if possible.

### 6.3 VM-Based GPU Clusters

- Same binary and config; NVML available inside VM if NVIDIA driver and NVML are present (e.g., vGPU or passthrough). No architectural change.

---

## 7. Integration Points

### 7.1 NVML

- **Interface:** C library (libnvidia-ml); Go via CGO or a thin CGO wrapper.
- **Usage:** Initialize once; enumerate devices; each poll: get power, utilization, memory, temperature, power limit; optionally per-process stats.
- **Failure:** On init or poll failure, log and increment `gpu.nvml.errors`; retry; do not crash.

### 7.2 OTLP Collector

- **Protocol:** OTLP gRPC (default port 4317).
- **Authentication:** TLS (server verification); optional mTLS (client cert).
- **Backpressure:** If collector is slow or unavailable, agent uses in-memory queue with bounded size and drop-oldest policy when full.

### 7.3 Configuration

- **Source:** YAML file.
- **No remote config:** No pull from API or feature flags; all behavior from local file (and env for overrides if desired).

---

## 8. Observability of the Agent

The agent exports its own health/performance via OTel metrics (see SPECIFICATION.md):

- `agent.cpu.usage`, `agent.memory.usage`
- `agent.export.errors`, `agent.poll.duration`, `agent.queue.size`

No HTTP health endpoint in v1 unless explicitly added. Monitoring systems can scrape the OTLP collector that receives these metrics.

---

## 9. Security Architecture

- **Trust boundary:** Node and OTLP collector are trusted; agent does not authenticate workloads.
- **Least privilege:** Read-only access to devices and proc; no inbound listeners; outbound only to OTLP.
- **Secrets:** Electricity cost and certs can be supplied via config file (e.g., mounted secret in K8s); no secrets in logs.

---

## 10. Scalability and Limits

- **Per node:** One agent process; scales with number of GPUs on the node (bounded).
- **Cardinality:** Metrics labels limited to gpu.id, node.id, cluster.id (bounded).
- **Traces:** Job summary only; no per-interval traces to avoid cardinality explosion.
- **Memory:** Bounded by queue size and in-memory state (cumulative maps per GPU); target &lt; 50 MB.

---

*End of architecture document.*

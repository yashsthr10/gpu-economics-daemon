# internal/models

Shared data types used across the GPU Economic Telemetry Agent pipeline. These types are defined in [SPECIFICATION.md section 3](../../docs/SPECIFICATION.md) and used by the collector, aggregator, and exporter.

## Types

### GPUSample

One GPU at one point in time. Produced by the collector and used for current gauges in the exporter.

| Field | Type | Description |
|-------|------|-------------|
| GPUIndex | int | 0-based GPU index. |
| PowerWatts | float64 | Instantaneous power draw (W). |
| SMUtilPct | float64 | SM utilization 0–100. |
| MemUtilPct | float64 | Memory utilization 0–100 (from used/total when available). |
| TemperatureC | float64 | GPU temperature (Celsius). |
| PowerLimitW | float64 | Configured power cap (W). |

### NodeSample

Full node snapshot at one poll. Built by the collector and sent on the channel to the aggregator.

| Field | Type | Description |
|-------|------|-------------|
| Timestamp | time.Time | Sample time (used for delta_t in aggregator). |
| GPUs | []GPUSample | Per-GPU metrics. |
| ProcessInfo | []ProcessGPUUsage | Optional; present when process_attribution is enabled. |

### ProcessGPUUsage

Links a process to GPU usage. Used when process attribution is enabled (v1: process mapper is stubbed).

| Field | Type | Description |
|-------|------|-------------|
| PID | int64 | Process identifier. |
| GPUIndex | int | GPU index. |
| MemoryBytes | uint64 | Per-process GPU memory. |
| UtilizationPct | float64 | Per-process utilization if available. |

## Usage

- **Collector** fills `GPUSample` from NVML and builds `NodeSample`; sends `NodeSample` on the channel.
- **Aggregator** consumes `NodeSample`, updates per-GPU state, and exposes current `GPUSample` and E_kWh via `Snapshot()`.
- **Exporter** uses the snapshot (gauges + E_kWh) and cost to build OTel metrics.

Pipeline types are kept minimal (no name, brand, UUID in `GPUSample`) to match the spec and limit cardinality in metrics.

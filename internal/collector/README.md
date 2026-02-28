# internal/collector

Collects GPU telemetry from NVML and optionally maps processes to GPU usage. Produces a stream of `NodeSample` values sent on a channel to the aggregator.

## Components

### nvml.go — NVML wrapper

- **NVMLReader** — Holds init state; provides `Init()`, `Shutdown()`, `ReadSnapshot()`.
- **ReadSnapshot()** — Enumerates devices via NVML, reads per-GPU: power (mW to W), power limit, temperature (GPU sensor), SM and memory utilization (with memory used/total for MemUtilPct when available). Returns `[]models.GPUSample` or an error.
- On partial read failure (e.g. one device errors), returns a partial slice when possible; if no device could be read, returns an error. Caller is expected to increment `gpu.nvml.errors` and retry on the next interval.

### process_mapper.go — Process attribution (v1 stub)

- **MapProcessesToGPUs(gpus)** — Returns `[]models.ProcessGPUUsage`. In v1 this is stubbed and always returns `nil`. A future implementation would use NVML process APIs and read-only `/proc`; processes that exit between poll and map are skipped without crashing (spec 6.6).

### collector.go — Collector loop

- **Run(ctx, intervalSeconds, processAttributionEnabled, samplesOut, onNVMLError)** — Starts the loop: initializes NVML, creates a ticker at `intervalSeconds`, and on each tick calls `ReadSnapshot()`, optionally `MapProcessesToGPUs()`, builds a `NodeSample`, and sends it on `samplesOut`. If send would block (e.g. bounded channel full), the sample is dropped. On NVML error, `onNVMLError()` is invoked and the loop continues. Exits only when `ctx` is cancelled; on exit it shuts down NVML.

## Data flow

1. Ticker fires at `collection_interval_seconds`.
2. NVML `ReadSnapshot()` yields `[]GPUSample`.
3. If process attribution enabled, `MapProcessesToGPUs(gpus)` yields `ProcessInfo` (v1: nil).
4. Build `NodeSample{Timestamp, GPUs, ProcessInfo}` and send on channel (or drop if full).
5. On error, call `onNVMLError()` and continue.

## Dependencies

- **github.com/NVIDIA/go-nvml** — NVML bindings (CGO).
- **internal/models** — GPUSample, NodeSample, ProcessGPUUsage.

## Edge cases

- NVML init or read failure: error reported to caller via return or callback; no crash.
- Channel full: sample dropped so the collector never blocks the pipeline.

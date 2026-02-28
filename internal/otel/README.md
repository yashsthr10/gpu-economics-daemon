# internal/otel

Builds OpenTelemetry metrics (and optional trace stubs) for the GPU Economic Telemetry Agent. Implements the metric names, types, and attributes defined in [SPECIFICATION.md section 5](../../docs/SPECIFICATION.md) and [METRICS_AND_TRACES.md](../../docs/METRICS_AND_TRACES.md).

## Components

### metrics.go — Metric registration

- **MetricsSnapshot** — Struct passed to callbacks: Gauges (current GPUSample per GPU), EKWhByGPU, CostByGPU, NVMLErrors, ExportErrors, QueueSize, AgentCPU, AgentMemory, PollDuration.
- **RegisterMetrics(meter, getSnapshot)** — Registers observable instruments with the meter. `getSnapshot` is a function called each collection cycle to obtain the current snapshot. Registers:
  - **Gauges (per gpu.id):** gpu.power.watts, gpu.utilization.percent, gpu.memory.utilization.percent, gpu.temperature.celsius, gpu.power.limit.watts.
  - **Counters (per gpu.id):** gpu.energy.kwh.total, gpu.cost.total.
  - **Error/agent:** gpu.nvml.errors (counter), agent.cpu.usage, agent.memory.usage, agent.export.errors, agent.poll.duration, agent.queue.size (gauges/counters as per spec).

All GPU metrics use the attribute `gpu.id` (string, GPU index) for cardinality control.

### tracing.go — Trace stub

- **JobSummarySpan()** — No-op in v1. Reserved for future job-level span emission (e.g. `gpu.job.summary`) when job boundaries are available (e.g. Kubernetes integration).

## Usage

The runtime creates a Meter from the exporter’s MeterProvider, then calls `RegisterMetrics(meter, getSnapshot)`. The OTel SDK’s PeriodicReader invokes the registered callbacks at each export interval; callbacks call `getSnapshot()` and observe the current values.

## Dependencies

- **go.opentelemetry.io/otel/metric** — Meter, observable instruments, callbacks.
- **go.opentelemetry.io/otel/attribute** — Attributes for gpu.id.
- **internal/models** — GPUSample (for gauges).

# internal

Private packages for the GPU Economic Telemetry Agent. Not intended for import by other projects.

## Package index

| Package | Purpose |
|---------|---------|
| [models](models/) | Shared data types: GPUSample, NodeSample, ProcessGPUUsage. |
| [config](config/) | YAML config load, validation, node_id resolution. |
| [collector](collector/) | NVML wrapper and collector loop; optional process mapper (stub). |
| [aggregator](aggregator/) | Per-GPU energy state (Riemann sum), snapshot for exporter. |
| [cost](cost/) | Cost calculation: E_kWh and config to cost per GPU. |
| [otel](otel/) | OTel metrics registration and tracing stub. |
| [exporter](exporter/) | OTLP gRPC MeterProvider and resource setup. |
| [runtime](runtime/) | Daemon wiring: channels, goroutines, signals, shutdown. |

## Data flow

```
config (load once)
    |
    v
collector --> [NodeSample channel] --> aggregator
    |                                      |
    | (NVML errors -> atomic)              | state.Snapshot() + cost.CostPerGPU()
    |                                      v
    +---------------------------------> exporter (MeterProvider)
                                              |
                                              v
                                         OTLP gRPC
```

See [ARCHITECTURE.md](../docs/ARCHITECTURE.md) for the full design.

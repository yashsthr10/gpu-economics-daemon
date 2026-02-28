# internal/runtime

Wires and runs the GPU Economic Telemetry Agent: configures the collector, aggregator, cost engine, OTLP MeterProvider, and metric registration; starts goroutines and handles shutdown on signal.

## Behaviour

### Run(ctx, cfg, version)

1. **Context and signals** — Creates a child context cancelled on SIGTERM/SIGINT so all components can stop gracefully.
2. **State and channel** — Allocates `aggregator.NewState()` and a bounded channel (cap 2) for `NodeSample` from collector to aggregator.
3. **NVML error counter** — Defines an atomic counter and `onNVMLError` callback; passed to the collector so it can increment the counter on each NVML read error.
4. **MeterProvider** — Calls `exporter.NewMeterProvider` with endpoint, TLS options, and resource attributes (service name, version, host.name = node_id, cluster.id, node.id). Gets back the provider and a shutdown function.
5. **Metrics registration** — Gets a Meter from the provider and builds `getSnapshot`: reads `state.Snapshot()` (gauges + E_kWh), computes `cost.CostPerGPU(eKWhByGPU, costCfg)`, and fills `otel.MetricsSnapshot` (including NVMLErrors from the atomic). Calls `otel.RegisterMetrics(meter, getSnapshot)`.
6. **Goroutines** — Starts the collector (sends on channel; closes channel on exit) and the aggregator (consumes channel, updates state). Waits for both to finish.
7. **Shutdown** — On context cancel, collector and aggregator exit; then the runtime calls the MeterProvider shutdown to flush the OTLP exporter.

The collector runs NVML init and shutdown internally; when its goroutine exits, NVML is shut down in a defer.

## Data flow

- Collector -> (channel) -> Aggregator.
- Exporter (PeriodicReader) periodically calls the registered callbacks -> callbacks call getSnapshot() -> read state + cost + atomics -> observe metrics -> SDK exports to OTLP.

## Dependencies

- internal/aggregator, internal/collector, internal/config, internal/cost, internal/exporter, internal/models, internal/otel.

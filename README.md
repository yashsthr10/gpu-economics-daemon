# GPU Economic Telemetry Agent

A node-level daemon (written in Go) that collects GPU telemetry via NVML, aggregates energy using a trapezoidal (Riemann sum) approximation, computes cost from configurable PUE and electricity rate, and exports metrics to an OpenTelemetry (OTLP) endpoint.

## Overview

- **Collector:** Polls NVIDIA GPUs at a configurable interval; reads power, utilization, temperature, and power limit; optionally attributes usage to processes.
- **Aggregator:** Maintains per-GPU state and accumulates energy (Watt-seconds) using the formula `E_interval = (P_prev + P_current) / 2 * delta_t`, then exposes E_kWh per GPU.
- **Cost engine:** Applies `E_effective = E_kWh * PUE` and `Cost = E_effective * electricity_cost_per_kwh` per GPU.
- **Exporter:** Sends OTLP metrics (gauges and cumulative counters) over gRPC to a configurable endpoint.

See [docs/README.md](docs/README.md) for the full specification and architecture.

## Requirements

- Go 1.24+ (or the version in `go.mod`)
- Linux with NVIDIA drivers and NVML (`libnvidia-ml.so`) for the collector
- CGO enabled when building the agent (required for go-nvml)

## Build

Using the Makefile (recommended):

```bash
make format   # Format code (gofmt + goimports if installed)
make build   # Build agent, extract_gpu_snapshot, and demo ingestion
make test    # Run tests
make help    # List all targets
```

Or build the agent only:

```bash
# Build the agent binary
go build -o gpu-econ-agent ./cmd/agent/

# Optional: set version for OTel resource attribute
make build-agent VERSION=1.0.0
```

## Run

Create a YAML config file (see [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for schema and examples), then:

```bash
./gpu-econ-agent -config /path/to/config.yaml
```

The agent runs until it receives SIGTERM or SIGINT; it then shuts down the collector, aggregator, and OTLP exporter gracefully.

## Configuration

Required fields:

- `cluster_id` — Identifier for the cluster.
- `node_id` — Node identifier, or `"auto"` to use hostname (or `NODE_NAME` in Kubernetes).
- `collection_interval_seconds` — Poll interval (1–300).
- `otel.endpoint` — OTLP gRPC endpoint (e.g. `localhost:4317`).
- `cost.pue` — Power usage effectiveness (>= 1.0).
- `cost.electricity_cost_per_kwh` — Cost per kWh (>= 0).

Optional: `otel.insecure`, `otel.tls_cert_path`, `process_attribution.enabled`.

## Project layout

| Path | Description |
|------|-------------|
| [cmd/agent](cmd/agent/) | Main entrypoint; flags and config load. |
| [internal/collector](internal/collector/) | NVML wrapper and collector loop. |
| [internal/aggregator](internal/aggregator/) | Per-GPU energy state and snapshot. |
| [internal/cost](internal/cost/) | Cost calculation from E_kWh and config. |
| [internal/otel](internal/otel/) | OTel metrics and tracing (stub). |
| [internal/exporter](internal/exporter/) | OTLP gRPC MeterProvider setup. |
| [internal/runtime](internal/runtime/) | Daemon wiring and signal handling. |
| [internal/config](internal/config/) | Config load, validation, node_id resolution. |
| [internal/models](internal/models/) | Shared types (GPUSample, NodeSample, etc.). |
| [scripts](scripts/) | One-shot GPU snapshot script for testing. |
| [demo](demo/) | Demo ingestion service (OTLP to JSON file). |
| [deploy](deploy/) | systemd unit and Kubernetes manifests. |
| [docs](docs/) | Specification, architecture, deployment, metrics. |

## Testing

```bash
go test ./...
```

## Deployment

- **Bare metal:** See [deploy/README.md](deploy/README.md) and [deploy/systemd](deploy/systemd/).
- **Kubernetes:** See [deploy/kubernetes](deploy/kubernetes/) and [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

## License

See repository license file.

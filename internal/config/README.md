# internal/config

Loads and validates YAML configuration for the GPU Economic Telemetry Agent. Implements [SPECIFICATION.md section 4](../../docs/SPECIFICATION.md) (configuration schema and validation).

## Behaviour

1. **Load(path)** — Reads the file at `path`, unmarshals YAML into the config struct, applies defaults, validates, and resolves `node_id` when set to `"auto"`.
2. **Validate(cfg)** — Enforces: non-empty `cluster_id` and `node_id`; `collection_interval_seconds` in [1, 300]; non-empty `otel.endpoint`; `cost.pue` >= 1.0; `cost.electricity_cost_per_kwh` >= 0.
3. **ResolveNodeID(cfg)** — If `node_id == "auto"`, sets it to hostname; if still empty, uses env `NODE_NAME` (e.g. Kubernetes); otherwise `"unknown"`.

Defaults: `collection_interval_seconds` 5 if zero; `cost.pue` 1.0 if zero.

## Config struct

- **ClusterID**, **NodeID** — Cluster and node identifiers.
- **CollectionIntervalSeconds** — Poll interval for the collector.
- **OTEL** — Endpoint, Insecure, TLSCertPath.
- **Cost** — ElectricityCostPerKWh, PUE.
- **ProcessAttribution** — Enabled (bool).

## Usage

Called from `cmd/agent/main.go` before starting the runtime. On validation failure, `Load` returns an error and the process exits.

## Files

- `config.go` — Struct definitions, `Load`, `Validate`, `ResolveNodeID`, defaults.

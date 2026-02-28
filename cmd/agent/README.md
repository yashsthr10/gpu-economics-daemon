# cmd/agent

Main entrypoint for the GPU Economic Telemetry Agent daemon.

## Behaviour

1. **Flags** — Parses `-config` (required): path to the YAML config file.
2. **Config** — Calls `config.Load(configPath)`; on error logs and exits with a non-zero code.
3. **Version** — Uses `main.Version` (set at build time via `-ldflags "-X main.Version=..."`); if unset, uses `"dev"`.
4. **Run** — Calls `runtime.Run(ctx, cfg, Version)` with a background context. The runtime runs until a signal is received; then it shuts down and returns. If `Run` returns an error, the process logs it and exits with code 1.

## Build

From the repository root:

```bash
go build -o gpu-econ-agent ./cmd/agent/
```

With version for OTel resource attribute:

```bash
go build -ldflags "-X main.Version=1.0.0" -o gpu-econ-agent ./cmd/agent/
```

## Run

```bash
./gpu-econ-agent -config /path/to/config.yaml
```

## Dependencies

- internal/config — Load and validate config.
- internal/runtime — Wire and run collector, aggregator, exporter.

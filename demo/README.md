# demo

Demo and testing utilities for the GPU Economic Telemetry Agent.

## ingestion

Dummy OTLP ingestion service that accepts OTLP gRPC (traces and metrics), converts each Export batch to JSON, and appends to a file (NDJSON). Useful for local testing without a full OpenTelemetry Collector or backend.

**See [ingestion/README.md](ingestion/README.md)** for build, run, and usage.

Quick start:

```bash
go build -o ingestion ./demo/ingestion/
./ingestion -addr :4317 -out otel_ingestion.jsonl
```

Then run the agent with `otel.endpoint: "localhost:4317"` and `otel.insecure: true`; exported metrics will be appended to `otel_ingestion.jsonl` (JSONL: one JSON object per line).

## frontend

React app to upload the ingestion NDJSON file and view GPU metrics analysis: total energy, total cost, average utilization, average power, max temperature, NVML errors, and per-GPU breakdown.

**See [frontend/README.md](frontend/README.md)** for setup and run.

Quick start:

```bash
cd demo/frontend && npm install && npm run dev
```

Then open the dev server URL and upload `otel_ingestion.jsonl`.

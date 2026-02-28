# GPU Economics — Ingestion Analysis Frontend

React frontend to upload the **JSONL** file produced by the [demo ingestion service](../ingestion/) and view aggregated GPU metrics: total energy, total cost, average utilization, average power, max temperature, NVML errors, and per-GPU breakdown.

## Setup

```bash
cd demo/frontend
npm install
```

## Run (dev)

```bash
npm run dev
```

Open the URL shown (e.g. http://localhost:5173). Upload `otel_ingestion.jsonl` (one JSON object per line).

## Build

```bash
npm run build
```

Output is in `dist/`. Serve with any static host or `npm run preview`.

## Flow

1. Run the [ingestion service](../ingestion/) and the GPU agent so that `otel_ingestion.jsonl` is populated.
2. Open this app and upload `otel_ingestion.jsonl`.
3. The app parses **JSONL** (one line = one JSON object), keeps only `type: "metrics"` entries, and aggregates OTLP metric data points to compute:
   - **Total energy (kWh)** — sum of latest `gpu.energy.kwh.total` per GPU
   - **Total cost** — sum of latest `gpu.cost.total` per GPU
   - **Avg utilization / memory util / power** — averages over all gauge data points
   - **Max temperature** — max of `gpu.temperature.celsius`
   - **NVML errors** — max of `gpu.nvml.errors` (cumulative)
4. A per-GPU table shows energy and cost per GPU ID.

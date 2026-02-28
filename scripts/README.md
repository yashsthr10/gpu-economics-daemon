# Scripts

## extract_gpu_snapshot

One-shot test script that collects GPU telemetry (same data the collector will use) and writes a single snapshot to a JSON file for analysis.

### Requirements

- Go 1.21+
- Linux with NVIDIA drivers and `libnvidia-ml.so` (NVML) available at runtime
- Physical or virtual GPU(s) on the machine

### Build and run

```bash
# From repo root
go run ./scripts/extract_gpu_snapshot

# Or build then run
go build -o extract_gpu_snapshot ./scripts/extract_gpu_snapshot
./extract_gpu_snapshot
```

### Options

- `-out <path>` — Output JSON file (default: `gpu_snapshot.json`)

### Output

The script writes a JSON file with:

**Node-level:**
- `timestamp` — UTC time (RFC3339)
- `hostname` — Hostname (if available)
- `driver_version` — NVIDIA driver version
- `nvml_version` — NVML library version
- `gpus` — Array of per-GPU samples

**Per-GPU:**
- `gpu_index`, `name`, `brand`, `uuid`, `serial`
- `power_watts`, `power_limit_watts`, `power_limit_min_watts`, `power_limit_max_watts`, `enforced_power_limit_watts`
- `sm_utilization_percent`, `memory_utilization_percent`, `temperature_celsius`
- `memory_total_bytes`, `memory_used_bytes`
- `pcie_link_gen_width` (e.g. "Gen4 x16"), `compute_capability` (e.g. "8.9")

This extends the collector data model (GPUSample / NodeSample) with extra details for analysis.

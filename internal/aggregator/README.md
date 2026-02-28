# internal/aggregator

Maintains per-GPU energy state using a trapezoidal (Riemann sum) approximation and exposes a snapshot of current gauges and cumulative E_kWh for the exporter.

## Components

### window.go — State and snapshot

- **State** — Holds a map of per-GPU state (`gpuState`: P_prev, E_total in Watt-seconds, last timestamp, current GPUSample). Protected by a mutex; updated only by the aggregator goroutine; snapshot is a short read under lock.
- **NewState()** — Returns an empty State.
- **Update(sample)** — For each GPU in the sample: computes `delta_t` from last timestamp, then `E_interval = (P_prev + P_current) / 2 * delta_t`, adds to E_total, and stores P_current and timestamp. GPUs not present in the sample are removed from state (GPU disappear, spec 6.2).
- **Snapshot()** — Returns a copy of current gauges (one GPUSample per GPU) and a map of GPU index to E_kWh (E_total / 3_600_000).

Constants: `joulesPerKWh = 3_600_000` (Watt-seconds per kWh).

### aggregator.go — Consumer loop

- **Run(ctx, state, samplesIn)** — Reads `NodeSample` from the channel and calls `state.Update(sample)` for each. Exits when `ctx` is cancelled or the channel is closed.

## Formula (Riemann sum)

Per [SPECIFICATION.md section 2.2](../../docs/SPECIFICATION.md):

- For each interval: `E_interval = (P_prev + P_current) / 2 * delta_t` (Watt-seconds).
- Accumulate: `E_total += E_interval` per GPU.
- For export: `E_kWh = E_total / 3_600_000` per GPU.

## Data flow

1. Aggregator goroutine receives `NodeSample` from channel.
2. `Update(sample)` updates per-GPU state (and prunes missing GPUs).
3. Exporter (or runtime) calls `Snapshot()` at export time to get current gauges and E_kWh by GPU.

## Thread safety

- Only the aggregator goroutine calls `Update`.
- `Snapshot()` is safe to call from other goroutines (short read lock).

## Dependencies

- **internal/models** — NodeSample, GPUSample.

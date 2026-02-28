# internal/cost

Stateless cost engine: given cumulative E_kWh per GPU and cost configuration (PUE, electricity rate), returns cost per GPU. Implements [SPECIFICATION.md section 2.3](../../docs/SPECIFICATION.md).

## Formula

- **E_effective** = E_kWh * PUE (effective energy after PUE).
- **Cost** = E_effective * electricity_cost_per_kwh (per GPU).

All values are float64; no persistent state.

## API

### CostConfig

- **PUE** — Power usage effectiveness (>= 1.0).
- **ElectricityCostPerKWh** — Cost per kWh (>= 0).

### CostPerGPU(eKWhByGPU, cfg) -> map[int]float64

- **eKWhByGPU** — Map of GPU index to cumulative E_kWh (from aggregator snapshot).
- **cfg** — CostConfig.
- **Returns** — Map of GPU index to cumulative cost (same keys as input).

## Usage

Called by the runtime when building the metrics snapshot: after reading aggregator state (gauges + E_kWh), the runtime calls `CostPerGPU(eKWhByGPU, costCfg)` and passes the result to the OTel metrics callbacks.

## Files

- `engine.go` — CostConfig type and CostPerGPU function.

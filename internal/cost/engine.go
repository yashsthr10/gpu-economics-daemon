// Package cost implements the cost engine: E_kWh and config (PUE, electricity rate) to cost per GPU.
// Per SPECIFICATION.md section 2.3: E_effective = E_kWh * PUE, Cost = E_effective * cost_per_kWh.
package cost

// CostConfig holds the parameters needed for cost calculation.
type CostConfig struct {
	PUE                   float64
	ElectricityCostPerKWh float64
}

// CostPerGPU returns cost per GPU index (same keys as eKWhByGPU). Stateless.
// Formula: E_effective = E_kWh * PUE; Cost = E_effective * electricity_cost_per_kWh.
func CostPerGPU(eKWhByGPU map[int]float64, cfg CostConfig) map[int]float64 {
	out := make(map[int]float64, len(eKWhByGPU))
	for id, eKWh := range eKWhByGPU {
		eEffective := eKWh * cfg.PUE
		out[id] = eEffective * cfg.ElectricityCostPerKWh
	}
	return out
}

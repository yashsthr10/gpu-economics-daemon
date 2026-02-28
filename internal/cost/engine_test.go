// Package cost tests the cost engine formula: E_effective = E_kWh * PUE, Cost = E_effective * rate.
package cost

import (
	"testing"
)

func TestCostPerGPU(t *testing.T) {
	cfg := CostConfig{PUE: 2.0, ElectricityCostPerKWh: 0.15}
	eKWh := map[int]float64{0: 10.0, 1: 5.0}
	got := CostPerGPU(eKWh, cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 GPUs, got %d", len(got))
	}
	// GPU 0: 10 * 2 * 0.15 = 3.0
	if got[0] != 3.0 {
		t.Errorf("CostPerGPU()[0] = %f, want 3.0", got[0])
	}
	// GPU 1: 5 * 2 * 0.15 = 1.5
	if got[1] != 1.5 {
		t.Errorf("CostPerGPU()[1] = %f, want 1.5", got[1])
	}
}

func TestCostPerGPU_empty(t *testing.T) {
	cfg := CostConfig{PUE: 1.0, ElectricityCostPerKWh: 0.1}
	got := CostPerGPU(nil, cfg)
	if got == nil || len(got) != 0 {
		t.Errorf("CostPerGPU(nil) should return empty map, got %v", got)
	}
	got = CostPerGPU(map[int]float64{}, cfg)
	if len(got) != 0 {
		t.Errorf("CostPerGPU(empty) should return empty map, got %v", got)
	}
}

func TestCostPerGPU_zeroRate(t *testing.T) {
	cfg := CostConfig{PUE: 1.5, ElectricityCostPerKWh: 0}
	got := CostPerGPU(map[int]float64{0: 100}, cfg)
	if got[0] != 0 {
		t.Errorf("CostPerGPU with zero rate should be 0, got %f", got[0])
	}
}

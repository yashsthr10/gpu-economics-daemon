// Package aggregator tests the Riemann sum energy accumulation and E_kWh snapshot.
package aggregator

import (
	"testing"
	"time"

	"github.com/mavvrik/gpu-economics-agent/internal/models"
)

func TestState_Update_and_Snapshot(t *testing.T) {
	state := NewState()

	t0 := time.Now()
	t1 := t0.Add(2 * time.Second) // delta_t = 2s
	t2 := t1.Add(2 * time.Second) // delta_t = 2s

	// First sample: GPU 0 at 100W. No previous -> no E_interval.
	state.Update(models.NodeSample{
		Timestamp: t0,
		GPUs:      []models.GPUSample{{GPUIndex: 0, PowerWatts: 100, PowerLimitW: 200}},
	})

	// Second sample: GPU 0 at 150W. E_interval = (100+150)/2 * 2 = 250 J.
	state.Update(models.NodeSample{
		Timestamp: t1,
		GPUs:      []models.GPUSample{{GPUIndex: 0, PowerWatts: 150, PowerLimitW: 200}},
	})

	// Third sample: GPU 0 at 50W. E_interval = (150+50)/2 * 2 = 200 J. E_total = 250 + 200 = 450 J.
	state.Update(models.NodeSample{
		Timestamp: t2,
		GPUs:      []models.GPUSample{{GPUIndex: 0, PowerWatts: 50, PowerLimitW: 200}},
	})

	gauges, eKWhByGPU := state.Snapshot()
	if len(gauges) != 1 || gauges[0].GPUIndex != 0 {
		t.Fatalf("expected one GPU in snapshot, got %v", gauges)
	}
	if gauges[0].PowerWatts != 50 || gauges[0].PowerLimitW != 200 {
		t.Errorf("gauges[0] = %+v", gauges[0])
	}
	// E_total = 450 J -> E_kWh = 450 / 3_600_000 = 0.000125
	expectedKWh := 450.0 / 3_600_000
	if eKWhByGPU[0] < expectedKWh*0.999 || eKWhByGPU[0] > expectedKWh*1.001 {
		t.Errorf("eKWhByGPU[0] = %f, want ~%f", eKWhByGPU[0], expectedKWh)
	}
}

func TestState_Update_GPU_disappear(t *testing.T) {
	state := NewState()
	now := time.Now()
	state.Update(models.NodeSample{
		Timestamp: now,
		GPUs: []models.GPUSample{
			{GPUIndex: 0, PowerWatts: 100},
			{GPUIndex: 1, PowerWatts: 80},
		},
	})
	gauges, eKWh := state.Snapshot()
	if len(gauges) != 2 {
		t.Fatalf("expected 2 GPUs, got %d", len(gauges))
	}

	// Next sample only has GPU 0 -> GPU 1 should be removed from state.
	state.Update(models.NodeSample{
		Timestamp: now.Add(time.Second),
		GPUs:      []models.GPUSample{{GPUIndex: 0, PowerWatts: 90}},
	})
	gauges, eKWh = state.Snapshot()
	if len(gauges) != 1 || gauges[0].GPUIndex != 0 {
		t.Errorf("expected one GPU after disappear, got %v", gauges)
	}
	if _, ok := eKWh[1]; ok {
		t.Error("eKWh should not contain GPU 1 after disappear")
	}
}

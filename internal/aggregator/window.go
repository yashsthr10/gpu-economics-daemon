// Package aggregator maintains per-GPU energy state and exposes a snapshot for the exporter.
package aggregator

import (
	"sync"
	"time"

	"github.com/mavvrik/gpu-economics-agent/internal/models"
)

const joulesPerKWh = 3_600_000 // Watt-seconds per kWh

// gpuState holds per-GPU aggregation state (Riemann sum: P_prev, E_total, last time).
// EIdleTotal and EActiveTotal split energy by utilization threshold for economic insight.
type gpuState struct {
	PPrev       float64
	ETotal      float64 // Watt-seconds (Joules), sum of idle + active
	EIdleTotal  float64 // Watt-seconds when util < threshold
	EActiveTotal float64 // Watt-seconds when util >= threshold
	LastAt      time.Time
	Current     models.GPUSample // Latest gauges for export
}

// State holds all GPU state and the latest full snapshot for gauge export.
// IdleThresholdPct: utilization below this is classified as idle (default 10).
type State struct {
	mu               sync.RWMutex
	byGPU            map[int]*gpuState
	latest           models.NodeSample
	IdleThresholdPct float64 // 1-100; util < this -> idle energy
}

// NewState returns an empty State with the given idle utilization threshold (1-100).
// If threshold is 0, defaults to 10.
func NewState(idleThresholdPct float64) *State {
	if idleThresholdPct <= 0 || idleThresholdPct > 100 {
		idleThresholdPct = 10
	}
	return &State{byGPU: make(map[int]*gpuState), IdleThresholdPct: idleThresholdPct}
}

// Update applies one NodeSample: computes E_interval = (P_prev + P_current)/2 * delta_t per GPU,
// adds to E_total, and updates P_prev and last timestamp. Uses monotonic time for delta_t when available.
// GPUs not in this sample are removed from state (GPU disappear, spec 6.2).
func (s *State) Update(sample models.NodeSample) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove state for GPUs no longer present
	seen := make(map[int]bool)
	for _, g := range sample.GPUs {
		seen[g.GPUIndex] = true
	}
	for id := range s.byGPU {
		if !seen[id] {
			delete(s.byGPU, id)
		}
	}

	for _, g := range sample.GPUs {
		gs, ok := s.byGPU[g.GPUIndex]
		if !ok {
			gs = &gpuState{}
			s.byGPU[g.GPUIndex] = gs
		}
		deltaT := 0.0
		if !gs.LastAt.IsZero() {
			deltaT = sample.Timestamp.Sub(gs.LastAt).Seconds()
		}
		if deltaT > 0 {
			eInterval := (gs.PPrev + g.PowerWatts) / 2.0 * deltaT
			gs.ETotal += eInterval
			if g.SMUtilPct < s.IdleThresholdPct {
				gs.EIdleTotal += eInterval
			} else {
				gs.EActiveTotal += eInterval
			}
		}
		gs.PPrev = g.PowerWatts
		gs.LastAt = sample.Timestamp
		gs.Current = g
	}
	s.latest = sample
}

// Snapshot returns a copy of the state suitable for the exporter: current gauges per GPU,
// total E_kWh per GPU, idle E_kWh per GPU, and active E_kWh per GPU.
func (s *State) Snapshot() (gauges []models.GPUSample, eKWhByGPU, eKWhIdleByGPU, eKWhActiveByGPU map[int]float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	gauges = make([]models.GPUSample, 0, len(s.byGPU))
	eKWhByGPU = make(map[int]float64)
	eKWhIdleByGPU = make(map[int]float64)
	eKWhActiveByGPU = make(map[int]float64)
	for id, gs := range s.byGPU {
		gauges = append(gauges, gs.Current)
		eKWhByGPU[id] = gs.ETotal / joulesPerKWh
		eKWhIdleByGPU[id] = gs.EIdleTotal / joulesPerKWh
		eKWhActiveByGPU[id] = gs.EActiveTotal / joulesPerKWh
	}
	return gauges, eKWhByGPU, eKWhIdleByGPU, eKWhActiveByGPU
}

// Package aggregator maintains per-GPU energy state and exposes a snapshot for the exporter.
package aggregator

import (
	"sync"
	"time"

	"github.com/mavvrik/gpu-economics-agent/internal/models"
)

const joulesPerKWh = 3_600_000 // Watt-seconds per kWh

// gpuState holds per-GPU aggregation state (Riemann sum: P_prev, E_total, last time).
type gpuState struct {
	PPrev   float64
	ETotal  float64 // Watt-seconds (Joules)
	LastAt  time.Time
	Current models.GPUSample // Latest gauges for export
}

// State holds all GPU state and the latest full snapshot for gauge export.
type State struct {
	mu     sync.RWMutex
	byGPU  map[int]*gpuState
	latest models.NodeSample // Latest NodeSample for current gauges
}

// NewState returns an empty State.
func NewState() *State {
	return &State{byGPU: make(map[int]*gpuState)}
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
		}
		gs.PPrev = g.PowerWatts
		gs.LastAt = sample.Timestamp
		gs.Current = g
	}
	s.latest = sample
}

// Snapshot returns a copy of the state suitable for the exporter: current gauges per GPU and E_kWh per GPU.
func (s *State) Snapshot() (gauges []models.GPUSample, eKWhByGPU map[int]float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	gauges = make([]models.GPUSample, 0, len(s.byGPU))
	eKWhByGPU = make(map[int]float64)
	for id, gs := range s.byGPU {
		gauges = append(gauges, gs.Current)
		eKWhByGPU[id] = gs.ETotal / joulesPerKWh
	}
	return gauges, eKWhByGPU
}

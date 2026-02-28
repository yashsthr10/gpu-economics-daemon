// Package validator evaluates power and NVML readings for sanity checks and anomaly flags.
// Used to expose gpu.anomaly.* gauges for observability.
package validator

import (
	"github.com/mavvrik/gpu-economics-agent/internal/models"
)

// DiscreteGPUPowerMinWatts is the threshold below which power is considered suspicious for a discrete GPU.
const DiscreteGPUPowerMinWatts = 5.0

// DiscreteGPUPowerLimitMinWatts: GPUs with power limit max >= this are treated as discrete (e.g. desktop/datacenter).
const DiscreteGPUPowerLimitMinWatts = 50.0

// AnomalyFlags holds per-GPU anomaly gauge values (0 or 1).
type AnomalyFlags struct {
	// SuspiciousLowPower: 1 if power < 5W and GPU appears to be discrete (PowerLimitMaxW >= 50).
	SuspiciousLowPower map[int]float64
	// PowerLimitZero: 1 if power_limit was reported as 0 by NVML (possible NVML anomaly).
	PowerLimitZero map[int]float64
}

// Eval computes anomaly flags from the current GPU gauges.
// Call with the same gauges passed to the exporter so flags align with the current snapshot.
func Eval(gauges []models.GPUSample) AnomalyFlags {
	suspicious := make(map[int]float64)
	powerLimitZero := make(map[int]float64)
	for _, g := range gauges {
		id := g.GPUIndex
		if g.PowerLimitW == 0 {
			powerLimitZero[id] = 1
		} else {
			powerLimitZero[id] = 0
		}
		discrete := g.PowerLimitMaxW >= DiscreteGPUPowerLimitMinWatts
		if discrete && g.PowerWatts < DiscreteGPUPowerMinWatts {
			suspicious[id] = 1
		} else {
			suspicious[id] = 0
		}
	}
	return AnomalyFlags{SuspiciousLowPower: suspicious, PowerLimitZero: powerLimitZero}
}

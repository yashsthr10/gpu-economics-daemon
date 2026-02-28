// Package collector implements telemetry collection from NVML and optional process attribution.
package collector

import (
	"github.com/mavvrik/gpu-economics-agent/internal/models"
)

// MapProcessesToGPUs returns per-process GPU usage when process_attribution is enabled.
// v1: stubbed to return nil (no process attribution). Can be implemented later using
// NVML compute running processes and read-only /proc; on process exit between poll
// and map, skip that process and do not crash (spec section 6.6).
func MapProcessesToGPUs(_ []models.GPUSample) []models.ProcessGPUUsage {
	return nil
}

// Package collector implements telemetry collection from NVML and optional process attribution.
package collector

import (
	"context"
	"log"
	"time"

	"github.com/mavvrik/gpu-economics-agent/internal/models"
)

// Run starts the collector loop: at each interval it reads a snapshot from NVML,
// optionally maps processes, and sends a NodeSample on the given channel.
// It exits when ctx is cancelled. samplesOut must be a send-only channel (e.g. bounded cap 1 or 2).
// On NVML read error it increments the provided onNVMLError callback and continues; it never exits except on context cancel.
func Run(ctx context.Context, intervalSeconds int, processAttributionEnabled bool, samplesOut chan<- models.NodeSample, onNVMLError func()) {
	if intervalSeconds < 1 {
		intervalSeconds = 5
	}
	interval := time.Duration(intervalSeconds) * time.Second
	reader := &NVMLReader{}
	if err := reader.Init(); err != nil {
		log.Printf("collector: NVML init failed: %v", err)
		return
	}
	defer func() {
		if err := reader.Shutdown(); err != nil {
			log.Printf("collector: NVML shutdown: %v", err)
		}
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gpus, err := reader.ReadSnapshot()
			if err != nil {
				if onNVMLError != nil {
					onNVMLError()
				}
				continue
			}
			var processInfo []models.ProcessGPUUsage
			if processAttributionEnabled {
				processInfo = MapProcessesToGPUs(gpus)
			}
			sample := models.NodeSample{
				Timestamp:   time.Now(),
				GPUs:        gpus,
				ProcessInfo: processInfo,
			}
			select {
			case samplesOut <- sample:
			case <-ctx.Done():
				return
			default:
				// Channel full; drop sample to avoid blocking (spec sync 3.2)
			}
		}
	}
}

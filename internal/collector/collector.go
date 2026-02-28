// Package collector implements telemetry collection from NVML and optional process attribution.
package collector

import (
	"context"
	"log"
	"time"

	"github.com/mavvrik/gpu-economics-agent/internal/models"
)

// OnSuccess is called after a successful NVML read. pollDurationSec and nvmlReadDurationSec are in seconds;
// jitterSec is |actual_interval - expected_interval| for integration observability.
type OnSuccess func(pollDurationSec, nvmlReadDurationSec, jitterSec float64)

// Run starts the collector loop: at each interval it reads a snapshot from NVML,
// optionally maps processes, and sends a NodeSample on the given channel.
// It exits when ctx is cancelled. samplesOut must be a send-only channel (e.g. bounded cap 1 or 2).
// On NVML read error it increments the provided onNVMLError callback and continues; it never exits except on context cancel.
// If onSuccess is non-nil, it is called after each successful read with poll duration, NVML read duration, and jitter (seconds).
func Run(ctx context.Context, intervalSeconds int, processAttributionEnabled bool, samplesOut chan<- models.NodeSample, onNVMLError func(), onSuccess OnSuccess) {
	if intervalSeconds < 1 {
		intervalSeconds = 5
	}
	interval := time.Duration(intervalSeconds) * time.Second
	intervalSec := interval.Seconds()
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

	var lastPollTime time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case tickAt := <-ticker.C:
			t0 := time.Now()
			gpus, err := reader.ReadSnapshot()
			nvmlDur := time.Since(t0).Seconds()
			if err != nil {
				if onNVMLError != nil {
					onNVMLError()
				}
				continue
			}
			pollDur := time.Since(tickAt).Seconds()
			var jitter float64
			if !lastPollTime.IsZero() {
				actualInterval := tickAt.Sub(lastPollTime).Seconds()
				jitter = actualInterval - intervalSec
				if jitter < 0 {
					jitter = -jitter
				}
			}
			lastPollTime = tickAt
			if onSuccess != nil {
				onSuccess(pollDur, nvmlDur, jitter)
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

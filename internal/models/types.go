// Package models defines shared data types for the GPU Economic Telemetry Agent
// pipeline: collector output, aggregator input, and exporter gauges.
package models

import "time"

// GPUSample represents one GPU at one point in time (per SPECIFICATION.md section 3.1).
// Used as collector output and for current gauges in the exporter.
// Extended fields (Name, Brand, UUID, etc.) match the extract_gpu_snapshot script for parity.
type GPUSample struct {
	GPUIndex     int     // 0-based index
	PowerWatts   float64 // Instantaneous power draw (W)
	SMUtilPct    float64 // SM utilization 0-100
	MemUtilPct   float64 // Memory utilization 0-100 (used/total)
	TemperatureC float64 // GPU temperature (Celsius)
	PowerLimitW  float64 // Configured power cap (W)

	// Identity and model (reported in telemetry when available)
	Name                 string  // GPU model name, e.g. "NVIDIA GeForce RTX 3050 6GB Laptop GPU"
	Brand                string  // Brand label, e.g. "Titan", "GeForce"
	UUID                 string  // Device UUID
	Serial               string  // Device serial (if available)
	PowerLimitMinW       float64 // Min power cap (W)
	PowerLimitMaxW       float64 // Max power cap (W)
	EnforcedPowerLimitW  float64 // Enforced power limit (W)
	MemoryTotalBytes     uint64  // Total GPU memory (bytes)
	MemoryUsedBytes      uint64  // Used GPU memory (bytes)
	PCIeLinkGenWidth     string  // e.g. "Gen4 x8"
	ComputeCapability    string  // e.g. "8.6"
}

// NodeSample is the full snapshot for the node at one poll (per SPECIFICATION.md section 3.1).
// Produced by the collector and consumed by the aggregator.
type NodeSample struct {
	Timestamp   time.Time         // Monotonic or wall clock
	GPUs        []GPUSample       // Per-GPU metrics
	ProcessInfo []ProcessGPUUsage // Optional; when process_attribution enabled
}

// ProcessGPUUsage links a process to GPU usage (per SPECIFICATION.md section 3.3).
// Used when process_attribution is enabled.
type ProcessGPUUsage struct {
	PID            int64   // Process identifier
	GPUIndex       int     // GPU index
	MemoryBytes    uint64  // Per-process GPU memory
	UtilizationPct float64 // Per-process utilization if available
}

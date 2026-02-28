// Package otel builds OpenTelemetry metrics (and optional traces) for the GPU Economic Telemetry Agent.
package otel

// JobSummarySpan is a stub for future job-level trace emission (SPECIFICATION.md section 5.4).
// v1: job boundaries are out of scope; no spans are emitted.
// When job lifecycle is available (e.g. K8s integration), emit one span "gpu.job.summary"
// with attributes: job.id, tenant.id, model.name, gpu.count, job.duration.seconds,
// job.energy.kwh, job.cost.estimated, avg.gpu.utilization, avg.gpu.power.
func JobSummarySpan() {
	// No-op for v1.
}

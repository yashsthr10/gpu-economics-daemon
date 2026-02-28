// Package otel builds OpenTelemetry metrics (and optional traces) for the GPU Economic Telemetry Agent.
package otel

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mavvrik/gpu-economics-agent/internal/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsSnapshot is the data passed to metric callbacks (gauges, counters) per collection cycle.
type MetricsSnapshot struct {
	Gauges               []models.GPUSample
	EKWhByGPU            map[int]float64
	EKWhEffectiveByGPU   map[int]float64 // E_kWh * PUE for transparency
	EKWhIdleByGPU        map[int]float64
	EKWhActiveByGPU      map[int]float64
	CostByGPU            map[int]float64
	AnomalySuspiciousLow map[int]float64 // 1 if power suspiciously low for discrete GPU
	AnomalyPowerLimitZero map[int]float64 // 1 if power_limit == 0 (NVML anomaly)
	RestartCount         uint64 // agent.restart.count: 1 per process start so backends can detect resets
	NVMLErrors           uint64
	ExportErrors         uint64
	QueueSize            int
	AgentCPU             float64
	AgentMemory          float64
	PollDuration         float64
	ExportDurationSec    float64 // last export flush duration (seconds)
	IntegrationJitterSec float64 // max |actual_interval - expected_interval| (seconds)
	NVMLReadLatencySec   float64 // last NVML ReadSnapshot duration (seconds)
	NVMLFailureRate      float64 // failures / (failures + successes), 0-1
}

// RegisterMetrics registers GPU and agent observable instruments with the meter.
// Callbacks read from getSnapshot each collection cycle (SPECIFICATION.md section 5.3).
func RegisterMetrics(meter metric.Meter, getSnapshot func() MetricsSnapshot) error {
	// Gauges: power, utilization, memory util, temperature, power limit (per gpu.id)
	_, err := meter.Float64ObservableGauge("gpu.power.watts", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for _, g := range s.Gauges {
			o.Observe(g.PowerWatts, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(g.GPUIndex))))
		}
		return nil
	}), metric.WithUnit("W"))
	if err != nil {
		return fmt.Errorf("gpu.power.watts: %w", err)
	}
	_, err = meter.Float64ObservableGauge("gpu.utilization.percent", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for _, g := range s.Gauges {
			o.Observe(g.SMUtilPct, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(g.GPUIndex))))
		}
		return nil
	}), metric.WithUnit("1"))
	if err != nil {
		return fmt.Errorf("gpu.utilization.percent: %w", err)
	}
	_, err = meter.Float64ObservableGauge("gpu.memory.utilization.percent", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for _, g := range s.Gauges {
			o.Observe(g.MemUtilPct, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(g.GPUIndex))))
		}
		return nil
	}), metric.WithUnit("1"))
	if err != nil {
		return fmt.Errorf("gpu.memory.utilization.percent: %w", err)
	}
	_, err = meter.Float64ObservableGauge("gpu.temperature.celsius", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for _, g := range s.Gauges {
			o.Observe(g.TemperatureC, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(g.GPUIndex))))
		}
		return nil
	}), metric.WithUnit("C"))
	if err != nil {
		return fmt.Errorf("gpu.temperature.celsius: %w", err)
	}
	_, err = meter.Float64ObservableGauge("gpu.power.limit.watts", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for _, g := range s.Gauges {
			o.Observe(g.PowerLimitW, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(g.GPUIndex))))
		}
		return nil
	}), metric.WithUnit("W"))
	if err != nil {
		return fmt.Errorf("gpu.power.limit.watts: %w", err)
	}

	// Anomaly gauges (0 or 1 per gpu.id): sanity checks on power and NVML readings
	_, err = meter.Float64ObservableGauge("gpu.anomaly.suspicious_low_power", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for id, v := range s.AnomalySuspiciousLow {
			o.Observe(v, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(id))))
		}
		return nil
	}))
	if err != nil {
		return fmt.Errorf("gpu.anomaly.suspicious_low_power: %w", err)
	}
	_, err = meter.Float64ObservableGauge("gpu.anomaly.power_limit_zero", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for id, v := range s.AnomalyPowerLimitZero {
			o.Observe(v, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(id))))
		}
		return nil
	}))
	if err != nil {
		return fmt.Errorf("gpu.anomaly.power_limit_zero: %w", err)
	}

	// Counters: energy and cost (cumulative, per gpu.id)
	_, err = meter.Float64ObservableCounter("gpu.energy.kwh.total", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for id, v := range s.EKWhByGPU {
			o.Observe(v, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(id))))
		}
		return nil
	}), metric.WithUnit("kWh"))
	if err != nil {
		return fmt.Errorf("gpu.energy.kwh.total: %w", err)
	}
	_, err = meter.Float64ObservableCounter("gpu.energy.kwh.effective", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for id, v := range s.EKWhEffectiveByGPU {
			o.Observe(v, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(id))))
		}
		return nil
	}), metric.WithUnit("kWh"))
	if err != nil {
		return fmt.Errorf("gpu.energy.kwh.effective: %w", err)
	}
	_, err = meter.Float64ObservableCounter("gpu.energy.kwh.idle", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for id, v := range s.EKWhIdleByGPU {
			o.Observe(v, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(id))))
		}
		return nil
	}), metric.WithUnit("kWh"))
	if err != nil {
		return fmt.Errorf("gpu.energy.kwh.idle: %w", err)
	}
	_, err = meter.Float64ObservableCounter("gpu.energy.kwh.active", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for id, v := range s.EKWhActiveByGPU {
			o.Observe(v, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(id))))
		}
		return nil
	}), metric.WithUnit("kWh"))
	if err != nil {
		return fmt.Errorf("gpu.energy.kwh.active: %w", err)
	}
	_, err = meter.Float64ObservableCounter("gpu.cost.total", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		for id, v := range s.CostByGPU {
			o.Observe(v, metric.WithAttributes(attribute.String("gpu.id", strconv.Itoa(id))))
		}
		return nil
	}))
	if err != nil {
		return fmt.Errorf("gpu.cost.total: %w", err)
	}

	// Agent restart counter: backends can detect counter resets when this increments
	_, err = meter.Float64ObservableCounter("agent.restart.count", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(float64(s.RestartCount))
		return nil
	}))
	if err != nil {
		return fmt.Errorf("agent.restart.count: %w", err)
	}

	// Error and agent metrics (no gpu.id)
	_, err = meter.Float64ObservableCounter("gpu.nvml.errors", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(float64(s.NVMLErrors))
		return nil
	}))
	if err != nil {
		return fmt.Errorf("gpu.nvml.errors: %w", err)
	}
	_, err = meter.Float64ObservableGauge("agent.cpu.usage", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(s.AgentCPU)
		return nil
	}), metric.WithUnit("1"))
	if err != nil {
		return fmt.Errorf("agent.cpu.usage: %w", err)
	}
	_, err = meter.Float64ObservableGauge("agent.memory.usage", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(s.AgentMemory)
		return nil
	}), metric.WithUnit("By"))
	if err != nil {
		return fmt.Errorf("agent.memory.usage: %w", err)
	}
	_, err = meter.Float64ObservableCounter("agent.export.errors", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(float64(s.ExportErrors))
		return nil
	}))
	if err != nil {
		return fmt.Errorf("agent.export.errors: %w", err)
	}
	_, err = meter.Float64ObservableGauge("agent.poll.duration", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(s.PollDuration)
		return nil
	}), metric.WithUnit("s"))
	if err != nil {
		return fmt.Errorf("agent.poll.duration: %w", err)
	}
	_, err = meter.Float64ObservableGauge("agent.export.duration", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(s.ExportDurationSec)
		return nil
	}), metric.WithUnit("s"))
	if err != nil {
		return fmt.Errorf("agent.export.duration: %w", err)
	}
	_, err = meter.Float64ObservableGauge("agent.integration.jitter", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(s.IntegrationJitterSec)
		return nil
	}), metric.WithUnit("s"))
	if err != nil {
		return fmt.Errorf("agent.integration.jitter: %w", err)
	}
	_, err = meter.Float64ObservableGauge("nvml.read.latency", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(s.NVMLReadLatencySec)
		return nil
	}), metric.WithUnit("s"))
	if err != nil {
		return fmt.Errorf("nvml.read.latency: %w", err)
	}
	_, err = meter.Float64ObservableGauge("nvml.failure.rate", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(s.NVMLFailureRate)
		return nil
	}))
	if err != nil {
		return fmt.Errorf("nvml.failure.rate: %w", err)
	}
	_, err = meter.Float64ObservableGauge("agent.queue.size", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		s := getSnapshot()
		o.Observe(float64(s.QueueSize))
		return nil
	}))
	if err != nil {
		return fmt.Errorf("agent.queue.size: %w", err)
	}
	return nil
}

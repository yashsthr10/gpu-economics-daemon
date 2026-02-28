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
	Gauges       []models.GPUSample
	EKWhByGPU    map[int]float64
	CostByGPU    map[int]float64
	NVMLErrors   uint64
	ExportErrors uint64
	QueueSize    int
	AgentCPU     float64
	AgentMemory  float64
	PollDuration float64
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

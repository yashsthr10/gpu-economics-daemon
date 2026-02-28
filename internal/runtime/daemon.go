// Package runtime wires and runs the GPU Economic Telemetry Agent: collector, aggregator, cost, and OTLP export.
// Per ARCHITECTURE.md and the implementation plan Phase 6.
package runtime

import (
	"context"
	"log"
	"math"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/mavvrik/gpu-economics-agent/internal/aggregator"
	"github.com/mavvrik/gpu-economics-agent/internal/collector"
	"github.com/mavvrik/gpu-economics-agent/internal/config"
	"github.com/mavvrik/gpu-economics-agent/internal/cost"
	"github.com/mavvrik/gpu-economics-agent/internal/exporter"
	"github.com/mavvrik/gpu-economics-agent/internal/models"
	"github.com/mavvrik/gpu-economics-agent/internal/otel"
	"github.com/mavvrik/gpu-economics-agent/internal/validator"
)

// nvmlFailureRate returns failures / (failures + successes); 0 if no polls yet.
func nvmlFailureRate(errors, success uint64) float64 {
	total := errors + success
	if total == 0 {
		return 0
	}
	return float64(errors) / float64(total)
}

// Run starts the agent with the given config and blocks until ctx is cancelled or a signal is received.
// It starts the collector, aggregator, and configures the OTLP MeterProvider with metric callbacks.
// On shutdown it flushes the exporter and stops all goroutines.
func Run(ctx context.Context, cfg *config.Config, version string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		cancel()
	}()

	state := aggregator.NewState(cfg.Cost.IdleUtilizationThresholdPercent)
	samplesCh := make(chan models.NodeSample, 2)

	var nvmlErrors, nvmlSuccess atomic.Uint64
	var pollDurationBits, nvmlReadLatencyBits, integrationJitterBits atomic.Uint64
	onNVMLError := func() { nvmlErrors.Add(1) }
	onSuccess := func(pollDur, nvmlDur, jitter float64) {
		nvmlSuccess.Add(1)
		pollDurationBits.Store(math.Float64bits(pollDur))
		nvmlReadLatencyBits.Store(math.Float64bits(nvmlDur))
		integrationJitterBits.Store(math.Float64bits(jitter))
	}

	// OTLP MeterProvider and metric registration
	mpCfg := exporter.MeterProviderConfig{
		Endpoint:       cfg.OTEL.Endpoint,
		Insecure:       cfg.OTEL.Insecure,
		TLSCertPath:    cfg.OTEL.TLSCertPath,
		ServiceName:    "gpu-econ-agent",
		ServiceVersion: version,
		HostName:       cfg.NodeID,
		ClusterID:      cfg.ClusterID,
		NodeID:         cfg.NodeID,
	}
	mp, shutdown, err := exporter.NewMeterProvider(ctx, mpCfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Printf("meter provider shutdown: %v", err)
		}
	}()

	meter := mp.Meter("gpu-economics-agent")
	costCfg := cost.CostConfig{PUE: cfg.Cost.PUE, ElectricityCostPerKWh: cfg.Cost.ElectricityCostPerKWh}
	getSnapshot := func() otel.MetricsSnapshot {
		gauges, eKWhByGPU, eKWhIdleByGPU, eKWhActiveByGPU := state.Snapshot()
		costByGPU := cost.CostPerGPU(eKWhByGPU, costCfg)
		eKWhEffectiveByGPU := make(map[int]float64, len(eKWhByGPU))
		for id, e := range eKWhByGPU {
			eKWhEffectiveByGPU[id] = e * costCfg.PUE
		}
		flags := validator.Eval(gauges)
		return otel.MetricsSnapshot{
			Gauges:                gauges,
			EKWhByGPU:             eKWhByGPU,
			EKWhEffectiveByGPU:    eKWhEffectiveByGPU,
			EKWhIdleByGPU:         eKWhIdleByGPU,
			EKWhActiveByGPU:       eKWhActiveByGPU,
			CostByGPU:             costByGPU,
			AnomalySuspiciousLow:  flags.SuspiciousLowPower,
			AnomalyPowerLimitZero: flags.PowerLimitZero,
			RestartCount:          1, // one per process start
			NVMLErrors:            nvmlErrors.Load(),
			ExportErrors:          0,
			QueueSize:             0,
			AgentCPU:              0,
			AgentMemory:           0,
			PollDuration:          math.Float64frombits(pollDurationBits.Load()),
			ExportDurationSec:     0, // SDK does not expose flush duration hook in v1
			IntegrationJitterSec:  math.Float64frombits(integrationJitterBits.Load()),
			NVMLReadLatencySec:    math.Float64frombits(nvmlReadLatencyBits.Load()),
			NVMLFailureRate:       nvmlFailureRate(nvmlErrors.Load(), nvmlSuccess.Load()),
		}
	}
	if err := otel.RegisterMetrics(meter, getSnapshot); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		collector.Run(ctx, cfg.CollectionIntervalSeconds, cfg.ProcessAttribution.Enabled, samplesCh, onNVMLError, onSuccess)
		close(samplesCh)
	}()
	go func() {
		defer wg.Done()
		aggregator.Run(ctx, state, samplesCh)
	}()

	wg.Wait()
	return nil
}

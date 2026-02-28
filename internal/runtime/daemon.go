// Package runtime wires and runs the GPU Economic Telemetry Agent: collector, aggregator, cost, and OTLP export.
// Per ARCHITECTURE.md and the implementation plan Phase 6.
package runtime

import (
	"context"
	"log"
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
)

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

	state := aggregator.NewState()
	samplesCh := make(chan models.NodeSample, 2)

	var nvmlErrors atomic.Uint64
	onNVMLError := func() { nvmlErrors.Add(1) }

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
		gauges, eKWhByGPU := state.Snapshot()
		costByGPU := cost.CostPerGPU(eKWhByGPU, costCfg)
		return otel.MetricsSnapshot{
			Gauges:       gauges,
			EKWhByGPU:    eKWhByGPU,
			CostByGPU:    costByGPU,
			NVMLErrors:   nvmlErrors.Load(),
			ExportErrors: 0,
			QueueSize:    0,
			AgentCPU:     0,
			AgentMemory:  0,
			PollDuration: 0,
		}
	}
	if err := otel.RegisterMetrics(meter, getSnapshot); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		collector.Run(ctx, cfg.CollectionIntervalSeconds, cfg.ProcessAttribution.Enabled, samplesCh, onNVMLError)
		close(samplesCh)
	}()
	go func() {
		defer wg.Done()
		aggregator.Run(ctx, state, samplesCh)
	}()

	wg.Wait()
	return nil
}

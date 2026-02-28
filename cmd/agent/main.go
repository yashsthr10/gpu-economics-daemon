// Command agent is the main entrypoint for the GPU Economic Telemetry Agent daemon.
// It loads config, starts the collector-aggregator-exporter pipeline, and handles shutdown.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/mavvrik/gpu-economics-agent/internal/config"
	"github.com/mavvrik/gpu-economics-agent/internal/runtime"
)

// Version is set at build time via ldflags (e.g. -X main.Version=1.0.0).
var Version string

func main() {
	configPath := flag.String("config", "", "Path to YAML config file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("config path is required (use -config)")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if Version == "" {
		Version = "dev"
	}

	ctx := context.Background()
	if err := runtime.Run(ctx, cfg, Version); err != nil {
		log.Printf("runtime: %v", err)
		os.Exit(1)
	}
}

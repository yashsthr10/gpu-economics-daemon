// Package config loads and validates YAML configuration for the GPU Economic Telemetry Agent.
// Per ARCHITECTURE.md and SPECIFICATION.md section 4.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the validated agent configuration (SPECIFICATION.md section 4.2).
type Config struct {
	ClusterID                 string             `yaml:"cluster_id"`
	NodeID                    string             `yaml:"node_id"`
	CollectionIntervalSeconds int                `yaml:"collection_interval_seconds"`
	OTEL                      OTELConfig         `yaml:"otel"`
	Cost                      CostConfig         `yaml:"cost"`
	ProcessAttribution        ProcessAttribution `yaml:"process_attribution"`
}

// OTELConfig holds OpenTelemetry export settings.
type OTELConfig struct {
	Endpoint    string `yaml:"endpoint"`
	Insecure    bool   `yaml:"insecure"`
	TLSCertPath string `yaml:"tls_cert_path"`
}

// CostConfig holds cost engine parameters.
type CostConfig struct {
	ElectricityCostPerKWh           float64 `yaml:"electricity_cost_per_kwh"`
	PUE                             float64 `yaml:"pue"`
	IdleUtilizationThresholdPercent float64 `yaml:"idle_utilization_threshold_percent"`
}

// ProcessAttribution holds optional process attribution settings.
type ProcessAttribution struct {
	Enabled bool `yaml:"enabled"`
}

// Load reads and unmarshals the config file at path, then validates and resolves node_id.
// Returns an error if file read, unmarshal, or validation fails.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(&cfg)
	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	ResolveNodeID(&cfg)
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.CollectionIntervalSeconds == 0 {
		cfg.CollectionIntervalSeconds = 5
	}
	if cfg.Cost.PUE == 0 {
		cfg.Cost.PUE = 1.0
	}
	if cfg.Cost.IdleUtilizationThresholdPercent == 0 {
		cfg.Cost.IdleUtilizationThresholdPercent = 10
	}
}

// Validate checks all validation rules (SPECIFICATION.md section 4.3).
// Returns an error describing the first violation.
func Validate(cfg *Config) error {
	if strings.TrimSpace(cfg.ClusterID) == "" {
		return fmt.Errorf("cluster_id is required and must be non-empty")
	}
	if strings.TrimSpace(cfg.NodeID) == "" {
		return fmt.Errorf("node_id is required and must be non-empty")
	}
	if cfg.CollectionIntervalSeconds < 1 || cfg.CollectionIntervalSeconds > 300 {
		return fmt.Errorf("collection_interval_seconds must be between 1 and 300, got %d", cfg.CollectionIntervalSeconds)
	}
	if strings.TrimSpace(cfg.OTEL.Endpoint) == "" {
		return fmt.Errorf("otel.endpoint is required and must be non-empty")
	}
	if cfg.Cost.PUE < 1.0 {
		return fmt.Errorf("cost.pue must be >= 1.0, got %f", cfg.Cost.PUE)
	}
	if cfg.Cost.ElectricityCostPerKWh < 0 {
		return fmt.Errorf("cost.electricity_cost_per_kwh must be >= 0, got %f", cfg.Cost.ElectricityCostPerKWh)
	}
	if cfg.Cost.IdleUtilizationThresholdPercent != 0 && (cfg.Cost.IdleUtilizationThresholdPercent < 1 || cfg.Cost.IdleUtilizationThresholdPercent > 100) {
		return fmt.Errorf("cost.idle_utilization_threshold_percent must be between 1 and 100, got %f", cfg.Cost.IdleUtilizationThresholdPercent)
	}
	return nil
}

// ResolveNodeID sets NodeID to hostname when "auto", then tries NODE_NAME env (K8s) if still empty.
func ResolveNodeID(cfg *Config) {
	if cfg.NodeID != "auto" {
		return
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		cfg.NodeID = h
		return
	}
	if n := os.Getenv("NODE_NAME"); n != "" {
		cfg.NodeID = n
		return
	}
	cfg.NodeID = "unknown"
}

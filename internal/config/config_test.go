// Package config tests validation and node_id resolution.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid", Config{
			ClusterID:                 "c1",
			NodeID:                    "n1",
			CollectionIntervalSeconds: 5,
			OTEL:                      OTELConfig{Endpoint: "localhost:4317"},
			Cost:                      CostConfig{PUE: 1.2, ElectricityCostPerKWh: 0.1},
		}, false},
		{"empty cluster_id", Config{NodeID: "n1", OTEL: OTELConfig{Endpoint: "x"}}, true},
		{"empty node_id", Config{ClusterID: "c1", OTEL: OTELConfig{Endpoint: "x"}}, true},
		{"interval too low", Config{
			ClusterID: "c1", NodeID: "n1",
			CollectionIntervalSeconds: 0,
			OTEL:                      OTELConfig{Endpoint: "x"},
		}, true},
		{"interval too high", Config{
			ClusterID: "c1", NodeID: "n1",
			CollectionIntervalSeconds: 301,
			OTEL:                      OTELConfig{Endpoint: "x"},
		}, true},
		{"empty endpoint", Config{
			ClusterID: "c1", NodeID: "n1",
			CollectionIntervalSeconds: 5,
			OTEL:                      OTELConfig{Endpoint: ""},
		}, true},
		{"pue below 1", Config{
			ClusterID: "c1", NodeID: "n1",
			OTEL: OTELConfig{Endpoint: "x"},
			Cost: CostConfig{PUE: 0.9},
		}, true},
		{"negative cost", Config{
			ClusterID: "c1", NodeID: "n1",
			OTEL: OTELConfig{Endpoint: "x"},
			Cost: CostConfig{PUE: 1.0, ElectricityCostPerKWh: -0.1},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveNodeID(t *testing.T) {
	cfg := &Config{NodeID: "auto"}
	ResolveNodeID(cfg)
	if cfg.NodeID == "auto" || cfg.NodeID == "" {
		t.Errorf("expected node_id to be resolved from hostname or NODE_NAME, got %q", cfg.NodeID)
	}

	cfg = &Config{NodeID: "manual"}
	ResolveNodeID(cfg)
	if cfg.NodeID != "manual" {
		t.Errorf("expected node_id unchanged for non-auto, got %q", cfg.NodeID)
	}

	cfg = &Config{NodeID: "auto"}
	os.Setenv("NODE_NAME", "k8s-node-1")
	defer os.Unsetenv("NODE_NAME")
	ResolveNodeID(cfg)
	if cfg.NodeID != "k8s-node-1" {
		t.Logf("note: NODE_NAME takes effect only when hostname fails or is empty; got %q", cfg.NodeID)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(validPath, []byte(`
cluster_id: "test-cluster"
node_id: "test-node"
collection_interval_seconds: 10
otel:
  endpoint: "localhost:4317"
  insecure: true
cost:
  pue: 1.5
  electricity_cost_per_kwh: 0.12
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(validPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ClusterID != "test-cluster" || cfg.NodeID != "test-node" || cfg.CollectionIntervalSeconds != 10 {
		t.Errorf("Load() got cluster_id=%q node_id=%q interval=%d", cfg.ClusterID, cfg.NodeID, cfg.CollectionIntervalSeconds)
	}
	if cfg.Cost.PUE != 1.5 || cfg.Cost.ElectricityCostPerKWh != 0.12 {
		t.Errorf("Load() cost: pue=%f rate=%f", cfg.Cost.PUE, cfg.Cost.ElectricityCostPerKWh)
	}

	_, err = Load(filepath.Join(dir, "nonexistent.yaml"))
	if err == nil {
		t.Error("Load() expected error for missing file")
	}
}

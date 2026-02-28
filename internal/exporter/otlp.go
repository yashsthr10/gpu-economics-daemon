// Package exporter provides the OTLP gRPC metrics exporter and MeterProvider setup.
// Per SPECIFICATION.md section 5 and ARCHITECTURE.md section 2.2.5.
package exporter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"
)

// MeterProviderConfig holds endpoint and resource attributes for the OTLP pipeline.
type MeterProviderConfig struct {
	Endpoint    string
	Insecure    bool
	TLSCertPath string
	// Resource attributes (SPECIFICATION.md 5.2)
	ServiceName    string
	ServiceVersion string
	HostName       string
	ClusterID      string
	NodeID         string
	Environment    string
}

// NewMeterProvider creates a MeterProvider that exports to the configured OTLP gRPC endpoint.
// Caller must call provider.Shutdown(ctx) when done. Export runs on the SDK's PeriodicReader interval.
func NewMeterProvider(ctx context.Context, cfg MeterProviderConfig) (*metric.MeterProvider, func(context.Context) error, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	} else if cfg.TLSCertPath != "" {
		data, err := os.ReadFile(cfg.TLSCertPath)
		if err != nil {
			return nil, nil, fmt.Errorf("read TLS cert: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(data)
		creds := credentials.NewTLS(&tls.Config{RootCAs: pool}) //nolint:gosec
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(creds))
	}

	exp, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("otlp exporter: %w", err)
	}

	reader := metric.NewPeriodicReader(exp, metric.WithInterval(10*time.Second))
	attrs := []attribute.KeyValue{
		attribute.String("service.name", firstNonEmpty(cfg.ServiceName, "gpu-econ-agent")),
		attribute.String("service.version", cfg.ServiceVersion),
		attribute.String("host.name", cfg.HostName),
		attribute.String("cluster.id", cfg.ClusterID),
		attribute.String("node.id", cfg.NodeID),
	}
	if cfg.Environment != "" {
		attrs = append(attrs, attribute.String("environment", cfg.Environment))
	}
	res := resource.NewSchemaless(attrs...)

	provider := metric.NewMeterProvider(metric.WithResource(res), metric.WithReader(reader))
	shutdown := func(ctx context.Context) error {
		return provider.Shutdown(ctx)
	}
	return provider, shutdown, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

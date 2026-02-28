# internal/exporter

Sets up the OTLP gRPC metrics exporter and MeterProvider with resource attributes. The actual export is performed by the OpenTelemetry SDK’s PeriodicReader at a fixed interval.

## Components

### otlp.go

- **MeterProviderConfig** — Endpoint, Insecure, TLSCertPath; plus resource attributes: ServiceName, ServiceVersion, HostName, ClusterID, NodeID, Environment (optional).
- **NewMeterProvider(ctx, cfg)** — Creates the OTLP gRPC exporter (using `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc`), a PeriodicReader (e.g. 10s interval), and a MeterProvider with a resource built from the config. If `TLSCertPath` is set and Insecure is false, loads the PEM file and uses it as the root CA for the gRPC connection. Returns the MeterProvider, a shutdown function, and an error.

Resource attributes attached to all metrics (per [SPECIFICATION.md 5.2](../../docs/SPECIFICATION.md)): service.name, service.version, host.name, cluster.id, node.id; optionally environment.

## Behaviour

- **Endpoint** — Passed to the OTLP client (e.g. `localhost:4317`).
- **Insecure** — If true, gRPC uses insecure credentials.
- **TLS** — If Insecure is false and TLSCertPath is non-empty, server verification uses the given CA cert. Client certs (mTLS) can be added in a future extension.

The returned MeterProvider is passed to the runtime; the runtime gets a Meter, registers metrics (via `otel.RegisterMetrics`), and on shutdown calls the returned shutdown function to flush and tear down the exporter.

## Dependencies

- **go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc**
- **go.opentelemetry.io/otel/sdk/metric**
- **go.opentelemetry.io/otel/sdk/resource**
- **go.opentelemetry.io/otel/attribute**
- **google.golang.org/grpc/credentials** (for TLS)

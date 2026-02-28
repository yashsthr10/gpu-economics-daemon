# GPU Economic Telemetry Agent — Documentation Index

This folder contains the detailed specification, architecture, deployment, and observability reference for the GPU Economic Telemetry Agent. The service is written in **Go (Golang)**.

---

## Documents

| Document | Description |
|----------|-------------|
| [SPECIFICATION.md](SPECIFICATION.md) | Full technical specification: requirements, data models, configuration schema, OTel contract, error handling, security, and compliance. |
| [ARCHITECTURE.md](ARCHITECTURE.md) | System architecture: components, data flow, concurrency model, Go package layout, deployment overview, and integration points. |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Deployment guide: bare-metal (systemd), Kubernetes (DaemonSet), config examples, security context, and operational checklist. |
| [METRICS_AND_TRACES.md](METRICS_AND_TRACES.md) | OpenTelemetry reference: resource attributes, metric names/types/attributes, trace span schema, cardinality rules, and example queries. |

---

## Quick Reference

- **Source spec:** See project root `gpu_economic_telemetry_agent_spec(1).md` for the original high-level specification.
- **Implementation:** Follow `docs/SPECIFICATION.md` and `docs/ARCHITECTURE.md` for design and package structure; use `docs/DEPLOYMENT.md` and `docs/METRICS_AND_TRACES.md` for operations and observability.

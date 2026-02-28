# Multi-stage build for GPU Economic Telemetry Agent.
# Runtime requires NVIDIA drivers and libnvidia-ml.so (NVML); typically run on GPU nodes with nvidia-container-toolkit.
# CGO is enabled for NVML bindings; build on Debian for glibc compatibility with typical GPU hosts.
FROM golang:1.24-bookworm AS builder
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build with CGO for go-nvml (Linux only).
ENV CGO_ENABLED=1
RUN go build -ldflags "-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" -o gpu-econ-agent ./cmd/agent/

# Minimal runtime: agent binary; NVML comes from host or nvidia container toolkit.
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /build/gpu-econ-agent /usr/local/bin/gpu-econ-agent
# Default config path; override with volume or ConfigMap.
RUN mkdir -p /etc/gpu-econ-agent
ENTRYPOINT ["/usr/local/bin/gpu-econ-agent"]
CMD ["-config", "/etc/gpu-econ-agent/config.yaml"]

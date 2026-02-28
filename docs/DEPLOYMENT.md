# GPU Economic Telemetry Agent — Deployment Guide

This document describes how to deploy the GPU Economic Telemetry Agent (a Go service) on bare-metal Linux and Kubernetes, including configuration, security context, and operational considerations.

---

## 1. Deployment Modes Overview

| Mode | Use Case | Process Manager | Config Source |
|------|----------|-----------------|---------------|
| Bare metal | Physical servers, on-prem GPU nodes | systemd | File path |
| Kubernetes | GPU nodes in K8s cluster | DaemonSet | ConfigMap / file |
| VM | Cloud or on-prem GPU VMs | systemd or init | File path |

---

## 2. Bare-Metal Deployment

### 2.1 Binary and Layout

- **Binary:** Single static executable built from the Go codebase (e.g., `gpu-econ-agent`).
- **Suggested paths:**
  - Binary: `/usr/local/bin/gpu-econ-agent`
  - Config: `/etc/gpu-econ-agent/config.yaml`
  - Logs: Optional; e.g., `/var/log/gpu-econ-agent/` (if file logging enabled)

### 2.2 systemd Unit Example

```ini
[Unit]
Description=GPU Economic Telemetry Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gpu-econ-agent --config /etc/gpu-econ-agent/config.yaml
Restart=on-failure
RestartSec=10
User=root
Group=root
# Optional: restrict capabilities
# CapabilityBoundingSet=CAP_SYS_ADMIN
# NoNewPrivileges=yes

[Install]
WantedBy=multi-user.target
```

If the agent can run as non-root with access to `/dev/nvidia*` (e.g., via udev rules or group), set `User` and `Group` accordingly.

### 2.3 Configuration (Bare Metal)

Example `/etc/gpu-econ-agent/config.yaml`:

```yaml
cluster_id: research-cluster-01
node_id: auto
collection_interval_seconds: 5

otel:
  endpoint: "otel-collector.example.com:4317"
  insecure: false
  tls_cert_path: "/etc/gpu-econ-agent/certs/ca.pem"

cost:
  electricity_cost_per_kwh: 0.12
  pue: 1.4

process_attribution:
  enabled: true
```

### 2.4 Permissions and Devices

- The process must be able to **read** `/dev/nvidia*` (and optionally `/dev/nvidiactl`, `/dev/nvidia-uvm`).
- Read-only access to `/proc` (and optionally cgroup mounts) for process attribution.
- No write access required except optional log directory.

---

## 3. Kubernetes Deployment (DaemonSet)

### 3.1 Design Choices

- **DaemonSet:** One pod per node; use `nodeSelector` or taints/tolerations so that only GPU nodes run the agent.
- **Single container:** One container per pod; the agent binary is the main process.
- **No sidecars required:** Agent exports via OTLP; no Prometheus scrape needed on the pod.

### 3.2 Resource Limits

Align with specification targets:

```yaml
resources:
  requests:
    memory: "32Mi"
    cpu: "50m"
  limits:
    memory: "64Mi"
    cpu: "200m"
```

### 3.3 Volume and Mounts

| Mount | Purpose |
|-------|--------|
| `/dev` (or hostPath `/dev`) | Access to `nvidia*` devices |
| `/proc` (read-only) | Process attribution |
| Config | ConfigMap or Secret mounted at e.g. `/etc/gpu-econ-agent/config.yaml` |

### 3.4 SecurityContext (Recommended)

```yaml
securityContext:
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  runAsNonRoot: true   # If possible; otherwise runAsUser for a dedicated user
  runAsUser: 1000
  runAsGroup: 1000
  capabilities:
    drop:
      - ALL
```

If NVML requires root or specific capabilities (e.g., CAP_SYS_ADMIN) in your environment, document the minimum needed and set only those.

### 3.5 DaemonSet Manifest Sketch

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: gpu-econ-agent
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: gpu-econ-agent
  template:
    metadata:
      labels:
        app: gpu-econ-agent
    spec:
      nodeSelector:
        nvidia.com/gpu.present: "true"   # Example: only GPU nodes
      hostPID: true
      containers:
        - name: agent
          image: your-registry/gpu-econ-agent:latest
          args:
            - "--config=/etc/gpu-econ-agent/config.yaml"
          volumeMounts:
            - name: dev
              mountPath: /dev
            - name: proc
              mountPath: /proc
              readOnly: true
            - name: config
              mountPath: /etc/gpu-econ-agent
              readOnly: true
          resources:
            requests:
              memory: "32Mi"
              cpu: "50m"
            limits:
              memory: "64Mi"
              cpu: "200m"
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
      volumes:
        - name: dev
          hostPath:
            path: /dev
        - name: proc
          hostPath:
            path: /proc
        - name: config
          configMap:
            name: gpu-econ-agent-config
```

### 3.6 ConfigMap for Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gpu-econ-agent-config
  namespace: monitoring
data:
  config.yaml: |
    cluster_id: research-cluster-01
    node_id: auto
    collection_interval_seconds: 5
    otel:
      endpoint: "otel-collector.monitoring.svc.cluster.local:4317"
      insecure: false
    cost:
      electricity_cost_per_kwh: 0.12
      pue: 1.4
    process_attribution:
      enabled: true
```

When `node_id: auto`, the agent can resolve the node name from the `NODE_NAME` environment variable (set by K8s downward API) or hostname.

### 3.7 Optional: NODE_NAME via Downward API

```yaml
env:
  - name: NODE_NAME
    valueFrom:
      fieldRef:
        fieldPath: spec.nodeName
```

---

## 4. OTLP Collector and Backends

- **Agent** sends OTLP (gRPC) to a single endpoint (e.g., OpenTelemetry Collector or Grafana Agent).
- **Collector** can:
  - Export metrics to Prometheus (remote write) or Prometheus-compatible backends
  - Export traces to Tempo, Jaeger, or vendor backends
- Ensure **network policy** allows egress from agent pods/nodes to the collector (e.g., port 4317).

---

## 5. Operational Checklist

- [ ] Config file validated (cluster_id, node_id, otel.endpoint, cost params).
- [ ] OTLP endpoint reachable from each node (DNS and firewall).
- [ ] TLS/mTLS certs installed if `insecure: false`.
- [ ] DaemonSet runs only on GPU nodes (nodeSelector/taints).
- [ ] SecurityContext set (readOnlyRootFilesystem, no privilege escalation).
- [ ] Resource limits set to avoid resource contention.
- [ ] Backend (Prometheus/Tempo) scraping or receiving data from the collector.

---

*End of deployment guide.*

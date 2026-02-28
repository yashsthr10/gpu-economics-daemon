# Deployment artifacts for GPU Economic Telemetry Agent

## systemd (bare-metal)

- Copy `systemd/gpu-econ-agent.service` to `/etc/systemd/system/`.
- Create `/etc/gpu-econ-agent/config.yaml` (see [docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)).
- Install binary to `/usr/local/bin/gpu-econ-agent`.
- Run: `systemctl daemon-reload && systemctl enable --now gpu-econ-agent`.

## Kubernetes

- Build and push the image: `docker build -t your-registry/gpu-econ-agent:latest .` (from repo root).
- Set `nodeSelector` in `kubernetes/daemonset.yaml` to match your GPU nodes (e.g. `nvidia.com/gpu.present: "true"`).
- Apply ConfigMap then DaemonSet: `kubectl apply -f kubernetes/configmap.yaml -f kubernetes/daemonset.yaml`.
- Ensure NODE_NAME is set via downward API (included in daemonset.yaml) so `node_id: auto` resolves correctly.

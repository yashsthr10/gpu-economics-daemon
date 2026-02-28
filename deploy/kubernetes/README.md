# deploy/kubernetes

Kubernetes manifests for running the GPU Economic Telemetry Agent as a DaemonSet on GPU nodes.

## Files

- **configmap.yaml** — ConfigMap with `config.yaml` for the agent (cluster_id, node_id, otel endpoint, cost settings). Adjust `cluster_id` and `otel.endpoint` for your environment.
- **daemonset.yaml** — DaemonSet: one pod per node; use `nodeSelector` (e.g. `nvidia.com/gpu.present: "true"`) so only GPU nodes run the agent. Mounts host `/dev` and `/proc`, sets `NODE_NAME` via downward API for `node_id: auto`.

## Apply

1. Build and push the agent image (e.g. `your-registry/gpu-econ-agent:latest`). Update the image in `daemonset.yaml` if needed.
2. Ensure the `monitoring` namespace exists, or change `namespace` in both manifests.
3. Apply: `kubectl apply -f configmap.yaml -f daemonset.yaml`.

## Notes

- The agent needs access to NVML (typically via host `/dev` and the node’s NVIDIA driver). When using the NVIDIA device plugin, the node selector should target GPU nodes.
- For `node_id: auto`, the DaemonSet sets the `NODE_NAME` env from `spec.nodeName` so the agent reports the correct node identifier.

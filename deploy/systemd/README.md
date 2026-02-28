# deploy/systemd

systemd unit file for running the GPU Economic Telemetry Agent on bare-metal Linux.

## Files

- **gpu-econ-agent.service** — Unit that runs the agent as a simple service; restarts on failure.

## Install

1. Copy the binary to `/usr/local/bin/gpu-econ-agent`.
2. Create `/etc/gpu-econ-agent/config.yaml` (see [docs/DEPLOYMENT.md](../../docs/DEPLOYMENT.md) for schema).
3. Copy `gpu-econ-agent.service` to `/etc/systemd/system/`.
4. Run: `systemctl daemon-reload && systemctl enable --now gpu-econ-agent`.

## Customisation

- Edit `ExecStart` if the binary or config path differs.
- Adjust `User`/`Group` if the agent runs as non-root (ensure access to `/dev/nvidia*` and NVML).

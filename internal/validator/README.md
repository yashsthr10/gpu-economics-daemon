# internal/validator

Evaluates GPU telemetry for sanity checks and anomaly flags. Used by the runtime to populate `gpu.anomaly.*` gauges.

## Behaviour

- **Eval(gauges)** takes the current per-GPU snapshot (from the aggregator) and returns:
  - **SuspiciousLowPower:** 1 for each GPU where power &lt; 5W and the GPU appears to be a discrete GPU (PowerLimitMaxW >= 50W); 0 otherwise.
  - **PowerLimitZero:** 1 for each GPU where NVML reported power limit as 0; 0 otherwise.

Results are passed into the OTel metrics snapshot and exported as `gpu.anomaly.suspicious_low_power` and `gpu.anomaly.power_limit_zero` (gauges, 0 or 1 per `gpu.id`). No persistent state; evaluation is stateless per call.

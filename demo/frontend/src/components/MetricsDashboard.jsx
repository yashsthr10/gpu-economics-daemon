/**
 * Displays aggregated GPU metrics from parsed ingestion data: totals, averages, and per-GPU breakdown.
 */
function num(v) {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

export default function MetricsDashboard({ analysis }) {
  if (!analysis) return null
  const {
    totalEnergyKwh,
    totalCost,
    avgUtilization,
    avgMemoryUtil,
    avgPowerWatts,
    maxTempC,
    totalNvmlErrors,
    batchCount,
    gpuCount,
    byGpu,
    sampleCount = {},
  } = analysis

  const gpuIds = Object.keys(byGpu || {}).sort()
  const allZeros =
    batchCount > 0 &&
    num(totalEnergyKwh) === 0 &&
    num(totalCost) === 0 &&
    num(avgPowerWatts) === 0 &&
    num(maxTempC) === 0

  return (
    <section className="dashboard">
      <p className="meta">
        {batchCount} metric batch(es), {gpuCount} GPU(s) detected
        {sampleCount && (sampleCount.power != null || sampleCount.util != null) && (
          <> &mdash; samples: power {sampleCount.power ?? 0}, util {sampleCount.util ?? 0}, temp {sampleCount.temp ?? 0}</>
        )}
      </p>
      {allZeros && (
        <p className="warning" role="alert">
          Data loaded but all values are zero. Check that the file is JSONL with a &quot;metrics&quot; array per line (simple format from the ingestion service).
        </p>
      )}

      <h2 className="section-title">Summary</h2>
      <div className="card-grid">
        <div className="card">
          <div className="card-label">Total energy</div>
          <div className="card-value">
            {num(totalEnergyKwh).toFixed(4)}
            <span className="card-unit">kWh</span>
          </div>
        </div>
        <div className="card">
          <div className="card-label">Total cost</div>
          <div className="card-value">
            {num(totalCost).toFixed(4)}
            <span className="card-unit"></span>
          </div>
        </div>
        <div className="card">
          <div className="card-label">Avg utilization</div>
          <div className="card-value">
            {num(avgUtilization).toFixed(2)}
            <span className="card-unit">%</span>
          </div>
        </div>
        <div className="card">
          <div className="card-label">Avg memory util</div>
          <div className="card-value">
            {num(avgMemoryUtil).toFixed(2)}
            <span className="card-unit">%</span>
          </div>
        </div>
        <div className="card">
          <div className="card-label">Avg power</div>
          <div className="card-value">
            {num(avgPowerWatts).toFixed(2)}
            <span className="card-unit">W</span>
          </div>
        </div>
        <div className="card">
          <div className="card-label">Max temperature</div>
          <div className="card-value">
            {num(maxTempC).toFixed(1)}
            <span className="card-unit">C</span>
          </div>
        </div>
        <div className="card">
          <div className="card-label">NVML errors</div>
          <div className="card-value">
            {num(totalNvmlErrors)}
            <span className="card-unit"></span>
          </div>
        </div>
      </div>

      {gpuIds.length > 0 && (
        <>
          <h2 className="section-title">Per GPU</h2>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>GPU</th>
                  <th>Energy (kWh)</th>
                  <th>Cost</th>
                </tr>
              </thead>
              <tbody>
                {gpuIds.map((id) => (
                  <tr key={id}>
                    <td>{id}</td>
                    <td>{(byGpu[id].energyKwh ?? 0).toFixed(4)}</td>
                    <td>{(byGpu[id].cost ?? 0).toFixed(4)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </section>
  )
}

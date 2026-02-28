/**
 * Displays aggregated GPU metrics from parsed ingestion data: totals, averages, and per-GPU breakdown.
 */
import TrendCharts from './TrendCharts'

function num(v) {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

function formatDuration(seconds) {
  const s = Math.floor(num(seconds))
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  const sec = s % 60
  if (m < 60) return sec ? `${m}m ${sec}s` : `${m}m`
  const h = Math.floor(m / 60)
  const min = m % 60
  return min ? `${h}h ${min}m` : `${h}h`
}

function formatTime(iso) {
  if (!iso) return '–'
  try {
    const d = new Date(iso)
    return Number.isNaN(d.getTime()) ? '–' : d.toLocaleString()
  } catch {
    return '–'
  }
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
    firstReceivedAt,
    lastReceivedAt,
    durationSeconds,
    costPerHour,
    energyPerHour,
    projected24hCost,
    projected24hEnergy,
    efficiencyClass,
    batches = [],
  } = analysis

  const gpuIds = Object.keys(byGpu || {}).sort()
  const avgUtilNum = num(avgUtilization)
  const avgPowerNum = num(avgPowerWatts)
  const maxTempNum = num(maxTempC)
  const wPerPercentUtil = avgUtilNum >= 1 ? avgPowerNum / avgUtilNum : null
  const thermalStatus = maxTempNum < 85 ? 'Normal' : 'High'
  const headroomC = maxTempNum < 85 ? 85 - maxTempNum : 0
  const idleBanner = avgUtilNum < 20 && avgPowerNum > 10
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

      {idleBanner && (
        <p className="warning" role="alert">
          GPU underutilized — burning idle power (low utilization with non-idle power draw).
        </p>
      )}

      {(durationSeconds > 0 || firstReceivedAt || efficiencyClass) && (
        <>
          <h2 className="section-title">Session and burn rate</h2>
          <div className="card-grid">
            {durationSeconds > 0 && (
              <>
                <div className="card">
                  <div className="card-label">Duration</div>
                  <div className="card-value">{formatDuration(durationSeconds)}</div>
                </div>
                <div className="card">
                  <div className="card-label">Start time</div>
                  <div className="card-value card-value-small">{formatTime(firstReceivedAt)}</div>
                </div>
                <div className="card">
                  <div className="card-label">End time</div>
                  <div className="card-value card-value-small">{formatTime(lastReceivedAt)}</div>
                </div>
              </>
            )}
            {durationSeconds > 0 && (
              <>
                <div className="card">
                  <div className="card-label">Cost per hour</div>
                  <div className="card-value">
                    {num(costPerHour).toFixed(4)}
                    <span className="card-unit"></span>
                  </div>
                </div>
                <div className="card">
                  <div className="card-label">Energy per hour</div>
                  <div className="card-value">
                    {num(energyPerHour).toFixed(4)}
                    <span className="card-unit">kWh</span>
                  </div>
                </div>
                <div className="card">
                  <div className="card-label">Projected 24h cost</div>
                  <div className="card-value">
                    {num(projected24hCost).toFixed(4)}
                    <span className="card-unit"></span>
                  </div>
                </div>
                <div className="card">
                  <div className="card-label">Projected 24h energy</div>
                  <div className="card-value">
                    {num(projected24hEnergy).toFixed(4)}
                    <span className="card-unit">kWh</span>
                  </div>
                </div>
              </>
            )}
            {efficiencyClass && (
              <div className="card">
                <div className="card-label">Efficiency</div>
                <div className="card-value">{efficiencyClass}</div>
              </div>
            )}
          </div>
        </>
      )}

      <TrendCharts batches={batches} />

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
            {maxTempNum.toFixed(1)}
            <span className="card-unit">C</span>
          </div>
        </div>
        {(thermalStatus || headroomC >= 0) && (
          <>
            <div className="card">
              <div className="card-label">Thermal status</div>
              <div className="card-value">{thermalStatus}</div>
            </div>
            {headroomC > 0 && (
              <div className="card">
                <div className="card-label">Headroom</div>
                <div className="card-value">
                  {headroomC.toFixed(0)}
                  <span className="card-unit">C</span>
                </div>
              </div>
            )}
          </>
        )}
        {wPerPercentUtil != null && (
          <div className="card">
            <div className="card-label">W per % utilization</div>
            <div className="card-value">
              {wPerPercentUtil.toFixed(2)}
              <span className="card-unit">W/%</span>
            </div>
          </div>
        )}
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

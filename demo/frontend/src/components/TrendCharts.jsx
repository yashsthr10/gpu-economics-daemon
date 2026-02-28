/**
 * Renders trend line charts from parsed batches: power over time, utilization over time,
 * and energy accumulation over time.
 */
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'

function avgPoints(points) {
  if (!points || points.length === 0) return 0
  const sum = points.reduce((a, p) => a + p.value, 0)
  return sum / points.length
}

function sumPoints(points) {
  if (!points || points.length === 0) return 0
  return points.reduce((a, p) => a + p.value, 0)
}

export default function TrendCharts({ batches }) {
  if (!Array.isArray(batches) || batches.length === 0) return null

  const powerSeries = batches.map((b) => ({
    time: b.received_at,
    power: avgPoints(b.byName?.['gpu.power.watts']),
  }))
  const utilSeries = batches.map((b) => ({
    time: b.received_at,
    util: avgPoints(b.byName?.['gpu.utilization.percent']),
  }))
  const energySeries = batches.map((b) => ({
    time: b.received_at,
    energy: sumPoints(b.byName?.['gpu.energy.kwh.total']),
  }))

  const hasPower = powerSeries.some((d) => d.power > 0)
  const hasUtil = utilSeries.some((d) => d.util > 0)
  const hasEnergy = energySeries.some((d) => d.energy > 0)
  if (!hasPower && !hasUtil && !hasEnergy) return null

  return (
    <section className="trend-charts">
      <h2 className="section-title">Trends</h2>
      {hasPower && (
        <div className="chart-wrap">
          <h3 className="chart-title">Power (W) over time</h3>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={powerSeries} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#30363d" />
              <XAxis dataKey="time" tick={{ fontSize: 10 }} tickFormatter={(v) => (v ? new Date(v).toLocaleTimeString() : '')} />
              <YAxis tick={{ fontSize: 10 }} unit=" W" />
              <Tooltip labelFormatter={(v) => (v ? new Date(v).toLocaleString() : '')} formatter={(v) => [Number(v).toFixed(2), 'Power (W)']} />
              <Line type="monotone" dataKey="power" stroke="#58a6ff" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
      {hasUtil && (
        <div className="chart-wrap">
          <h3 className="chart-title">Utilization (%) over time</h3>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={utilSeries} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#30363d" />
              <XAxis dataKey="time" tick={{ fontSize: 10 }} tickFormatter={(v) => (v ? new Date(v).toLocaleTimeString() : '')} />
              <YAxis tick={{ fontSize: 10 }} unit=" %" />
              <Tooltip labelFormatter={(v) => (v ? new Date(v).toLocaleString() : '')} formatter={(v) => [Number(v).toFixed(2), 'Util %']} />
              <Line type="monotone" dataKey="util" stroke="#3fb950" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
      {hasEnergy && (
        <div className="chart-wrap">
          <h3 className="chart-title">Energy (kWh) accumulation over time</h3>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={energySeries} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#30363d" />
              <XAxis dataKey="time" tick={{ fontSize: 10 }} tickFormatter={(v) => (v ? new Date(v).toLocaleTimeString() : '')} />
              <YAxis tick={{ fontSize: 10 }} />
              <Tooltip labelFormatter={(v) => (v ? new Date(v).toLocaleString() : '')} formatter={(v) => [Number(v).toFixed(6), 'Energy (kWh)']} />
              <Line type="monotone" dataKey="energy" stroke="#d29922" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </section>
  )
}

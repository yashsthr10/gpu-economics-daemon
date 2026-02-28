/**
 * Parses JSONL ingestion output: one JSON object per line.
 * Supports:
 * - OTLP (valid OTel): { "received_at", "type": "metrics", "data": { "resourceMetrics": [ { "resource", "scopeMetrics": [ { "scope", "metrics": [ { "name", "gauge"|"sum", "dataPoints": [{ "attributes", "asDouble"|"asInt" }] } ] } ] } ] } }
 *   (resourceMetrics -> scopeMetrics -> metrics -> gauge/sum -> dataPoints; camelCase from protojson)
 * - Flat/simple: { "received_at", "type": "metrics", "metrics": [ { "name", "gpu_id", "value", "unit" }, ... ] }
 * Extracts GPU metrics for analysis and builds batches for trend charts.
 */

const DEBUG = true
function log(...args) {
  if (DEBUG && typeof console !== 'undefined') console.log('[parseIngestion]', ...args)
}

/** OTLP NumberDataPoint value: protojson uses camelCase (asDouble, asInt). */
function getDataPointValue(dp) {
  const v = dp.asDouble ?? dp.as_double ?? dp.asInt ?? dp.as_int
  if (v !== undefined && v !== null) return Number(v)
  return 0
}

/** OTLP attributes: key "gpu.id", value.stringValue (camelCase) or string_value. */
function getGpuId(dp) {
  const attrs = dp.attributes || []
  const gpu = attrs.find((a) => a && a.key === 'gpu.id')
  const val = gpu?.value
  if (!val || typeof val !== 'object') return 'global'
  return (val.stringValue ?? val.string_value ?? '').toString() || 'global'
}

/** Convert OTLP data (resourceMetrics -> scopeMetrics -> metrics -> gauge/sum -> dataPoints) into byName for batches and aggregation. */
function dataToByName(data) {
  const byName = {}
  const rms = data?.resourceMetrics || data?.resource_metrics || []
  for (const rm of rms) {
    const sms = rm.scopeMetrics || rm.scope_metrics || []
    for (const sm of sms) {
      const ms = sm.metrics || []
      for (const m of ms) {
        const name = m.name
        if (!name) continue
        const gaugeOrSum = m.gauge || m.sum
        const points = gaugeOrSum?.dataPoints || gaugeOrSum?.data_points || []
        if (!byName[name]) byName[name] = []
        for (const dp of points) {
          const value = getDataPointValue(dp)
          const gpuId = getGpuId(dp)
          byName[name].push({ gpuId, value })
        }
      }
    }
  }
  return byName
}

/** Collect from nested OTLP data (resourceMetrics/scopeMetrics/metrics). */
function collectDataPoints(metrics, metricName) {
  const out = []
  const rms = metrics.resourceMetrics || metrics.resource_metrics || []
  for (const rm of rms) {
    const sms = rm.scopeMetrics || rm.scope_metrics || []
    for (const sm of sms) {
      const ms = sm.metrics || []
      for (const m of ms) {
        if (m.name !== metricName) continue
        const gaugeOrSum = m.gauge || m.sum
        const points = gaugeOrSum?.dataPoints || gaugeOrSum?.data_points || []
        for (const dp of points) {
          out.push({
            gpuId: getGpuId(dp),
            value: getDataPointValue(dp),
            timeUnixNano: dp.timeUnixNano,
          })
        }
      }
    }
  }
  return out
}

/** Collect from simple format: entry.metrics = [ { name, gpu_id, value, unit }, ... ]. */
function collectFromSimpleMetrics(metricsArray) {
  const byName = {}
  for (const p of metricsArray || []) {
    const name = p.name
    if (!name) continue
    const num = Number(p.value)
    const value = Number.isFinite(num) ? num : 0
    if (!byName[name]) byName[name] = []
    byName[name].push({
      gpuId: p.gpu_id ?? p.gpuId ?? 'global',
      value,
    })
  }
  return byName
}

/**
 * Parse JSONL content: one JSON object per line. Empty lines are skipped.
 * Multi-line input is always treated as JSONL (line-by-line). Single-line input
 * can be a single JSON object or a JSON array of objects.
 * Returns array of parsed objects.
 */
export function parseJSONL(text) {
  let trimmed = (text && String(text)).trim()
  if (!trimmed) return []
  if (trimmed.charCodeAt(0) === 0xfeff) trimmed = trimmed.slice(1)

  const hasNewline = /\r?\n/.test(trimmed)
  const lines = trimmed.split(/\r?\n/)

  // Multi-line: always treat as JSONL (one JSON object per line)
  if (hasNewline && lines.length > 1) {
    const entries = []
    for (const line of lines) {
      const s = line.trim()
      if (!s) continue
      try {
        entries.push(JSON.parse(s))
      } catch (err) {
        log('JSONL parse error for line:', err.message)
      }
    }
    log('JSONL lines parsed (multi-line):', entries.length)
    if (entries.length > 0) {
      const first = entries[0]
      log('First entry keys:', Object.keys(first).slice(0, 5))
      log('First entry has metrics array:', Array.isArray(first.metrics))
      if (first.metrics?.length) log('First metrics[0]:', first.metrics[0])
    }
    return entries
  }

  // Single line: one JSON object or JSON array
  const single = lines[0]?.trim() || trimmed
  if (!single) return []
  try {
    const parsed = JSON.parse(single)
    if (Array.isArray(parsed)) {
      log('Parsed as single JSON array:', parsed.length)
      return parsed
    }
    if (parsed && typeof parsed === 'object') {
      log('Parsed as single JSON object')
      return [parsed]
    }
  } catch (err) {
    log('Single-line parse error:', err.message)
  }
  return []
}

/**
 * From parsed entries, extract only metrics entries and aggregate for analysis.
 * Supports: (1) OTLP lines with type "metrics" and data.resourceMetrics; (2) flat lines with entry.metrics array.
 */
export function analyzeMetrics(entries) {
  const metricsEntries = []
  for (const e of entries) {
    // Flat format: "metrics" array at top level or under "data"
    let simpleMetrics = e.metrics ?? e.metric
    if (!Array.isArray(simpleMetrics) || simpleMetrics.length === 0) {
      let data = e.data
      if (typeof data === 'string') {
        try {
          data = JSON.parse(data)
        } catch {
          data = null
        }
      }
      if (data && typeof data === 'object' && Array.isArray(data.metrics) && data.metrics.length > 0) {
        simpleMetrics = data.metrics
      }
    }
    if (Array.isArray(simpleMetrics) && simpleMetrics.length > 0) {
      metricsEntries.push({ simple: true, metrics: simpleMetrics, received_at: e.received_at })
      continue
    }
    // OTLP format: type "metrics" and data.resourceMetrics (valid OTel structure)
    if (e.type !== 'metrics') continue
    let data = e.data
    if (typeof data === 'string') {
      try {
        data = JSON.parse(data)
      } catch {
        continue
      }
    }
    if (!data || typeof data !== 'object') continue
    const rms = data.resourceMetrics || data.resource_metrics
    if (!Array.isArray(rms) || rms.length === 0) continue
    metricsEntries.push({ simple: false, data, received_at: e.received_at })
  }

  log('Metrics entries:', metricsEntries.length, metricsEntries.length ? (metricsEntries[0].simple ? 'simple' : 'nested') : '')

  if (!metricsEntries.length) {
    log('No metrics entries found; returning zeros')
    return {
      totalEnergyKwh: 0,
      totalCost: 0,
      avgUtilization: 0,
      avgMemoryUtil: 0,
      avgPowerWatts: 0,
      maxTempC: 0,
      totalNvmlErrors: 0,
      batchCount: 0,
      gpuCount: 0,
      byGpu: {},
      sampleCount: { power: 0, util: 0, memUtil: 0, temp: 0 },
      firstReceivedAt: null,
      lastReceivedAt: null,
      durationSeconds: 0,
      costPerHour: 0,
      energyPerHour: 0,
      projected24hCost: 0,
      projected24hEnergy: 0,
      efficiencyClass: '',
      batches: [],
    }
  }

  const allPower = []
  const allUtil = []
  const allMemUtil = []
  const allTemp = []
  const gpuEnergy = {}
  const gpuCost = {}
  let nvmlErrors = 0
  const gpuIds = new Set()
  const batches = []

  for (const entry of metricsEntries) {
    if (entry.simple) {
      const byName = collectFromSimpleMetrics(entry.metrics)
      const add = (arr, name) => {
        const list = byName[name] || []
        for (const p of list) {
          arr.push(p)
          if (name === 'gpu.energy.kwh.total' || name === 'gpu.cost.total') gpuIds.add(p.gpuId)
        }
      }
      add(allPower, 'gpu.power.watts')
      add(allUtil, 'gpu.utilization.percent')
      add(allMemUtil, 'gpu.memory.utilization.percent')
      add(allTemp, 'gpu.temperature.celsius')
      for (const p of byName['gpu.energy.kwh.total'] || []) {
        gpuIds.add(p.gpuId)
        gpuEnergy[p.gpuId] = p.value
      }
      for (const p of byName['gpu.cost.total'] || []) {
        gpuCost[p.gpuId] = p.value
      }
      const errList = byName['gpu.nvml.errors'] || []
      if (errList.length) nvmlErrors = Math.max(nvmlErrors, ...errList.map((p) => p.value))
      batches.push({
        received_at: entry.received_at,
        byName,
      })
      continue
    }
    const data = entry.data
    allPower.push(...collectDataPoints(data, 'gpu.power.watts'))
    allUtil.push(...collectDataPoints(data, 'gpu.utilization.percent'))
    allMemUtil.push(...collectDataPoints(data, 'gpu.memory.utilization.percent'))
    allTemp.push(...collectDataPoints(data, 'gpu.temperature.celsius'))
    for (const dp of collectDataPoints(data, 'gpu.energy.kwh.total')) {
      gpuIds.add(dp.gpuId)
      gpuEnergy[dp.gpuId] = dp.value
    }
    for (const dp of collectDataPoints(data, 'gpu.cost.total')) {
      gpuCost[dp.gpuId] = dp.value
    }
    const errPoints = collectDataPoints(data, 'gpu.nvml.errors')
    if (errPoints.length) nvmlErrors = Math.max(nvmlErrors, ...errPoints.map((p) => p.value))
    batches.push({
      received_at: entry.received_at,
      byName: dataToByName(data),
    })
  }

  log('Collected dataPoints - power:', allPower.length, 'util:', allUtil.length, 'memUtil:', allMemUtil.length, 'temp:', allTemp.length)
  if (allPower.length) log('Sample power value:', allPower[0])
  log('gpuEnergy:', gpuEnergy, 'gpuCost:', gpuCost, 'gpuIds:', [...gpuIds])

  const sum = (arr) => arr.reduce((a, b) => a + b.value, 0)
  const avg = (arr) => (arr.length ? sum(arr) / arr.length : 0)
  const max = (arr) => (arr.length ? Math.max(...arr.map((p) => p.value)) : 0)

  const totalEnergyKwh = Object.values(gpuEnergy).reduce((a, b) => a + b, 0)
  const totalCost = Object.values(gpuCost).reduce((a, b) => a + b, 0)

  const byGpu = {}
  for (const id of gpuIds) {
    byGpu[id] = {
      energyKwh: gpuEnergy[id] ?? 0,
      cost: gpuCost[id] ?? 0,
    }
  }

  const receivedAts = metricsEntries.map((e) => e.received_at).filter(Boolean)
  let firstReceivedAt = null
  let lastReceivedAt = null
  let durationSeconds = 0
  if (receivedAts.length >= 1) {
    firstReceivedAt = receivedAts[0]
    lastReceivedAt = receivedAts[receivedAts.length - 1]
    const first = new Date(firstReceivedAt).getTime()
    const last = new Date(lastReceivedAt).getTime()
    if (!Number.isNaN(first) && !Number.isNaN(last)) durationSeconds = Math.max(0, (last - first) / 1000)
  }
  const durationHours = durationSeconds / 3600
  const costPerHour = durationHours > 0 ? totalCost / durationHours : 0
  const energyPerHour = durationHours > 0 ? totalEnergyKwh / durationHours : 0
  const projected24hCost = costPerHour * 24
  const projected24hEnergy = energyPerHour * 24

  const avgUtil = avg(allUtil)
  let efficiencyClass = 'Moderate'
  if (avgUtil < 40) efficiencyClass = 'Underutilized'
  else if (avgUtil >= 40 && avgUtil <= 85) efficiencyClass = 'Efficient'
  else if (avgUtil >= 90) efficiencyClass = 'Saturated'

  const result = {
    totalEnergyKwh,
    totalCost,
    avgUtilization: avgUtil,
    avgMemoryUtil: avg(allMemUtil),
    avgPowerWatts: avg(allPower),
    maxTempC: max(allTemp),
    totalNvmlErrors: nvmlErrors,
    batchCount: metricsEntries.length,
    gpuCount: gpuIds.size,
    byGpu,
    sampleCount: {
      power: allPower.length,
      util: allUtil.length,
      memUtil: allMemUtil.length,
      temp: allTemp.length,
    },
    firstReceivedAt,
    lastReceivedAt,
    durationSeconds,
    costPerHour,
    energyPerHour,
    projected24hCost,
    projected24hEnergy,
    efficiencyClass,
    batches,
  }
  log('Analysis result:', result)
  return result
}

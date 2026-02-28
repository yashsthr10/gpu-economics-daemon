import { useState, useCallback } from 'react'
import { parseJSONL, analyzeMetrics } from './utils/parseIngestion'
import MetricsDashboard from './components/MetricsDashboard'
import FileUpload from './components/FileUpload'
import './App.css'

function App() {
  const [analysis, setAnalysis] = useState(null)
  const [error, setError] = useState(null)
  const [fileName, setFileName] = useState(null)

  const handleFile = useCallback((file) => {
    setError(null)
    setAnalysis(null)
    setFileName(file?.name ?? null)
    if (!file) return

    const reader = new FileReader()
    reader.onload = (e) => {
      try {
        const text = e.target?.result
        if (!text || typeof text !== 'string') {
          setError('Could not read file')
          return
        }
        console.log('[App] File loaded, length:', text.length, 'first 80 chars:', text.slice(0, 80))
        const entries = parseJSONL(text)
        console.log('[App] Parsed entries:', entries.length)
        const metrics = analyzeMetrics(entries)
        console.log('[App] Analysis batchCount:', metrics.batchCount, 'gpuCount:', metrics.gpuCount)
        setAnalysis(metrics)
      } catch (err) {
        console.error('[App] Parse error:', err)
        setError(err.message || 'Parse error')
      }
    }
    reader.onerror = () => setError('Failed to read file')
    reader.readAsText(file, 'utf-8')
  }, [])

  return (
    <div className="app">
      <header className="app-header">
        <h1>GPU Economics — Ingestion Analysis</h1>
        <p className="subtitle">Upload JSONL from the demo ingestion service (e.g. otel_ingestion.jsonl)</p>
      </header>

      <FileUpload onFile={handleFile} fileName={fileName} />

      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      {analysis && <MetricsDashboard analysis={analysis} />}
    </div>
  )
}

export default App

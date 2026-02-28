import { useRef } from 'react'

/**
 * File upload zone for ingestion file. Accepts JSONL (.jsonl), JSON (.json), and plain text.
 */
export default function FileUpload({ onFile, fileName }) {
  const inputRef = useRef(null)

  const handleChange = (e) => {
    const file = e.target.files?.[0]
    onFile(file ?? null)
  }

  const handleDrop = (e) => {
    e.preventDefault()
    const file = e.dataTransfer?.files?.[0]
    if (file) onFile(file)
  }

  const handleDragOver = (e) => e.preventDefault()

  return (
    <div
      className="upload-zone"
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      onClick={() => inputRef.current?.click()}
    >
      <input
        ref={inputRef}
        type="file"
        accept=".jsonl,.ndjson,application/jsonl,application/x-ndjson,.json,application/json,text/plain"
        onChange={handleChange}
        className="upload-input"
      />
      {fileName ? (
        <span className="upload-name">{fileName}</span>
      ) : (
        <span className="upload-placeholder">Drop OTLP or flat JSONL (e.g. otel_ingestion.jsonl) here or click to browse</span>
      )}
    </div>
  )
}

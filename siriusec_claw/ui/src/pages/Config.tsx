import { useEffect, useState } from 'react'
import type { RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

export default function ConfigPage({ call }: Props) {
  const [config, setConfig] = useState<any>(null)
  const [raw, setRaw] = useState('')
  const [editing, setEditing] = useState(false)
  const [status, setStatus] = useState('')

  const load = () => {
    call('config.get').then(r => {
      if (r.ok) {
        setConfig(r.payload)
        setRaw(r.payload?.raw || JSON.stringify(r.payload?.parsed || {}, null, 2))
      }
    })
  }

  useEffect(() => { load() }, [call])

  const handleSave = async () => {
    try {
      const parsed = JSON.parse(raw)
      const res = await call('config.set', { config: parsed })
      if (res.ok) {
        setStatus('Saved successfully')
        setEditing(false)
        load()
      } else {
        setStatus('Error: ' + (res.error?.message || 'unknown'))
      }
    } catch (e) {
      setStatus('Invalid JSON')
    }
    setTimeout(() => setStatus(''), 3000)
  }

  return (
    <div>
      <div className="flex gap-12 mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Configuration</h1>
        <div className="ml-auto btn-group">
          <button className="btn btn-sm" onClick={load}>Refresh</button>
          {!editing ? (
            <button className="btn btn-sm btn-primary" onClick={() => setEditing(true)}>Edit</button>
          ) : (
            <>
              <button className="btn btn-sm btn-primary" onClick={handleSave}>Save</button>
              <button className="btn btn-sm" onClick={() => { setEditing(false); load() }}>Cancel</button>
            </>
          )}
        </div>
      </div>

      {status && (
        <div className="card mb-16" style={{ borderColor: status.startsWith('Error') || status === 'Invalid JSON' ? 'var(--red)' : 'var(--green)' }}>
          {status}
        </div>
      )}

      <div className="card mb-16">
        <div className="card-title">Config Info</div>
        <table className="table">
          <tbody>
            <tr><td style={{ width: 150 }}>Path</td><td className="mono">{config?.path || '-'}</td></tr>
            <tr><td>Exists</td><td>{config?.exists ? 'Yes' : 'No'}</td></tr>
            <tr><td>Hash</td><td className="mono">{config?.hash || '-'}</td></tr>
          </tbody>
        </table>
      </div>

      <div className="card">
        <div className="card-title">Config Content</div>
        {editing ? (
          <textarea
            className="textarea"
            style={{ minHeight: 400, fontFamily: 'var(--font)' }}
            value={raw}
            onChange={e => setRaw(e.target.value)}
          />
        ) : (
          <pre className="code-block">{raw || '(empty)'}</pre>
        )}
      </div>
    </div>
  )
}

import { useEffect, useState } from 'react'
import type { RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

export default function Sessions({ call }: Props) {
  const [sessions, setSessions] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  const load = () => {
    setLoading(true)
    call('sessions.list').then(r => {
      if (r.ok) setSessions(r.payload?.sessions || [])
      setLoading(false)
    })
  }

  useEffect(() => { load() }, [call])

  const handleDelete = async (key: string) => {
    if (!confirm(`Delete session ${key}?`)) return
    await call('sessions.delete', { sessionKey: key })
    load()
  }

  const handleReset = async (key: string) => {
    await call('sessions.reset', { sessionKey: key })
    load()
  }

  return (
    <div>
      <div className="flex gap-12 mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Sessions</h1>
        <button className="btn btn-sm ml-auto" onClick={load}>Refresh</button>
      </div>

      {loading ? (
        <div className="empty">Loading...</div>
      ) : sessions.length === 0 ? (
        <div className="empty">No sessions</div>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Session Key</th>
              <th>Agent</th>
              <th>Messages</th>
              <th>Updated</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {sessions.map((s: any) => (
              <tr key={s.sessionKey || s.id}>
                <td className="mono">{s.sessionKey || s.id}</td>
                <td>{s.agentId || 'main'}</td>
                <td>{s.messageCount ?? '-'}</td>
                <td className="mono">
                  {s.updatedAt ? new Date(s.updatedAt).toLocaleString() : '-'}
                </td>
                <td>
                  <div className="btn-group">
                    <button className="btn btn-sm" onClick={() => handleReset(s.sessionKey || s.id)}>
                      Reset
                    </button>
                    <button className="btn btn-sm btn-danger" onClick={() => handleDelete(s.sessionKey || s.id)}>
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

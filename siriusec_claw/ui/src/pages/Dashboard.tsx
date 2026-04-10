import { useEffect, useState } from 'react'
import type { RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

export default function Dashboard({ call }: Props) {
  const [health, setHealth] = useState<any>(null)
  const [sessions, setSessions] = useState<any[]>([])
  const [agents, setAgents] = useState<any[]>([])
  const [cronJobs, setCronJobs] = useState<any[]>([])

  useEffect(() => {
    call('health').then(r => r.ok && setHealth(r.payload))
    call('sessions.list').then(r => r.ok && setSessions(r.payload?.sessions || []))
    call('agents.list').then(r => r.ok && setAgents(r.payload?.agents || []))
    call('cron.list').then(r => r.ok && setCronJobs(r.payload?.jobs || []))
  }, [call])

  return (
    <div>
      <h1 className="page-title">Dashboard</h1>
      <div className="stat-grid">
        <div className="stat-card">
          <div className="stat-label">Version</div>
          <div className="stat-value mono">{health?.version || '-'}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Sessions</div>
          <div className="stat-value">{sessions.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Agents</div>
          <div className="stat-value">{agents.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Cron Jobs</div>
          <div className="stat-value">{cronJobs.length}</div>
        </div>
      </div>

      <div className="card">
        <div className="card-title">Recent Sessions</div>
        {sessions.length === 0 ? (
          <div className="empty">No sessions yet</div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Session Key</th>
                <th>Agent</th>
                <th>Updated</th>
              </tr>
            </thead>
            <tbody>
              {sessions.slice(0, 10).map((s: any) => (
                <tr key={s.sessionKey || s.id}>
                  <td className="mono">{s.sessionKey || s.id}</td>
                  <td>{s.agentId || 'main'}</td>
                  <td className="mono">{s.updatedAt ? new Date(s.updatedAt).toLocaleString() : '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

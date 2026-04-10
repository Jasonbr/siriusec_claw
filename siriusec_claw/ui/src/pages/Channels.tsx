import { useEffect, useState } from 'react'
import type { RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

export default function Channels({ call }: Props) {
  const [channels, setChannels] = useState<any[]>([])
  const [approvals, setApprovals] = useState<any[]>([])

  const load = () => {
    call('channels.status').then(r => r.ok && setChannels(r.payload?.channels || []))
    call('approvals.list').then(r => r.ok && setApprovals(r.payload?.approvals || []))
  }

  useEffect(() => { load() }, [call])

  const handleLogout = async (id: string) => {
    await call('channels.logout', { channelId: id })
    load()
  }

  const handleApprove = async (id: string) => {
    await call('approvals.approve', { id })
    load()
  }

  const handleDeny = async (id: string) => {
    await call('approvals.deny', { id, reason: 'denied via UI' })
    load()
  }

  return (
    <div>
      <div className="flex gap-12 mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Channels & Approvals</h1>
        <button className="btn btn-sm ml-auto" onClick={load}>Refresh</button>
      </div>

      <div className="card mb-16">
        <div className="card-title">Channels</div>
        {channels.length === 0 ? (
          <div className="empty">No channels registered</div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Channel</th>
                <th>Status</th>
                <th>Enabled</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {channels.map((c: any) => (
                <tr key={c.id}>
                  <td>{c.displayName || c.id}</td>
                  <td>
                    <span className={`badge ${c.connected ? 'badge-green' : 'badge-red'}`}>
                      {c.connected ? 'connected' : 'disconnected'}
                    </span>
                  </td>
                  <td>
                    <span className={`badge ${c.enabled ? 'badge-green' : 'badge-yellow'}`}>
                      {c.enabled ? 'yes' : 'no'}
                    </span>
                  </td>
                  <td>
                    <button className="btn btn-sm" onClick={() => handleLogout(c.id)}>Logout</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="card">
        <div className="card-title">Pending Approvals</div>
        {approvals.length === 0 ? (
          <div className="empty">No pending approvals</div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Command</th>
                <th>Session</th>
                <th>Tool</th>
                <th>Requested</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {approvals.map((a: any) => (
                <tr key={a.id}>
                  <td className="mono" style={{ maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {a.command}
                  </td>
                  <td className="mono">{a.sessionKey}</td>
                  <td>{a.tool || '-'}</td>
                  <td className="mono">{new Date(a.requestedAt).toLocaleString()}</td>
                  <td>
                    <div className="btn-group">
                      <button className="btn btn-sm btn-primary" onClick={() => handleApprove(a.id)}>Approve</button>
                      <button className="btn btn-sm btn-danger" onClick={() => handleDeny(a.id)}>Deny</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

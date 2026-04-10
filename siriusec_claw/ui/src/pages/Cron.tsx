import { useEffect, useState } from 'react'
import type { RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

export default function CronPage({ call }: Props) {
  const [jobs, setJobs] = useState<any[]>([])
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({ name: '', schedule: '*/5 * * * *', message: '' })

  const load = () => {
    call('cron.list').then(r => r.ok && setJobs(r.payload?.jobs || []))
  }

  useEffect(() => { load() }, [call])

  const handleAdd = async () => {
    if (!form.name.trim() || !form.message.trim()) return
    await call('cron.add', form)
    setForm({ name: '', schedule: '*/5 * * * *', message: '' })
    setShowAdd(false)
    load()
  }

  const handleRemove = async (id: string) => {
    if (!confirm('Remove this job?')) return
    await call('cron.remove', { id })
    load()
  }

  const handleRun = async (id: string) => {
    await call('cron.run', { id })
  }

  const handleToggle = async (job: any) => {
    await call('cron.update', { id: job.id, enabled: !job.enabled })
    load()
  }

  return (
    <div>
      <div className="flex gap-12 mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Cron Jobs</h1>
        <div className="ml-auto btn-group">
          <button className="btn btn-sm" onClick={load}>Refresh</button>
          <button className="btn btn-sm btn-primary" onClick={() => setShowAdd(!showAdd)}>
            {showAdd ? 'Cancel' : '+ Add Job'}
          </button>
        </div>
      </div>

      {showAdd && (
        <div className="card mb-16">
          <div className="card-title">Add Cron Job</div>
          <div className="form-group">
            <label className="form-label">Name</label>
            <input className="input" value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} placeholder="Job name" />
          </div>
          <div className="form-group">
            <label className="form-label">Schedule (cron)</label>
            <input className="input mono" value={form.schedule} onChange={e => setForm({ ...form, schedule: e.target.value })} />
          </div>
          <div className="form-group">
            <label className="form-label">Message</label>
            <textarea className="textarea" value={form.message} onChange={e => setForm({ ...form, message: e.target.value })} placeholder="Message to send..." />
          </div>
          <button className="btn btn-primary" onClick={handleAdd}>Add Job</button>
        </div>
      )}

      {jobs.length === 0 ? (
        <div className="empty">No cron jobs</div>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Schedule</th>
              <th>Status</th>
              <th>Runs</th>
              <th>Last Run</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((j: any) => (
              <tr key={j.id}>
                <td>{j.name || j.id}</td>
                <td className="mono">{j.schedule}</td>
                <td>
                  <span className={`badge ${j.enabled ? 'badge-green' : 'badge-yellow'}`}>
                    {j.enabled ? 'enabled' : 'disabled'}
                  </span>
                </td>
                <td>{j.runCount ?? 0}</td>
                <td className="mono">{j.lastRunAt ? new Date(j.lastRunAt).toLocaleString() : 'never'}</td>
                <td>
                  <div className="btn-group">
                    <button className="btn btn-sm" onClick={() => handleToggle(j)}>
                      {j.enabled ? 'Disable' : 'Enable'}
                    </button>
                    <button className="btn btn-sm btn-primary" onClick={() => handleRun(j.id)}>Run</button>
                    <button className="btn btn-sm btn-danger" onClick={() => handleRemove(j.id)}>Remove</button>
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

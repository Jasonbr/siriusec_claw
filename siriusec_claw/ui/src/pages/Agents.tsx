import { useEffect, useState } from 'react'
import type { RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

export default function Agents({ call }: Props) {
  const [agents, setAgents] = useState<any[]>([])
  const [models, setModels] = useState<any[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [newPrompt, setNewPrompt] = useState('')

  const load = () => {
    call('agents.list').then(r => r.ok && setAgents(r.payload?.agents || []))
    call('models.list').then(r => r.ok && setModels(r.payload?.models || []))
  }

  useEffect(() => { load() }, [call])

  const handleCreate = async () => {
    if (!newName.trim()) return
    await call('agents.create', { name: newName, prompt: newPrompt })
    setNewName('')
    setNewPrompt('')
    setShowCreate(false)
    load()
  }

  const handleDelete = async (id: string) => {
    if (!confirm(`Delete agent ${id}?`)) return
    await call('agents.delete', { agentId: id })
    load()
  }

  return (
    <div>
      <div className="flex gap-12 mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Agents</h1>
        <div className="ml-auto btn-group">
          <button className="btn btn-sm" onClick={load}>Refresh</button>
          <button className="btn btn-sm btn-primary" onClick={() => setShowCreate(!showCreate)}>
            {showCreate ? 'Cancel' : '+ New Agent'}
          </button>
        </div>
      </div>

      {showCreate && (
        <div className="card mb-16">
          <div className="card-title">Create Agent</div>
          <div className="form-group">
            <label className="form-label">Name</label>
            <input className="input" value={newName} onChange={e => setNewName(e.target.value)} placeholder="Agent name" />
          </div>
          <div className="form-group">
            <label className="form-label">System Prompt</label>
            <textarea className="textarea" value={newPrompt} onChange={e => setNewPrompt(e.target.value)} placeholder="Agent system prompt..." />
          </div>
          <button className="btn btn-primary" onClick={handleCreate}>Create</button>
        </div>
      )}

      {agents.length === 0 ? (
        <div className="empty">No agents configured</div>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Name</th>
              <th>Model</th>
              <th>Type</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {agents.map((a: any) => (
              <tr key={a.id || a.name}>
                <td className="mono">{a.id || '-'}</td>
                <td>{a.name || a.id || '-'}</td>
                <td className="mono">{a.model || a.modelRef || '-'}</td>
                <td>
                  <span className={`badge ${a.isMain ? 'badge-blue' : 'badge-green'}`}>
                    {a.isMain ? 'main' : 'custom'}
                  </span>
                </td>
                <td>
                  {!a.isMain && (
                    <button className="btn btn-sm btn-danger" onClick={() => handleDelete(a.id)}>
                      Delete
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div className="card" style={{ marginTop: 24 }}>
        <div className="card-title">Available Models</div>
        {models.length === 0 ? (
          <div className="empty">No models configured</div>
        ) : (
          <table className="table">
            <thead>
              <tr><th>Provider</th><th>Model</th><th>Ref</th></tr>
            </thead>
            <tbody>
              {models.map((m: any, i: number) => (
                <tr key={i}>
                  <td>{m.provider || '-'}</td>
                  <td>{m.name || m.modelId || '-'}</td>
                  <td className="mono">{m.ref || `${m.provider}/${m.modelId}` || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

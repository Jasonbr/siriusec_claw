import { useEffect, useState } from 'react'
import type { RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

export default function MemoryPage({ call }: Props) {
  const [entries, setEntries] = useState<any[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<any[] | null>(null)
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({ title: '', content: '', category: 'note', tags: '' })

  const load = () => {
    call('memory.list').then(r => r.ok && setEntries(r.payload?.entries || []))
    setSearchResults(null)
  }

  useEffect(() => { load() }, [call])

  const handleSearch = async () => {
    if (!searchQuery.trim()) { setSearchResults(null); return }
    const r = await call('memory.search', { query: searchQuery, limit: 20 })
    if (r.ok) setSearchResults(r.payload?.results || [])
  }

  const handleAdd = async () => {
    if (!form.title.trim() && !form.content.trim()) return
    const tags = form.tags.split(',').map(t => t.trim()).filter(Boolean)
    await call('memory.add', { ...form, tags })
    setForm({ title: '', content: '', category: 'note', tags: '' })
    setShowAdd(false)
    load()
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this entry?')) return
    await call('memory.delete', { id })
    load()
  }

  const displayEntries = searchResults
    ? searchResults.map((r: any) => ({ ...r.entry, _score: r.score }))
    : entries

  return (
    <div>
      <div className="flex gap-12 mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Memory</h1>
        <div className="ml-auto btn-group">
          <button className="btn btn-sm" onClick={load}>Refresh</button>
          <button className="btn btn-sm btn-primary" onClick={() => setShowAdd(!showAdd)}>
            {showAdd ? 'Cancel' : '+ Add Entry'}
          </button>
        </div>
      </div>

      <div className="flex gap-8 mb-16">
        <input
          className="input"
          style={{ maxWidth: 400 }}
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && handleSearch()}
          placeholder="Search memory..."
        />
        <button className="btn btn-sm" onClick={handleSearch}>Search</button>
        {searchResults && (
          <button className="btn btn-sm" onClick={() => { setSearchResults(null); setSearchQuery('') }}>Clear</button>
        )}
      </div>

      {showAdd && (
        <div className="card mb-16">
          <div className="card-title">Add Memory Entry</div>
          <div className="form-group">
            <label className="form-label">Title</label>
            <input className="input" value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} />
          </div>
          <div className="form-group">
            <label className="form-label">Content</label>
            <textarea className="textarea" value={form.content} onChange={e => setForm({ ...form, content: e.target.value })} />
          </div>
          <div className="flex gap-8">
            <div className="form-group" style={{ flex: 1 }}>
              <label className="form-label">Category</label>
              <select className="input" value={form.category} onChange={e => setForm({ ...form, category: e.target.value })}>
                <option value="note">Note</option>
                <option value="snippet">Snippet</option>
                <option value="reference">Reference</option>
                <option value="context">Context</option>
              </select>
            </div>
            <div className="form-group" style={{ flex: 2 }}>
              <label className="form-label">Tags (comma separated)</label>
              <input className="input" value={form.tags} onChange={e => setForm({ ...form, tags: e.target.value })} placeholder="tag1, tag2" />
            </div>
          </div>
          <button className="btn btn-primary" onClick={handleAdd}>Add</button>
        </div>
      )}

      {displayEntries.length === 0 ? (
        <div className="empty">{searchResults ? 'No results found' : 'No memory entries'}</div>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Title</th>
              <th>Category</th>
              <th>Tags</th>
              {searchResults && <th>Score</th>}
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {displayEntries.map((e: any) => (
              <tr key={e.id}>
                <td>{e.title || '(untitled)'}</td>
                <td><span className="badge badge-blue">{e.category || '-'}</span></td>
                <td className="mono">{(e.tags || []).join(', ') || '-'}</td>
                {searchResults && <td className="mono">{(e._score || 0).toFixed(2)}</td>}
                <td className="mono">{e.createdAt ? new Date(e.createdAt).toLocaleString() : '-'}</td>
                <td>
                  <button className="btn btn-sm btn-danger" onClick={() => handleDelete(e.id)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

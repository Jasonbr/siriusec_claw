import { useEffect, useState } from 'react'
import type { RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

type SkillItem = {
  id: string
  name: string
  emoji?: string
  description: string
  author?: string
  tags?: string[]
  version?: string
  downloads?: number
  rating?: number
  installed?: boolean
}

type MCPItem = {
  id: string
  name: string
  emoji?: string
  description: string
  author?: string
  tags?: string[]
  version?: string
  downloads?: number
  rating?: number
  transport?: string
  installed?: boolean
}

type EmployeeItem = {
  id: string
  name: string
  emoji?: string
  description: string
  author?: string
  tags?: string[]
  version?: string
  downloads?: number
  rating?: number
  installed?: boolean
}

type Tab = 'skills' | 'mcps' | 'employees'

export default function Marketplace({ call }: Props) {
  const [tab, setTab] = useState<Tab>('skills')
  const [loading, setLoading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [skills, setSkills] = useState<SkillItem[]>([])
  const [mcps, setMcps] = useState<MCPItem[]>([])
  const [employees, setEmployees] = useState<EmployeeItem[]>([])
  const [installing, setInstalling] = useState<string | null>(null)

  const loadSkills = async () => {
    setLoading(true)
    const r = await call('marketplace.skills', { query: searchQuery })
    setLoading(false)
    if (r.ok) {
      setSkills(r.payload?.skills || [])
    }
  }

  const loadMCPs = async () => {
    setLoading(true)
    const r = await call('marketplace.mcps', { query: searchQuery })
    setLoading(false)
    if (r.ok) {
      setMcps(r.payload?.mcpServers || [])
    }
  }

  const loadEmployees = async () => {
    setLoading(true)
    const r = await call('marketplace.employees', { query: searchQuery })
    setLoading(false)
    if (r.ok) {
      setEmployees(r.payload?.employees || [])
    }
  }

  useEffect(() => {
    if (tab === 'skills') loadSkills()
    else if (tab === 'mcps') loadMCPs()
    else if (tab === 'employees') loadEmployees()
  }, [tab])

  const handleSearch = () => {
    if (tab === 'skills') loadSkills()
    else if (tab === 'mcps') loadMCPs()
    else if (tab === 'employees') loadEmployees()
  }

  const handleInstall = async (type: Tab, id: string) => {
    setInstalling(id)
    let r: RPCResponse
    if (type === 'skills') {
      r = await call('marketplace.skill.install', { id })
    } else if (type === 'mcps') {
      r = await call('marketplace.mcp.install', { id })
    } else {
      r = await call('marketplace.employee.install', { id })
    }
    setInstalling(null)
    if (r.ok) {
      alert(r.payload?.message || 'Installed successfully!')
      // Refresh the list
      if (type === 'skills') loadSkills()
      else if (type === 'mcps') loadMCPs()
      else loadEmployees()
    } else {
      alert('Install failed: ' + (r.error?.message || 'Unknown error'))
    }
  }

  return (
    <div>
      <div className="flex gap-12 mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Marketplace</h1>
        <div className="ml-auto btn-group">
          <input
            type="text"
            className="input input-sm"
            placeholder="Search..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleSearch()}
            style={{ width: 200 }}
          />
          <button className="btn btn-sm" onClick={handleSearch} disabled={loading}>
            Search
          </button>
        </div>
      </div>

      <div className="tabs mb-16">
        <button
          className={`tab ${tab === 'skills' ? 'active' : ''}`}
          onClick={() => setTab('skills')}
        >
          📦 Skills
        </button>
        <button
          className={`tab ${tab === 'mcps' ? 'active' : ''}`}
          onClick={() => setTab('mcps')}
        >
          🔌 MCP Servers
        </button>
        <button
          className={`tab ${tab === 'employees' ? 'active' : ''}`}
          onClick={() => setTab('employees')}
        >
          👥 Employees
        </button>
      </div>

      {loading ? (
        <div className="empty">Loading...</div>
      ) : (
        <>
          {tab === 'skills' && (
            <div className="marketplace-grid">
              {skills.length === 0 ? (
                <div className="empty">No skills found</div>
              ) : (
                skills.map(item => (
                  <div key={item.id} className="marketplace-card">
                    <div className="marketplace-card-header">
                      <span className="marketplace-icon">{item.emoji || '📦'}</span>
                      <div className="marketplace-title">
                        <div className="marketplace-name">{item.name}</div>
                        {item.author && <div className="marketplace-author">by {item.author}</div>}
                      </div>
                    </div>
                    <div className="marketplace-card-body">
                      <p className="marketplace-desc">{item.description}</p>
                      {item.tags && item.tags.length > 0 && (
                        <div className="marketplace-tags">
                          {item.tags.slice(0, 3).map(tag => (
                            <span key={tag} className="badge badge-gray">{tag}</span>
                          ))}
                        </div>
                      )}
                    </div>
                    <div className="marketplace-card-footer">
                      <div className="marketplace-stats">
                        {item.downloads !== undefined && <span>⬇️ {item.downloads}</span>}
                        {item.rating !== undefined && <span>⭐ {item.rating.toFixed(1)}</span>}
                      </div>
                      <button
                        className={`btn btn-sm ${item.installed ? '' : 'btn-primary'}`}
                        onClick={() => handleInstall('skills', item.id)}
                        disabled={installing === item.id || item.installed}
                      >
                        {installing === item.id ? 'Installing...' : item.installed ? 'Installed' : 'Install'}
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}

          {tab === 'mcps' && (
            <div className="marketplace-grid">
              {mcps.length === 0 ? (
                <div className="empty">No MCP servers found</div>
              ) : (
                mcps.map(item => (
                  <div key={item.id} className="marketplace-card">
                    <div className="marketplace-card-header">
                      <span className="marketplace-icon">{item.emoji || '🔌'}</span>
                      <div className="marketplace-title">
                        <div className="marketplace-name">{item.name}</div>
                        {item.author && <div className="marketplace-author">by {item.author}</div>}
                      </div>
                      {item.transport && (
                        <span className="badge badge-blue">{item.transport}</span>
                      )}
                    </div>
                    <div className="marketplace-card-body">
                      <p className="marketplace-desc">{item.description}</p>
                      {item.tags && item.tags.length > 0 && (
                        <div className="marketplace-tags">
                          {item.tags.slice(0, 3).map(tag => (
                            <span key={tag} className="badge badge-gray">{tag}</span>
                          ))}
                        </div>
                      )}
                    </div>
                    <div className="marketplace-card-footer">
                      <div className="marketplace-stats">
                        {item.downloads !== undefined && <span>⬇️ {item.downloads}</span>}
                        {item.rating !== undefined && <span>⭐ {item.rating.toFixed(1)}</span>}
                      </div>
                      <button
                        className={`btn btn-sm ${item.installed ? '' : 'btn-primary'}`}
                        onClick={() => handleInstall('mcps', item.id)}
                        disabled={installing === item.id || item.installed}
                      >
                        {installing === item.id ? 'Installing...' : item.installed ? 'Installed' : 'Install'}
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}

          {tab === 'employees' && (
            <div className="marketplace-grid">
              {employees.length === 0 ? (
                <div className="empty">No employees found</div>
              ) : (
                employees.map(item => (
                  <div key={item.id} className="marketplace-card">
                    <div className="marketplace-card-header">
                      <span className="marketplace-icon">{item.emoji || '👥'}</span>
                      <div className="marketplace-title">
                        <div className="marketplace-name">{item.name}</div>
                        {item.author && <div className="marketplace-author">by {item.author}</div>}
                      </div>
                    </div>
                    <div className="marketplace-card-body">
                      <p className="marketplace-desc">{item.description}</p>
                      {item.tags && item.tags.length > 0 && (
                        <div className="marketplace-tags">
                          {item.tags.slice(0, 3).map(tag => (
                            <span key={tag} className="badge badge-gray">{tag}</span>
                          ))}
                        </div>
                      )}
                    </div>
                    <div className="marketplace-card-footer">
                      <div className="marketplace-stats">
                        {item.downloads !== undefined && <span>⬇️ {item.downloads}</span>}
                        {item.rating !== undefined && <span>⭐ {item.rating.toFixed(1)}</span>}
                      </div>
                      <button
                        className={`btn btn-sm ${item.installed ? '' : 'btn-primary'}`}
                        onClick={() => handleInstall('employees', item.id)}
                        disabled={installing === item.id || item.installed}
                      >
                        {installing === item.id ? 'Installing...' : item.installed ? 'Installed' : 'Install'}
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </>
      )}

      <div className="card" style={{ marginTop: 24 }}>
        <div className="card-title">About Marketplace</div>
        <p className="text-secondary">
          Browse and install skills, MCP servers, and employee templates from OpenOcta marketplace.
          Connect your systems, APIs, and tools with pre-built integrations.
        </p>
        <ul className="list">
          <li><strong>Skills:</strong> Pre-built AI capabilities and behaviors</li>
          <li><strong>MCP Servers:</strong> Model Context Protocol servers for tool integration</li>
          <li><strong>Employees:</strong> Pre-configured AI agents for specific tasks</li>
        </ul>
      </div>
    </div>
  )
}

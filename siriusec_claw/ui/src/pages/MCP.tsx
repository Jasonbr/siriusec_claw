import { useState, useEffect, useRef } from 'react'

interface MCPServer {
  name: string
  enabled: boolean
  transport?: string
  command?: string
  args?: string[]
  url?: string
  service?: string
  toolPrefix?: string
  description?: string
  author?: string
  source?: string
  installedAt?: number
  env?: Record<string, string>
}

interface Props {
  call: (method: string, params?: any) => Promise<any>
}

// JSON Editor Component
function JsonEditor({
  value,
  onChange,
  placeholder,
  label,
  hint
}: {
  value: string
  onChange: (value: string, isValid: boolean) => void
  placeholder?: string
  label?: string
  hint?: string
}) {
  const [error, setError] = useState<string | null>(null)

  const validate = (text: string): boolean => {
    if (!text.trim()) return true
    try {
      JSON.parse(text)
      setError(null)
      return true
    } catch (e) {
      setError((e as Error).message)
      return false
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const text = e.target.value
    const isValid = validate(text)
    onChange(text, isValid)
  }

  return (
    <div className="form-group">
      {label && <label>{label} <span className="optional">(JSON format)</span></label>}
      <textarea
        className={`json-editor ${error ? 'error' : ''}`}
        value={value}
        onChange={handleChange}
        placeholder={placeholder || '{\n  "KEY": "value"\n}'}
        rows={6}
      />
      <div className="json-editor-hint">
        {error ? (
          <span className="invalid">❌ {error}</span>
        ) : (
          <span className="valid">✓ Valid JSON</span>
        )}
      </div>
      {hint && <div className="form-hint">{hint}</div>}
    </div>
  )
}

export default function MCPPage({ call }: Props) {
  const [servers, setServers] = useState<MCPServer[]>([])
  const [loading, setLoading] = useState(false)
  const [showInstall, setShowInstall] = useState(false)
  const [installTab, setInstallTab] = useState<'github' | 'upload' | 'zip' | 'manual'>('github')
  const [installing, setInstalling] = useState(false)

  // GitHub install
  const [githubUrl, setGithubUrl] = useState('')

  // File upload
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadName, setUploadName] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Manual form
  const [formData, setFormData] = useState<Partial<MCPServer>>({
    name: '',
    transport: 'stdio',
    command: '',
    args: [],
    url: '',
    enabled: true,
  })
  const [envJson, setEnvJson] = useState('')
  const [envValid, setEnvValid] = useState(true)

  useEffect(() => {
    loadServers()
  }, [])

  const loadServers = async () => {
    setLoading(true)
    try {
      const res = await call('mcp.list')
      if (res.ok) {
        setServers(res.payload?.servers || [])
      }
    } catch (err) {
      console.error('Failed to load MCP servers:', err)
    }
    setLoading(false)
  }

  const handleInstallFromGitHub = async () => {
    if (!githubUrl.trim()) return
    setInstalling(true)
    try {
      const res = await call('mcp.install', { source: 'github', url: githubUrl.trim() })
      setInstalling(false)
      if (res.ok) {
        setGithubUrl('')
        setShowInstall(false)
        loadServers()
      } else {
        alert('Install failed: ' + (res.error?.message || 'Unknown error'))
      }
    } catch (err) {
      setInstalling(false)
      alert('Install failed: ' + err)
    }
  }

  const handleUpload = async () => {
    if (!uploadFile) return
    setInstalling(true)

    try {
      // Check if it's a zip file
      if (uploadFile.name.toLowerCase().endsWith('.zip')) {
        const buffer = await uploadFile.arrayBuffer()
        const base64 = btoa(String.fromCharCode(...new Uint8Array(buffer)))
        const res = await call('mcp.install', {
          source: 'zip',
          name: uploadName.trim() || undefined,
          content: base64
        })
        setInstalling(false)
        if (res.ok) {
          setUploadFile(null)
          setUploadName('')
          setShowInstall(false)
          loadServers()
        } else {
          alert('Upload failed: ' + (res.error?.message || 'Unknown error'))
        }
        return
      }

      // Handle .md file
      const content = await uploadFile.text()
      const base64 = btoa(unescape(encodeURIComponent(content)))
      const res = await call('mcp.install', {
        source: 'upload',
        name: uploadName.trim() || undefined,
        content: base64
      })
      setInstalling(false)
      if (res.ok) {
        setUploadFile(null)
        setUploadName('')
        setShowInstall(false)
        loadServers()
      } else {
        alert('Upload failed: ' + (res.error?.message || 'Unknown error'))
      }
    } catch (err) {
      setInstalling(false)
      alert('Upload failed: ' + err)
    }
  }

  const handleManualAdd = async () => {
    if (!formData.name) return
    if (!envValid) {
      alert('Please fix the environment variables JSON format')
      return
    }

    try {
      const params: any = {
        name: formData.name,
        transport: formData.transport,
        enabled: formData.enabled,
      }

      if (formData.transport === 'stdio') {
        params.command = formData.command
        params.args = formData.args?.filter(Boolean) || []
      } else {
        params.url = formData.url
      }

      // Add environment variables if any
      if (envJson.trim()) {
        params.env = JSON.parse(envJson)
      }

      const res = await call('mcp.add', params)
      if (res.ok) {
        setShowInstall(false)
        setFormData({ name: '', transport: 'stdio', command: '', args: [], url: '', enabled: true })
        setEnvJson('')
        loadServers()
      } else {
        alert('Failed to add MCP server: ' + (res.error?.message || 'Unknown error'))
      }
    } catch (err) {
      alert('Failed to add MCP server: ' + err)
    }
  }

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete MCP server "${name}"?`)) return
    try {
      const res = await call('mcp.delete', { name })
      if (res.ok) {
        loadServers()
      } else {
        alert('Failed to delete: ' + (res.error?.message || 'Unknown error'))
      }
    } catch (err) {
      alert('Failed to delete: ' + err)
    }
  }

  const handleToggle = async (server: MCPServer) => {
    try {
      const res = await call('mcp.update', {
        name: server.name,
        enabled: !server.enabled
      })
      if (res.ok) {
        loadServers()
      } else {
        alert('Failed to toggle: ' + (res.error?.message || 'Unknown error'))
      }
    } catch (err) {
      alert('Failed to toggle: ' + err)
    }
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setUploadFile(file)
      if (!uploadName) {
        const baseName = file.name.replace(/\.(md|zip)$/i, '')
        setUploadName(baseName)
      }
    }
  }

  const resetForm = () => {
    setShowInstall(false)
    setInstallTab('github')
    setGithubUrl('')
    setUploadFile(null)
    setUploadName('')
    setFormData({ name: '', transport: 'stdio', command: '', args: [], url: '', enabled: true })
    setEnvJson('')
    setEnvValid(true)
  }

  return (
    <div className="page">
      <div className="page-header">
        <h2>MCP Servers</h2>
        <button className="btn btn-primary" onClick={() => setShowInstall(true)}>
          + Add Server
        </button>
      </div>

      {loading ? (
        <div className="loading">Loading...</div>
      ) : servers.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">🔌</div>
          <div className="empty-state-title">No MCP servers configured</div>
          <div className="empty-state-desc">Add your first MCP server to enable external tool capabilities</div>
          <button className="btn btn-primary" onClick={() => setShowInstall(true)}>
            Add Your First MCP Server
          </button>
        </div>
      ) : (
        <div className="mcp-list">
          {servers.map(server => (
            <div key={server.name} className={`mcp-card ${server.enabled ? '' : 'disabled'}`}>
              <div className="mcp-card-header">
                <div>
                  <div className="mcp-name">
                    {server.name}
                    {server.toolPrefix && <span className="tool-prefix">({server.toolPrefix})</span>}
                  </div>
                  {server.description && (
                    <div className="mcp-card-body" style={{ marginTop: 4 }}>{server.description}</div>
                  )}
                </div>
                <label className="toggle">
                  <input
                    type="checkbox"
                    checked={server.enabled}
                    onChange={() => handleToggle(server)}
                  />
                  <span className="toggle-slider"></span>
                </label>
              </div>
              <div className="mcp-meta">
                <span className="transport">{server.transport}</span>
                {server.command && (
                  <code className="command">{server.command} {server.args?.join(' ')}</code>
                )}
                {server.url && <code className="url">{server.url}</code>}
                {server.service && <span className="tag">{server.service}</span>}
              </div>
              {server.author && <div className="author">By {server.author}</div>}
              {server.env && Object.keys(server.env).length > 0 && (
                <div style={{ marginTop: 8, fontSize: 12, color: 'var(--text-dim)' }}>
                  Env: {Object.keys(server.env).join(', ')}
                </div>
              )}
              <div className="mcp-card-footer">
                <div className="tag tag-primary">{server.enabled ? 'Active' : 'Disabled'}</div>
                <button
                  className="btn btn-sm btn-danger"
                  onClick={() => handleDelete(server.name)}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showInstall && (
        <div className="modal-overlay" onClick={resetForm}>
          <div className="modal modal-large" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Add MCP Server</h3>
              <button className="modal-close" onClick={resetForm}>×</button>
            </div>
            <div className="modal-body">
              <div className="tabs">
                <button
                  className={`tab ${installTab === 'github' ? 'active' : ''}`}
                  onClick={() => setInstallTab('github')}
                >
                  From GitHub
                </button>
                <button
                  className={`tab ${installTab === 'upload' ? 'active' : ''}`}
                  onClick={() => setInstallTab('upload')}
                >
                  Upload .md
                </button>
                <button
                  className={`tab ${installTab === 'zip' ? 'active' : ''}`}
                  onClick={() => setInstallTab('zip')}
                >
                  Upload .zip
                </button>
                <button
                  className={`tab ${installTab === 'manual' ? 'active' : ''}`}
                  onClick={() => setInstallTab('manual')}
                >
                  Manual Config
                </button>
              </div>

              {installTab === 'github' ? (
                <div className="form-compact">
                  <div className="form-group">
                    <label>GitHub URL</label>
                    <input
                      className="input"
                      value={githubUrl}
                      onChange={e => setGithubUrl(e.target.value)}
                      placeholder="github.com/owner/repo"
                    />
                    <div className="form-hint">
                      Supports: <code>github.com/owner/repo</code> or full GitHub URL
                    </div>
                  </div>
                </div>
              ) : installTab === 'upload' ? (
                <div className="form-compact">
                  <div
                    className={`file-upload-area ${uploadFile ? 'has-file' : ''}`}
                    onClick={() => fileInputRef.current?.click()}
                  >
                    <input
                      type="file"
                      ref={fileInputRef}
                      onChange={handleFileChange}
                      accept=".md"
                      style={{ display: 'none' }}
                    />
                    <div className="file-upload-icon">📄</div>
                    <div className="file-upload-text">
                      {uploadFile ? uploadFile.name : 'Click to select MCP.md file'}
                    </div>
                    <div className="file-upload-hint">
                      {uploadFile ? `${Math.round(uploadFile.size / 1024)} KB` : 'Supports Markdown files'}
                    </div>
                  </div>
                  <div className="form-group" style={{ marginTop: 16 }}>
                    <label>Server Name <span className="optional">(optional)</span></label>
                    <input
                      className="input"
                      value={uploadName}
                      onChange={e => setUploadName(e.target.value)}
                      placeholder="Auto-detected from file"
                    />
                  </div>
                </div>
              ) : installTab === 'zip' ? (
                <div className="form-compact">
                  <div
                    className={`file-upload-area ${uploadFile ? 'has-file' : ''}`}
                    onClick={() => fileInputRef.current?.click()}
                  >
                    <input
                      type="file"
                      ref={fileInputRef}
                      onChange={handleFileChange}
                      accept=".zip"
                      style={{ display: 'none' }}
                    />
                    <div className="file-upload-icon">📦</div>
                    <div className="file-upload-text">
                      {uploadFile ? uploadFile.name : 'Click to select ZIP file'}
                    </div>
                    <div className="file-upload-hint">
                      {uploadFile ? `${Math.round(uploadFile.size / 1024)} KB` : 'Supports ZIP archives'}
                    </div>
                  </div>
                  <div className="form-group" style={{ marginTop: 16 }}>
                    <label>Server Name <span className="optional">(optional)</span></label>
                    <input
                      className="input"
                      value={uploadName}
                      onChange={e => setUploadName(e.target.value)}
                      placeholder="Auto-detected from manifest"
                    />
                  </div>
                </div>
              ) : (
                <div className="form-compact">
                  <div className="form-group">
                    <label>Name</label>
                    <input
                      type="text"
                      className="input"
                      value={formData.name}
                      onChange={e => setFormData({ ...formData, name: e.target.value })}
                      placeholder="e.g., k8s-mcp"
                    />
                  </div>
                  <div className="form-group">
                    <label>Transport</label>
                    <select
                      value={formData.transport}
                      onChange={e => setFormData({ ...formData, transport: e.target.value })}
                    >
                      <option value="stdio">stdio (local command)</option>
                      <option value="sse">sse (remote URL)</option>
                    </select>
                  </div>
                  {formData.transport === 'stdio' ? (
                    <>
                      <div className="form-group">
                        <label>Command</label>
                        <input
                          type="text"
                          className="input"
                          value={formData.command}
                          onChange={e => setFormData({ ...formData, command: e.target.value })}
                          placeholder="e.g., npx"
                        />
                      </div>
                      <div className="form-group">
                        <label>Arguments</label>
                        <input
                          type="text"
                          className="input"
                          value={formData.args?.join(' ')}
                          onChange={e => setFormData({ ...formData, args: e.target.value.split(' ').filter(Boolean) })}
                          placeholder="-y @kubernetes-mcp/server"
                        />
                      </div>
                    </>
                  ) : (
                    <div className="form-group">
                      <label>URL</label>
                      <input
                        type="text"
                        className="input"
                        value={formData.url}
                        onChange={e => setFormData({ ...formData, url: e.target.value })}
                        placeholder="http://localhost:3000/sse"
                      />
                    </div>
                  )}
                  <JsonEditor
                    label="Environment Variables"
                    value={envJson}
                    onChange={(value, valid) => {
                      setEnvJson(value)
                      setEnvValid(valid)
                    }}
                    placeholder='{\n  "API_KEY": "your-api-key",\n  "BASE_URL": "https://api.example.com"\n}'
                    hint="Configure environment variables as JSON object"
                  />
                </div>
              )}
            </div>
            <div className="modal-footer">
              <button className="btn" onClick={resetForm}>Cancel</button>
              {installTab === 'github' ? (
                <button
                  className="btn btn-primary"
                  onClick={handleInstallFromGitHub}
                  disabled={!githubUrl.trim() || installing}
                >
                  {installing ? 'Installing...' : 'Install'}
                </button>
              ) : installTab === 'upload' || installTab === 'zip' ? (
                <button
                  className="btn btn-primary"
                  onClick={handleUpload}
                  disabled={!uploadFile || installing}
                >
                  {installing ? 'Uploading...' : 'Upload'}
                </button>
              ) : (
                <button
                  className="btn btn-primary"
                  onClick={handleManualAdd}
                  disabled={!formData.name || !envValid}
                >
                  Add Server
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

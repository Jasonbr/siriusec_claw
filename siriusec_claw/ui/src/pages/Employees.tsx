import { useState, useEffect, useRef } from 'react'

interface Employee {
  id: string
  name: string
  description?: string
  prompt?: string
  enabled: boolean
  builtin?: boolean
  type?: string
  from?: string
  skillIds?: string[]
  mcpServerKeys?: string[]
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

// MCP Selector Component
function MCPSelector({
  availableMCPs,
  selectedMCPs,
  onChange
}: {
  availableMCPs: string[]
  selectedMCPs: string[]
  onChange: (selected: string[]) => void
}) {
  if (availableMCPs.length === 0) {
    return (
      <div className="form-hint" style={{ color: 'var(--text-dim)' }}>
        No MCP servers available. Add MCP servers first.
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
      {availableMCPs.map(mcp => (
        <label
          key={mcp}
          className={`tag ${selectedMCPs.includes(mcp) ? 'tag-primary' : ''}`}
          style={{
            cursor: 'pointer',
            padding: '6px 12px',
            border: selectedMCPs.includes(mcp) ? 'none' : '1px solid var(--border)'
          }}
        >
          <input
            type="checkbox"
            checked={selectedMCPs.includes(mcp)}
            onChange={e => {
              if (e.target.checked) {
                onChange([...selectedMCPs, mcp])
              } else {
                onChange(selectedMCPs.filter(m => m !== mcp))
              }
            }}
            style={{ display: 'none' }}
          />
          {selectedMCPs.includes(mcp) && '✓ '}
          {mcp}
        </label>
      ))}
    </div>
  )
}

export default function EmployeesPage({ call }: Props) {
  const [employees, setEmployees] = useState<Employee[]>([])
  const [loading, setLoading] = useState(false)
  const [showInstall, setShowInstall] = useState(false)
  const [installTab, setInstallTab] = useState<'github' | 'upload' | 'zip' | 'manual'>('github')
  const [installing, setInstalling] = useState(false)
  const [editing, setEditing] = useState<Employee | null>(null)

  // GitHub install
  const [githubUrl, setGithubUrl] = useState('')

  // File upload
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadName, setUploadName] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Manual form
  const [formData, setFormData] = useState<Partial<Employee>>({
    name: '',
    description: '',
    prompt: '',
    enabled: true,
    type: '',
  })
  const [availableMCPs, setAvailableMCPs] = useState<string[]>([])
  const [selectedMCPs, setSelectedMCPs] = useState<string[]>([])
  const [envJson, setEnvJson] = useState('')
  const [envValid, setEnvValid] = useState(true)

  useEffect(() => {
    loadEmployees()
    loadAvailableMCPs()
  }, [])

  const loadAvailableMCPs = async () => {
    try {
      const res = await call('mcp.list')
      if (res.ok && res.payload.servers) {
        setAvailableMCPs(res.payload.servers.map((s: any) => s.name))
      }
    } catch (err) {
      console.error('Failed to load MCP servers:', err)
    }
  }

  const loadEmployees = async () => {
    setLoading(true)
    try {
      const res = await call('employees.list')
      if (res.ok) {
        setEmployees(res.payload.employees || [])
      }
    } catch (err) {
      console.error('Failed to load employees:', err)
    }
    setLoading(false)
  }

  const handleInstallFromGitHub = async () => {
    if (!githubUrl.trim()) return
    setInstalling(true)
    try {
      const res = await call('employees.install', { source: 'github', url: githubUrl.trim() })
      setInstalling(false)
      if (res.ok) {
        setGithubUrl('')
        setShowInstall(false)
        loadEmployees()
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
        const res = await call('employees.install', {
          source: 'zip',
          name: uploadName.trim() || undefined,
          content: base64
        })
        setInstalling(false)
        if (res.ok) {
          setUploadFile(null)
          setUploadName('')
          setShowInstall(false)
          loadEmployees()
        } else {
          alert('Upload failed: ' + (res.error?.message || 'Unknown error'))
        }
        return
      }

      // Handle .md file
      const content = await uploadFile.text()
      const base64 = btoa(unescape(encodeURIComponent(content)))
      const res = await call('employees.install', {
        source: 'upload',
        name: uploadName.trim() || undefined,
        content: base64
      })
      setInstalling(false)
      if (res.ok) {
        setUploadFile(null)
        setUploadName('')
        setShowInstall(false)
        loadEmployees()
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
        description: formData.description,
        prompt: formData.prompt,
        enabled: formData.enabled,
        type: formData.type,
      }

      // Add MCP dependencies if any selected
      if (selectedMCPs.length > 0) {
        params.mcpServers = {}
        selectedMCPs.forEach(mcpName => {
          params.mcpServers[mcpName] = { enabled: true }
        })
      }

      // Add environment variables if any
      if (envJson.trim()) {
        params.env = JSON.parse(envJson)
      }

      const res = await call('employees.create', params)
      if (res.ok) {
        resetForm()
        loadEmployees()
      } else {
        alert('Failed to create employee: ' + (res.error?.message || 'Unknown error'))
      }
    } catch (err) {
      alert('Failed to create employee: ' + err)
    }
  }

  const handleUpdate = async () => {
    if (!editing || !formData.name) return

    try {
      const res = await call('employees.update', {
        id: editing.id,
        name: formData.name,
        description: formData.description,
        prompt: formData.prompt,
        enabled: formData.enabled,
        type: formData.type,
      })
      if (res.ok) {
        setEditing(null)
        setFormData({ name: '', description: '', prompt: '', enabled: true, type: '' })
        loadEmployees()
      } else {
        alert('Failed to update employee: ' + (res.error?.message || 'Unknown error'))
      }
    } catch (err) {
      alert('Failed to update employee: ' + err)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this employee?')) return
    try {
      const res = await call('employees.delete', { id })
      if (res.ok) {
        loadEmployees()
      } else {
        alert('Failed to delete: ' + (res.error?.message || 'Unknown error'))
      }
    } catch (err) {
      alert('Failed to delete: ' + err)
    }
  }

  const handleToggle = async (emp: Employee) => {
    try {
      const res = await call('employees.update', {
        id: emp.id,
        enabled: !emp.enabled
      })
      if (res.ok) {
        loadEmployees()
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

  const startEdit = (emp: Employee) => {
    setEditing(emp)
    setFormData({
      name: emp.name,
      description: emp.description || '',
      prompt: emp.prompt || '',
      enabled: emp.enabled,
      type: emp.type || '',
    })
  }

  const resetForm = () => {
    setShowInstall(false)
    setInstallTab('github')
    setGithubUrl('')
    setUploadFile(null)
    setUploadName('')
    setFormData({ name: '', description: '', prompt: '', enabled: true, type: '' })
    setSelectedMCPs([])
    setEnvJson('')
    setEnvValid(true)
  }

  return (
    <div className="page">
      <div className="page-header">
        <h2>Digital Employees</h2>
        <button className="btn btn-primary" onClick={() => setShowInstall(true)}>
          + Create Employee
        </button>
      </div>

      {loading ? (
        <div className="loading">Loading...</div>
      ) : employees.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">👤</div>
          <div className="empty-state-title">No digital employees created yet</div>
          <div className="empty-state-desc">Create your first digital employee to enable specialized AI assistants</div>
          <button className="btn btn-primary" onClick={() => setShowInstall(true)}>
            Create Your First Employee
          </button>
        </div>
      ) : (
        <div className="employees-grid">
          {employees.map(emp => (
            <div key={emp.id} className={`employee-card ${emp.enabled ? '' : 'disabled'}`}>
              <div className="employee-card-header">
                <div>
                  <div className="employee-name">{emp.name}</div>
                  <div className="employee-badges" style={{ marginTop: 6 }}>
                    {emp.builtin && <span className="badge builtin">Built-in</span>}
                    {emp.type && <span className="badge type">{emp.type}</span>}
                  </div>
                </div>
                <label className="toggle">
                  <input
                    type="checkbox"
                    checked={emp.enabled}
                    onChange={() => handleToggle(emp)}
                  />
                  <span className="toggle-slider"></span>
                </label>
              </div>
              <div className="employee-card-body">
                {emp.description && <p>{emp.description}</p>}
                <div className="employee-meta">
                  {emp.mcpServerKeys && emp.mcpServerKeys.length > 0 && (
                    <div className="meta-item">
                      <span className="meta-label">MCP:</span>
                      <span className="meta-value">{emp.mcpServerKeys.join(', ')}</span>
                    </div>
                  )}
                  {emp.skillIds && emp.skillIds.length > 0 && (
                    <div className="meta-item">
                      <span className="meta-label">Skills:</span>
                      <span className="meta-value">{emp.skillIds.length}</span>
                    </div>
                  )}
                </div>
              </div>
              <div className="employee-card-footer">
                <div className="tag tag-primary">{emp.enabled ? 'Active' : 'Disabled'}</div>
                <div className="btn-group">
                  <button className="btn btn-sm" onClick={() => startEdit(emp)}>
                    Edit
                  </button>
                  {!emp.builtin && (
                    <button
                      className="btn btn-sm btn-danger"
                      onClick={() => handleDelete(emp.id)}
                    >
                      Delete
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create Modal */}
      {showInstall && (
        <div className="modal-overlay" onClick={resetForm}>
          <div className="modal modal-large" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Create Digital Employee</h3>
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
                  Manual Create
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
                      {uploadFile ? uploadFile.name : 'Click to select EMPLOYEE.md file'}
                    </div>
                    <div className="file-upload-hint">
                      {uploadFile ? `${Math.round(uploadFile.size / 1024)} KB` : 'Supports Markdown files'}
                    </div>
                  </div>
                  <div className="form-group" style={{ marginTop: 16 }}>
                    <label>Employee Name <span className="optional">(optional)</span></label>
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
                    <label>Employee Name <span className="optional">(optional)</span></label>
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
                      placeholder="e.g., MySQL Expert"
                    />
                  </div>
                  <div className="form-group">
                    <label>Type <span className="optional">(optional)</span></label>
                    <input
                      type="text"
                      className="input"
                      value={formData.type}
                      onChange={e => setFormData({ ...formData, type: e.target.value })}
                      placeholder="e.g., Database, DevOps"
                    />
                  </div>
                  <div className="form-group">
                    <label>Description <span className="optional">(optional)</span></label>
                    <textarea
                      className="textarea"
                      value={formData.description}
                      onChange={e => setFormData({ ...formData, description: e.target.value })}
                      placeholder="Brief description of this employee's expertise"
                      rows={2}
                    />
                  </div>
                  <div className="form-group">
                    <label>System Prompt <span className="optional">(optional)</span></label>
                    <textarea
                      className="textarea"
                      value={formData.prompt}
                      onChange={e => setFormData({ ...formData, prompt: e.target.value })}
                      placeholder="You are an expert in..."
                      rows={4}
                    />
                  </div>
                  <div className="form-group">
                    <label>Dependent MCP Servers <span className="optional">(optional)</span></label>
                    <MCPSelector
                      availableMCPs={availableMCPs}
                      selectedMCPs={selectedMCPs}
                      onChange={setSelectedMCPs}
                    />
                    <div className="form-hint">
                      Select MCP servers this employee depends on
                    </div>
                  </div>
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
                  Create
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Edit Modal */}
      {editing && (
        <div className="modal-overlay" onClick={() => setEditing(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Edit Employee</h3>
              <button className="modal-close" onClick={() => setEditing(null)}>×</button>
            </div>
            <div className="modal-body form-compact">
              <div className="form-group">
                <label>Name</label>
                <input
                  type="text"
                  className="input"
                  value={formData.name}
                  onChange={e => setFormData({ ...formData, name: e.target.value })}
                  placeholder="e.g., MySQL Expert"
                />
              </div>
              <div className="form-group">
                <label>Type <span className="optional">(optional)</span></label>
                <input
                  type="text"
                  className="input"
                  value={formData.type}
                  onChange={e => setFormData({ ...formData, type: e.target.value })}
                  placeholder="e.g., Database, DevOps"
                />
              </div>
              <div className="form-group">
                <label>Description <span className="optional">(optional)</span></label>
                <textarea
                  className="textarea"
                  value={formData.description}
                  onChange={e => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Brief description"
                  rows={2}
                />
              </div>
              <div className="form-group">
                <label>System Prompt <span className="optional">(optional)</span></label>
                <textarea
                  className="textarea"
                  value={formData.prompt}
                  onChange={e => setFormData({ ...formData, prompt: e.target.value })}
                  placeholder="You are an expert in..."
                  rows={4}
                />
              </div>
              <div className="form-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={e => setFormData({ ...formData, enabled: e.target.checked })}
                  />
                  Enabled
                </label>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn" onClick={() => setEditing(null)}>Cancel</button>
              <button
                className="btn btn-primary"
                onClick={handleUpdate}
                disabled={!formData.name}
              >
                Update
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

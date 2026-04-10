import { useEffect, useState, useRef } from 'react'
import type { RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

type APIConfigField = {
  name: string
  label: string
  type: 'text' | 'password' | 'number' | 'boolean' | 'select'
  required: boolean
  default?: string
  placeholder?: string
  options?: string[]
  description?: string
}

type APIConfig = {
  url?: string
  urlLabel?: string
  urlRequired?: boolean
  authType?: 'none' | 'token' | 'basic' | 'apikey'
  tokenLabel?: string
  usernameLabel?: string
  passwordLabel?: string
  apiKeyLabel?: string
  apiKeyHeader?: string
  extraFields?: APIConfigField[]
}

type Skill = {
  name: string
  source: string
  enabled: boolean
  filePath: string
  hasMetadata: boolean
  emoji?: string
  homepage?: string
  description?: string
  sourceURL?: string
  installedAt?: string
  updatedAt?: string
  installSource?: string
  apiConfig?: APIConfig
}

type SkillConfig = {
  enabled?: boolean
  apiKey?: string
  env?: Record<string, string>
  config?: Record<string, any>
}

export default function Skills({ call }: Props) {
  const [skills, setSkills] = useState<Skill[]>([])
  const [loading, setLoading] = useState(false)
  const [showInstall, setShowInstall] = useState(false)
  const [installTab, setInstallTab] = useState<'github' | 'upload' | 'zip'>('github')
  const [githubUrl, setGithubUrl] = useState('')
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadName, setUploadName] = useState('')
  const [installing, setInstalling] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Config modal state
  const [configSkill, setConfigSkill] = useState<Skill | null>(null)
  const [skillConfig, setSkillConfig] = useState<SkillConfig>({})
  const [configLoading, setConfigLoading] = useState(false)
  const [configSaving, setConfigSaving] = useState(false)

  const load = async () => {
    setLoading(true)
    const r = await call('skills.status')
    if (r.ok) {
      setSkills(r.payload?.skills || [])
    }
    setLoading(false)
  }

  useEffect(() => { load() }, [call])

  const loadSkillConfig = async (skill: Skill) => {
    setConfigLoading(true)
    setConfigSkill(skill)
    const r = await call('skills.getConfig', { name: skill.name })
    if (r.ok) {
      setSkillConfig(r.payload?.config || {})
    }
    setConfigLoading(false)
  }

  const saveSkillConfig = async () => {
    if (!configSkill) return
    setConfigSaving(true)
    const r = await call('skills.setConfig', {
      name: configSkill.name,
      ...skillConfig
    })
    setConfigSaving(false)
    if (r.ok) {
      setConfigSkill(null)
    } else {
      alert('Failed to save config: ' + (r.error?.message || 'Unknown error'))
    }
  }

  const handleInstallFromGitHub = async () => {
    if (!githubUrl.trim()) return
    setInstalling(true)
    const r = await call('skills.install', { source: 'github', url: githubUrl.trim() })
    setInstalling(false)
    if (r.ok) {
      setGithubUrl('')
      setShowInstall(false)
      load()
    } else {
      alert('Install failed: ' + (r.error?.message || 'Unknown error'))
    }
  }

  const handleUpload = async () => {
    if (!uploadFile) return
    setInstalling(true)
    
    // Check if it's a zip file
    if (uploadFile.name.toLowerCase().endsWith('.zip')) {
      const buffer = await uploadFile.arrayBuffer()
      const base64 = btoa(String.fromCharCode(...new Uint8Array(buffer)))
      const r = await call('skills.install', {
        source: 'zip',
        name: uploadName.trim() || undefined,
        content: base64
      })
      setInstalling(false)
      if (r.ok) {
        setUploadFile(null)
        setUploadName('')
        setShowInstall(false)
        load()
      } else {
        alert('Upload failed: ' + (r.error?.message || 'Unknown error'))
      }
      return
    }
    
    // Handle SKILL.md file
    const content = await uploadFile.text()
    const base64 = btoa(unescape(encodeURIComponent(content)))
    const r = await call('skills.install', {
      source: 'upload',
      name: uploadName.trim() || undefined,
      content: base64
    })
    setInstalling(false)
    if (r.ok) {
      setUploadFile(null)
      setUploadName('')
      setShowInstall(false)
      load()
    } else {
      alert('Upload failed: ' + (r.error?.message || 'Unknown error'))
    }
  }

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete skill "${name}"?`)) return
    const r = await call('skills.delete', { name })
    if (r.ok) {
      load()
    } else {
      alert('Delete failed: ' + (r.error?.message || 'Unknown error'))
    }
  }

  const handleUpdate = async (name: string) => {
    setInstalling(true)
    const r = await call('skills.update', { name })
    setInstalling(false)
    if (r.ok) {
      load()
    } else {
      alert('Update failed: ' + (r.error?.message || 'Unknown error'))
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

  const filteredSkills = skills.filter(s =>
    s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (s.description && s.description.toLowerCase().includes(searchQuery.toLowerCase()))
  )

  const groupedSkills = filteredSkills.reduce((acc, s) => {
    const source = s.source === 'managed' ? 'Installed' :
                   s.source === 'workspace' ? 'Workspace' :
                   s.source === 'builtin' ? 'Built-in' : 'Other'
    if (!acc[source]) acc[source] = []
    acc[source].push(s)
    return acc
  }, {} as Record<string, Skill[]>)

  const sourceOrder = ['Installed', 'Workspace', 'Built-in', 'Other']

  const renderConfigModal = () => {
    if (!configSkill) return null

    const api = configSkill.apiConfig
    const hasApiConfig = api && (api.urlRequired || api.authType || (api.extraFields && api.extraFields.length > 0))

    return (
      <div className="modal-overlay" onClick={() => setConfigSkill(null)}>
        <div className="modal" onClick={e => e.stopPropagation()} style={{ minWidth: 480, maxWidth: 600 }}>
          <div className="modal-header">
            <h3 className="modal-title">
              {configSkill.emoji || '✨'} {configSkill.name} Configuration
            </h3>
            <button className="modal-close" onClick={() => setConfigSkill(null)}>×</button>
          </div>

          {configLoading ? (
            <div className="modal-body" style={{ textAlign: 'center', padding: 40 }}>
              Loading...
            </div>
          ) : (
            <>
              <div className="modal-body">
                {configSkill.description && (
                  <p className="text-secondary mb-12">{configSkill.description}</p>
                )}

                {hasApiConfig ? (
                  <div className="api-config-form">
                    {/* URL Field */}
                    {api!.urlRequired !== false && (
                      <div className="form-group">
                        <label className="form-label">
                          {api!.urlLabel || 'API URL'}
                          {api!.urlRequired && <span className="required">*</span>}
                        </label>
                        <input
                          className="input"
                          type="url"
                          value={skillConfig.config?.url || ''}
                          onChange={e => setSkillConfig({
                            ...skillConfig,
                            config: { ...skillConfig.config, url: e.target.value }
                          })}
                          placeholder="https://api.example.com"
                        />
                      </div>
                    )}

                    {/* Auth Type: Token */}
                    {api!.authType === 'token' && (
                      <div className="form-group">
                        <label className="form-label">
                          {api!.tokenLabel || 'Token'}
                          <span className="required">*</span>
                        </label>
                        <input
                          className="input"
                          type="password"
                          value={skillConfig.config?.token || ''}
                          onChange={e => setSkillConfig({
                            ...skillConfig,
                            config: { ...skillConfig.config, token: e.target.value }
                          })}
                          placeholder="Enter your API token"
                        />
                      </div>
                    )}

                    {/* Auth Type: Basic */}
                    {api!.authType === 'basic' && (
                      <>
                        <div className="form-group">
                          <label className="form-label">
                            {api!.usernameLabel || 'Username'}
                            <span className="required">*</span>
                          </label>
                          <input
                            className="input"
                            type="text"
                            value={skillConfig.config?.username || ''}
                            onChange={e => setSkillConfig({
                              ...skillConfig,
                              config: { ...skillConfig.config, username: e.target.value }
                            })}
                          />
                        </div>
                        <div className="form-group">
                          <label className="form-label">
                            {api!.passwordLabel || 'Password'}
                            <span className="required">*</span>
                          </label>
                          <input
                            className="input"
                            type="password"
                            value={skillConfig.config?.password || ''}
                            onChange={e => setSkillConfig({
                              ...skillConfig,
                              config: { ...skillConfig.config, password: e.target.value }
                            })}
                          />
                        </div>
                      </>
                    )}

                    {/* Auth Type: API Key */}
                    {api!.authType === 'apikey' && (
                      <div className="form-group">
                        <label className="form-label">
                          {api!.apiKeyLabel || 'API Key'}
                          <span className="required">*</span>
                        </label>
                        <input
                          className="input"
                          type="password"
                          value={skillConfig.config?.apiKey || skillConfig.apiKey || ''}
                          onChange={e => setSkillConfig({
                            ...skillConfig,
                            apiKey: e.target.value,
                            config: { ...skillConfig.config, apiKey: e.target.value }
                          })}
                          placeholder="Enter your API key"
                        />
                        {api!.apiKeyHeader && (
                          <small className="form-hint">Will be sent as header: {api!.apiKeyHeader}</small>
                        )}
                      </div>
                    )}

                    {/* Extra Fields */}
                    {api!.extraFields?.map(field => (
                      <div key={field.name} className="form-group">
                        <label className="form-label">
                          {field.label}
                          {field.required && <span className="required">*</span>}
                        </label>
                        {field.type === 'boolean' ? (
                          <label className="checkbox-label">
                            <input
                              type="checkbox"
                              checked={skillConfig.config?.[field.name] === true}
                              onChange={e => setSkillConfig({
                                ...skillConfig,
                                config: { ...skillConfig.config, [field.name]: e.target.checked }
                              })}
                            />
                            {field.description}
                          </label>
                        ) : field.type === 'select' && field.options ? (
                          <select
                            className="input"
                            value={skillConfig.config?.[field.name] || field.default || ''}
                            onChange={e => setSkillConfig({
                              ...skillConfig,
                              config: { ...skillConfig.config, [field.name]: e.target.value }
                            })}
                          >
                            <option value="">Select...</option>
                            {field.options.map(opt => (
                              <option key={opt} value={opt}>{opt}</option>
                            ))}
                          </select>
                        ) : (
                          <input
                            className="input"
                            type={field.type === 'password' ? 'password' : 'text'}
                            value={skillConfig.config?.[field.name] || field.default || ''}
                            onChange={e => setSkillConfig({
                              ...skillConfig,
                              config: { ...skillConfig.config, [field.name]: e.target.value }
                            })}
                            placeholder={field.placeholder}
                          />
                        )}
                        {field.description && field.type !== 'boolean' && (
                          <small className="form-hint">{field.description}</small>
                        )}
                      </div>
                    ))}

                    <div className="form-group">
                      <label className="form-label">Environment Variables (optional)</label>
                      <textarea
                        className="input"
                        rows={3}
                        value={Object.entries(skillConfig.env || {}).map(([k, v]) => `${k}=${v}`).join('\n')}
                        onChange={e => {
                          const env: Record<string, string> = {}
                          e.target.value.split('\n').forEach(line => {
                            const [k, ...v] = line.split('=')
                            if (k && v.length) env[k.trim()] = v.join('=').trim()
                          })
                          setSkillConfig({ ...skillConfig, env })
                        }}
                        placeholder="KEY=value&#10;ANOTHER_KEY=value"
                      />
                    </div>
                  </div>
                ) : (
                  <div className="form-group">
                    <p className="text-secondary">This skill does not require API configuration.</p>
                    <label className="form-label">Environment Variables (optional)</label>
                    <textarea
                      className="input"
                      rows={3}
                      value={Object.entries(skillConfig.env || {}).map(([k, v]) => `${k}=${v}`).join('\n')}
                      onChange={e => {
                        const env: Record<string, string> = {}
                        e.target.value.split('\n').forEach(line => {
                          const [k, ...v] = line.split('=')
                          if (k && v.length) env[k.trim()] = v.join('=').trim()
                        })
                        setSkillConfig({ ...skillConfig, env })
                      }}
                      placeholder="KEY=value"
                    />
                  </div>
                )}
              </div>

              <div className="modal-footer">
                <button className="btn" onClick={() => setConfigSkill(null)}>Cancel</button>
                <button
                  className="btn btn-primary"
                  onClick={saveSkillConfig}
                  disabled={configSaving}
                >
                  {configSaving ? 'Saving...' : 'Save Configuration'}
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    )
  }

  return (
    <div>
      <div className="flex gap-12 mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Skills</h1>
        <div className="ml-auto btn-group">
          <input
            type="text"
            className="input input-sm"
            placeholder="Search skills..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            style={{ width: 200 }}
          />
          <button className="btn btn-sm" onClick={load} disabled={loading}>
            {loading ? 'Loading...' : 'Refresh'}
          </button>
          <button className="btn btn-sm btn-primary" onClick={() => setShowInstall(!showInstall)}>
            {showInstall ? 'Cancel' : '+ Install Skill'}
          </button>
        </div>
      </div>

      {showInstall && (
        <div className="card mb-16">
          <div className="card-title">Install Skill</div>
          <div className="tabs mb-12">
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
              Upload SKILL.md
            </button>
            <button
              className={`tab ${installTab === 'zip' ? 'active' : ''}`}
              onClick={() => setInstallTab('zip')}
            >
              Upload .zip
            </button>
          </div>

          {installTab === 'github' ? (
            <div>
              <div className="form-group">
                <label className="form-label">GitHub URL</label>
                <input
                  className="input"
                  value={githubUrl}
                  onChange={e => setGithubUrl(e.target.value)}
                  placeholder="github.com/owner/repo or full URL"
                />
                <small className="form-hint">
                  Supports: github.com/owner/repo, github.com/owner/repo/tree/main/path/to/skill
                </small>
              </div>
              <button
                className="btn btn-primary"
                onClick={handleInstallFromGitHub}
                disabled={!githubUrl.trim() || installing}
              >
                {installing ? 'Installing...' : 'Install from GitHub'}
              </button>
            </div>
          ) : installTab === 'upload' ? (
            <div>
              <div className="form-group">
                <label className="form-label">SKILL.md File</label>
                <input
                  type="file"
                  ref={fileInputRef}
                  onChange={handleFileChange}
                  accept=".md"
                  style={{ display: 'none' }}
                />
                <div className="flex gap-8">
                  <button
                    className="btn"
                    onClick={() => fileInputRef.current?.click()}
                  >
                    {uploadFile ? 'Change File' : 'Select File'}
                  </button>
                  {uploadFile && (
                    <span className="form-hint" style={{ alignSelf: 'center' }}>
                      {uploadFile.name} ({Math.round(uploadFile.size / 1024)}KB)
                    </span>
                  )}
                </div>
              </div>
              <div className="form-group">
                <label className="form-label">Skill Name (optional)</label>
                <input
                  className="input"
                  value={uploadName}
                  onChange={e => setUploadName(e.target.value)}
                  placeholder="Auto-detected from file or frontmatter"
                />
              </div>
              <button
                className="btn btn-primary"
                onClick={handleUpload}
                disabled={!uploadFile || installing}
              >
                {installing ? 'Uploading...' : 'Upload Skill'}
              </button>
            </div>
          ) : (
            <div>
              <div className="form-group">
                <label className="form-label">Skill Package (.zip)</label>
                <input
                  type="file"
                  ref={fileInputRef}
                  onChange={handleFileChange}
                  accept=".zip"
                  style={{ display: 'none' }}
                />
                <div className="flex gap-8">
                  <button
                    className="btn"
                    onClick={() => fileInputRef.current?.click()}
                  >
                    {uploadFile ? 'Change File' : 'Select File'}
                  </button>
                  {uploadFile && (
                    <span className="form-hint" style={{ alignSelf: 'center' }}>
                      {uploadFile.name} ({Math.round(uploadFile.size / 1024)}KB)
                    </span>
                  )}
                </div>
                <small className="form-hint">
                  Upload a .zip file containing SKILL.md or a folder with SKILL.md inside
                </small>
              </div>
              <div className="form-group">
                <label className="form-label">Skill Name (optional)</label>
                <input
                  className="input"
                  value={uploadName}
                  onChange={e => setUploadName(e.target.value)}
                  placeholder="Auto-detected from frontmatter or folder name"
                />
              </div>
              <button
                className="btn btn-primary"
                onClick={handleUpload}
                disabled={!uploadFile || installing}
              >
                {installing ? 'Uploading...' : 'Upload'}
              </button>
            </div>
          )}
        </div>
      )}

      {skills.length === 0 ? (
        <div className="empty">No skills installed</div>
      ) : (
        <div>
          {sourceOrder.map(source => {
            const group = groupedSkills[source]
            if (!group || group.length === 0) return null
            return (
              <div key={source} className="mb-16">
                <h3 className="section-title">{source.toUpperCase()} ({group.length})</h3>
                <div className="skill-grid">
                  {group.map(skill => (
                    <div key={skill.name} className="skill-card">
                      <div className="skill-card-header">
                        <div className="skill-icon">
                          {skill.emoji || '✨'}
                        </div>
                        <div className="skill-title">
                          <div className="skill-name">{skill.name}</div>
                          {skill.description && (
                            <div className="skill-desc">{skill.description}</div>
                          )}
                        </div>
                        <span className={`badge ${skill.source === 'managed' ? 'badge-blue' : skill.source === 'workspace' ? 'badge-green' : 'badge-gray'}`}>
                          {skill.source}
                        </span>
                      </div>

                      <div className="skill-card-body">
                        {skill.sourceURL && (
                          <div className="skill-meta">
                            <span className="skill-meta-label">Source:</span>
                            <a
                              href={skill.sourceURL}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="skill-meta-value link"
                            >
                              {skill.sourceURL.replace(/^https?:\/\//, '').substring(0, 30)}...
                            </a>
                          </div>
                        )}
                        {skill.installedAt && (
                          <div className="skill-meta">
                            <span className="skill-meta-label">Installed:</span>
                            <span className="skill-meta-value">
                              {new Date(skill.installedAt).toLocaleDateString()}
                            </span>
                          </div>
                        )}
                      </div>

                      <div className="skill-card-footer">
                        {/* Config button for skills with apiConfig */}
                        {skill.apiConfig && (
                          <button
                            className="btn btn-sm"
                            onClick={() => loadSkillConfig(skill)}
                          >
                            ⚙️ Configure
                          </button>
                        )}
                        {skill.source === 'managed' && skill.sourceURL && (
                          <button
                            className="btn btn-sm"
                            onClick={() => handleUpdate(skill.name)}
                            disabled={installing}
                          >
                            Update
                          </button>
                        )}
                        {skill.source === 'managed' && (
                          <button
                            className="btn btn-sm btn-danger"
                            onClick={() => handleDelete(skill.name)}
                          >
                            Delete
                          </button>
                        )}
                        {skill.source !== 'managed' && !skill.apiConfig && (
                          <span className="skill-hint">
                            {skill.source === 'workspace' ? 'Managed in workspace' : 'Built-in skill'}
                          </span>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}

      <div className="card" style={{ marginTop: 24 }}>
        <div className="card-title">About Skills</div>
        <p className="text-secondary">
          Skills extend the agent's capabilities with custom tools and behaviors.
          Install skills from GitHub repositories or upload SKILL.md files directly.
        </p>
        <ul className="list">
          <li><strong>Installed:</strong> Skills from GitHub or uploaded files, stored in ~/.siriusec_claw/skills/</li>
          <li><strong>Workspace:</strong> Project-specific skills in the workspace/skills/ directory</li>
          <li><strong>Built-in:</strong> Bundled skills that come with the application</li>
        </ul>
      </div>

      {/* Config Modal */}
      {renderConfigModal()}
    </div>
  )
}

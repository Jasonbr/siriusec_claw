import { useState } from 'react'
import './styles.css'
import { useGateway } from './hooks'
import Dashboard from './pages/Dashboard'
import Sessions from './pages/Sessions'
import Chat from './pages/Chat'
import Agents from './pages/Agents'
import CronPage from './pages/Cron'
import Channels from './pages/Channels'
import MemoryPage from './pages/Memory'
import ConfigPage from './pages/Config'
import SkillsPage from './pages/Skills'
import MCPPage from './pages/MCP'
import EmployeesPage from './pages/Employees'

type Page = 'dashboard' | 'sessions' | 'chat' | 'agents' | 'skills' | 'cron' | 'channels' | 'memory' | 'config' | 'mcp' | 'employees'

const NAV: { id: Page; label: string; icon: string }[] = [
  { id: 'dashboard', label: 'Dashboard', icon: '\u25a3' },
  { id: 'sessions', label: 'Sessions', icon: '\u2630' },
  { id: 'chat', label: 'Chat', icon: '\u2709' },
  { id: 'agents', label: 'Agents', icon: '\u2699' },
  { id: 'employees', label: 'Employees', icon: '\u263a' },
  { id: 'skills', label: 'Skills', icon: '\u2728' },
  { id: 'mcp', label: 'MCP', icon: '\u26a1' },
  { id: 'cron', label: 'Cron Jobs', icon: '\u23f0' },
  { id: 'channels', label: 'Channels', icon: '\u260d' },
  { id: 'memory', label: 'Memory', icon: '\u2601' },
  { id: 'config', label: 'Config', icon: '\u2638' },
]

export default function App() {
  const [page, setPage] = useState<Page>('dashboard')
  const { connected, call } = useGateway()

  const renderPage = () => {
    switch (page) {
      case 'dashboard': return <Dashboard call={call} />
      case 'sessions': return <Sessions call={call} />
      case 'chat': return <Chat call={call} />
      case 'agents': return <Agents call={call} />
      case 'employees': return <EmployeesPage call={call} />
      case 'skills': return <SkillsPage call={call} />
      case 'mcp': return <MCPPage call={call} />
      case 'cron': return <CronPage call={call} />
      case 'channels': return <Channels call={call} />
      case 'memory': return <MemoryPage call={call} />
      case 'config': return <ConfigPage call={call} />
    }
  }

  return (
    <div className="layout">
      <aside className="sidebar">
        <div className="sidebar-brand">
          SiriuSec Claw
          <small>Control Panel</small>
        </div>
        <nav className="sidebar-nav">
          {NAV.map(n => (
            <div
              key={n.id}
              className={`nav-item ${page === n.id ? 'active' : ''}`}
              onClick={() => setPage(n.id)}
            >
              <span className="icon">{n.icon}</span>
              {n.label}
            </div>
          ))}
        </nav>
        <div className="sidebar-footer">
          <span className={`status-dot ${connected ? 'on' : 'off'}`} />
          {connected ? 'Connected' : 'Disconnected'}
        </div>
      </aside>
      <main className="main">
        {renderPage()}
      </main>
    </div>
  )
}

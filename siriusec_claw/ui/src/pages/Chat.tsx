import { useState, useRef, useEffect } from 'react'
import { getClient, type RPCResponse } from '../gateway'

type Props = { call: (m: string, p?: any) => Promise<RPCResponse> }

type Message = { role: string; content: string; streaming?: boolean }
type Agent = { id: string; label?: string; model?: string }

export default function Chat({ call }: Props) {
  const [agents, setAgents] = useState<Agent[]>([])
  const [selectedAgent, setSelectedAgent] = useState('main')
  const [sessionKey, setSessionKey] = useState('agent:main:ui-chat')
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [streamingText, setStreamingText] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  // Load agents list
  useEffect(() => {
    call('agents.list').then(r => {
      if (r.ok && r.payload?.agents) {
        setAgents(r.payload.agents)
      }
    })
  }, [call])

  // Update sessionKey when agent changes
  useEffect(() => {
    setSessionKey(`agent:${selectedAgent}:ui-chat`)
  }, [selectedAgent])

  const loadHistory = () => {
    call('chat.history', { sessionKey }).then(r => {
      if (r.ok && r.payload?.messages) {
        const hist: Message[] = []
        for (const m of r.payload.messages) {
          if (m.role === 'user') {
            hist.push({ role: 'user', content: m.content?.[0]?.text || m.content || '' })
          } else if (m.role === 'assistant') {
            // Extract text from content blocks
            let text = ''
            for (const block of (m.content || [])) {
              if (block.type === 'text') text += block.text
            }
            if (text) hist.push({ role: 'assistant', content: text })
          }
        }
        setMessages(hist)
      }
    })
  }

  useEffect(() => { loadHistory() }, [sessionKey])

  // Listen to WebSocket events for streaming
  useEffect(() => {
    const client = getClient()
    const unsub = client.onEvent((event, payload) => {
      if (payload?.sessionKey !== sessionKey) return
      
      if (event === 'chat') {
        const type = payload.type
        if (type === 'stream.text_delta') {
          setStreamingText(prev => prev + (payload.text || ''))
        } else if (type === 'stream.message_stop') {
          // Message completed - use accumulated streamingText or payload.text
          setStreamingText(prev => {
            const finalText = prev || payload.text || ''
            if (finalText) {
              setMessages(msgs => [...msgs, { role: 'assistant', content: finalText }])
            }
            return ''
          })
          setSending(false)
        } else if (type === 'stream.error') {
          setMessages(prev => [...prev, { role: 'assistant', content: 'Error: ' + payload.text }])
          setStreamingText('')
          setSending(false)
        } else if (type === 'stream.tool_use') {
          // Show tool usage indicator
          setStreamingText(prev => prev + `\n🔧 Using tool: ${payload.toolName}...`)
        } else if (type === 'stream.tool_result') {
          // Show tool result
          const result = payload.result?.substring(0, 200) || '(empty)'
          setStreamingText(prev => prev + `\n✅ Result: ${result}`)
        }
        // Ignore message.assistant - we use stream.message_stop instead
      }
    })
    return unsub
  }, [sessionKey])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingText])

  const send = async () => {
    if (!input.trim() || sending) return
    const text = input.trim()
    setInput('')
    setSending(true)
    setStreamingText('')
    setMessages(prev => [...prev, { role: 'user', content: text }])

    const res = await call('chat.send', { sessionKey, message: text })
    if (!res.ok) {
      setMessages(prev => [...prev, { role: 'assistant', content: 'Error: ' + (res.error?.message || 'Unknown error') }])
      setSending(false)
    }
    // Response comes via WebSocket events, not direct response
  }

  return (
    <div>
      <div className="flex gap-12 mb-16">
        <h1 className="page-title" style={{ marginBottom: 0 }}>Chat</h1>
        <div className="ml-auto flex gap-8">
          <select
            className="input"
            style={{ width: 150 }}
            value={selectedAgent}
            onChange={e => setSelectedAgent(e.target.value)}
          >
            {agents.map(a => (
              <option key={a.id} value={a.id}>
                {a.label || a.id} {a.model ? `(${a.model})` : ''}
              </option>
            ))}
          </select>
          <input
            className="input"
            style={{ width: 200 }}
            value={sessionKey}
            onChange={e => setSessionKey(e.target.value)}
            placeholder="Session key"
          />
          <button className="btn btn-sm" onClick={loadHistory}>Load</button>
        </div>
      </div>

      <div className="chat-container">
        <div className="chat-messages">
          {messages.length === 0 && !streamingText && <div className="empty">No messages yet. Start a conversation.</div>}
          {messages.map((m, i) => (
            <div key={i} className={`chat-message ${m.role}`}>
              <div className="role">{m.role}</div>
              <div style={{ whiteSpace: 'pre-wrap', fontSize: 14 }}>{m.content}</div>
            </div>
          ))}
          {streamingText && (
            <div className="chat-message assistant streaming">
              <div className="role">assistant</div>
              <div style={{ whiteSpace: 'pre-wrap', fontSize: 14 }}>{streamingText}</div>
            </div>
          )}
          <div ref={bottomRef} />
        </div>
        <div className="chat-input-area">
          <input
            className="input"
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && send()}
            placeholder="Type a message..."
            disabled={sending}
          />
          <button className="btn btn-primary" onClick={send} disabled={sending}>
            {sending ? 'Sending...' : 'Send'}
          </button>
        </div>
      </div>
    </div>
  )
}

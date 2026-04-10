// WebSocket RPC client for the Gateway
export type RPCResponse = {
  ok: boolean
  payload?: any
  error?: { code: number; message: string }
}

type PendingRequest = {
  resolve: (res: RPCResponse) => void
  timer: ReturnType<typeof setTimeout>
}

type EventHandler = (event: string, payload: any) => void

export class GatewayClient {
  private ws: WebSocket | null = null
  private seq = 0
  private pending = new Map<string, PendingRequest>()
  private eventHandlers: EventHandler[] = []
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private _connected = false
  private _handshakeComplete = false
  private url: string
  private token: string

  constructor(url: string, token: string) {
    this.url = url
    this.token = token
  }

  get connected() { return this._connected && this._handshakeComplete }

  connect() {
    if (this.ws) return
    const wsUrl = `${this.url}?token=${encodeURIComponent(this.token)}&client=control-ui`
    console.log('[WS] connecting to:', wsUrl)
    this.ws = new WebSocket(wsUrl)
    this.ws.onopen = () => {
      console.log('[WS] connected')
      this._connected = true
      this.emit('_connected', {})
      // Send connect handshake
      this.sendHandshake()
    }
    this.ws.onclose = () => {
      console.log('[WS] disconnected')
      this._connected = false
      this._handshakeComplete = false
      this.ws = null
      this.emit('_disconnected', {})
      this.scheduleReconnect()
    }
    this.ws.onerror = (err) => {
      console.error('[WS] error:', err)
      this.ws?.close()
    }
    this.ws.onmessage = (e) => {
      try {
        const frame = JSON.parse(e.data)
        console.log('[WS] received:', frame.type, frame)
        // Backend protocol: type='res', id=string
        if (frame.type === 'res' && frame.id != null) {
          // Check if this is the handshake response
          if (!this._handshakeComplete) {
            this._handshakeComplete = true
            console.log('[WS] handshake complete')
          }
          const p = this.pending.get(frame.id)
          if (p) {
            this.pending.delete(frame.id)
            clearTimeout(p.timer)
            p.resolve({ ok: frame.ok, payload: frame.payload, error: frame.error })
          }
        } else if (frame.type === 'event') {
          this.emit(frame.event, frame.payload)
        }
      } catch (err) {
        console.error('[WS] onMessage error:', err, 'data:', e.data)
      }
    }
  }

  private sendHandshake() {
    if (!this.ws) return
    const id = String(++this.seq)
    // Backend requires: minProtocol, maxProtocol, client{id,version,platform,mode}, auth.token
    const connectParams = {
      minProtocol: 3,
      maxProtocol: 3,
      client: {
        id: 'control-ui-' + Date.now(),
        version: '1.0.0',
        platform: 'web',
        mode: 'operator'
      },
      role: 'operator',
      auth: {
        token: this.token
      }
    }
    console.log('[WS] sending handshake:', connectParams)
    this.ws.send(JSON.stringify({ 
      type: 'req', 
      id, 
      method: 'connect', 
      params: connectParams
    }))
  }

  disconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.ws?.close()
    this.ws = null
  }

  async call(method: string, params: Record<string, any> = {}): Promise<RPCResponse> {
    return new Promise((resolve) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        console.error('[WS] call failed: not connected')
        resolve({ ok: false, error: { code: -1, message: 'not connected' } })
        return
      }
      const id = String(++this.seq)
      console.log('[WS] sending request:', method, id, params)
      const timer = setTimeout(() => {
        console.error('[WS] request timeout:', method, id)
        this.pending.delete(id)
        resolve({ ok: false, error: { code: -2, message: 'timeout' } })
      }, 30000)
      this.pending.set(id, { resolve, timer })
      // Backend protocol: type=req, id=string
      this.ws.send(JSON.stringify({ type: 'req', id, method, params }))
    })
  }

  onEvent(handler: EventHandler) {
    this.eventHandlers.push(handler)
    return () => {
      this.eventHandlers = this.eventHandlers.filter(h => h !== handler)
    }
  }

  private emit(event: string, payload: any) {
    for (const h of this.eventHandlers) {
      try { h(event, payload) } catch {}
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, 3000)
  }
}

let clientInstance: GatewayClient | null = null

export function getClient(): GatewayClient {
  if (!clientInstance) {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${proto}//${location.host}/ws`
    // Try URL param first, then localStorage, then default token
    let token = new URLSearchParams(location.search).get('token')
    if (!token) {
      token = localStorage.getItem('gateway_token') || 'siriusec_claw_default_token_2026'
    }
    clientInstance = new GatewayClient(url, token)
    clientInstance.connect()
  }
  return clientInstance
}

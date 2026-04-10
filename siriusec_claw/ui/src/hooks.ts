import { useState, useEffect, useCallback } from 'react'
import type { RPCResponse } from './gateway'
import { getClient } from './gateway'

export function useGateway() {
  const [connected, setConnected] = useState(false)
  const client = getClient()

  useEffect(() => {
    // Poll for connection status since handshake happens async
    const checkConnected = () => {
      setConnected(client.connected)
    }
    checkConnected()
    const interval = setInterval(checkConnected, 100)
    
    const off = client.onEvent((event) => {
      if (event === '_connected') checkConnected()
      if (event === '_disconnected') setConnected(false)
    })
    
    return () => {
      clearInterval(interval)
      off()
    }
  }, [client])

  const call = useCallback(
    async (method: string, params: Record<string, any> = {}): Promise<RPCResponse> => {
      // Wait for handshake to complete
      let attempts = 0
      while (!client.connected && attempts < 50) {
        await new Promise(r => setTimeout(r, 100))
        attempts++
      }
      if (!client.connected) {
        return { ok: false, error: { code: -1, message: 'not connected' } }
      }
      return client.call(method, params)
    },
    [client]
  )

  return { connected, call, client }
}

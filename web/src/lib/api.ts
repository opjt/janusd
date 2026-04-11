export interface Secret {
  name: string
  namespace: string
  type: 'database' | 'external' | 'generated'
  db_type?: string
  rotation_days: number
  last_rotated_at: string | null
  status: string
  pods: string[]
  data?: Record<string, string>
}

export interface AuditLog {
  id: number
  target_id: number
  action: string
  actor: string
  result: string
  reason: string
  created_at: string
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init)
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error ?? 'unknown error')
  }
  return res.json()
}

export const api = {
  secrets: {
    list: () =>
      request<Secret[]>('/api/v1/secrets'),

    get: (namespace: string, name: string) =>
      request<Secret>(`/api/v1/secrets/${namespace}/${name}`),

    rotate: (namespace: string, name: string) =>
      request<{ status: string }>(`/api/v1/secrets/${namespace}/${name}/rotate`, {
        method: 'POST',
      }),
  },

  audit: {
    list: (params?: { namespace?: string; secret?: string }) => {
      const q = new URLSearchParams(params as Record<string, string>).toString()
      return request<AuditLog[]>(`/api/v1/audit${q ? `?${q}` : ''}`)
    },
  },
}

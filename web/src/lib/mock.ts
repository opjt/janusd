export type SecretType = 'database' | 'external' | 'generated'

export interface Secret {
  name: string
  namespace: string
  type: SecretType
  db_type: string | null
  rotation_days: number
  last_rotated_at: string | null
  pods: string[]
  data: Record<string, string>
}

export const secrets: Secret[] = [
  {
    name: 'torchi-db-secret',
    namespace: 'default',
    type: 'database',
    db_type: 'postgres',
    rotation_days: 30,
    last_rotated_at: '2026-03-12T00:00:00Z',
    pods: ['torchi-db-0', 'torchi-api'],
    data: {
      username: 'torchi_db',
      password: '••••••••••••••••',
      host: 'postgres.default.svc.cluster.local',
      port: '5432',
      dbname: 'torchi',
    },
  },
  {
    name: 'redis-secret',
    namespace: 'default',
    type: 'database',
    db_type: 'redis',
    rotation_days: 14,
    last_rotated_at: null,
    pods: ['redis-0'],
    data: {
      password: '••••••••••••••••',
      host: 'redis.default.svc.cluster.local',
      port: '6379',
    },
  },
  {
    name: 'stripe-secret',
    namespace: 'production',
    type: 'external',
    db_type: null,
    rotation_days: 90,
    last_rotated_at: '2025-12-01T00:00:00Z',
    pods: ['payment-api', 'billing-worker'],
    data: {
      api_key: '••••••••••••••••',
      webhook_secret: '••••••••••••••••',
    },
  },
]

export function daysUntilRotation(lastRotatedAt: string | null, rotationDays: number): number | null {
  if (!lastRotatedAt) return null
  const next = new Date(lastRotatedAt).getTime() + rotationDays * 86400_000
  return Math.ceil((next - Date.now()) / 86400_000)
}

export function rotationStatus(lastRotatedAt: string | null, rotationDays: number) {
  const days = daysUntilRotation(lastRotatedAt, rotationDays)
  if (days === null) return 'never'
  if (days < 0) return 'overdue'
  if (days <= 7) return 'soon'
  return 'ok'
}

export function formatDate(dateStr: string | null) {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric', month: 'short', day: 'numeric',
  })
}

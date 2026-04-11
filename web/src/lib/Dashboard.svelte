<script lang="ts">
  const workloads = [
    {
      id: 1,
      pod_name: 'torchi-db-0',
      namespace: 'default',
      secret_name: 'torchi-db-secret',
      type: 'database',
      db_type: 'postgres',
      status: 'active',
      rotation_days: 30,
      last_rotated_at: '2026-03-12T00:00:00Z',
    },
    {
      id: 2,
      pod_name: 'redis-0',
      namespace: 'default',
      secret_name: 'redis-secret',
      type: 'database',
      db_type: 'redis',
      status: 'active',
      rotation_days: 14,
      last_rotated_at: null,
    },
    {
      id: 3,
      pod_name: 'payment-api',
      namespace: 'production',
      secret_name: 'stripe-secret',
      type: 'manual',
      db_type: null,
      status: 'active',
      rotation_days: 90,
      last_rotated_at: '2025-12-01T00:00:00Z',
    },
  ]

  function statusBadge(status) {
    return status === 'active' ? 'badge-success' : 'badge-error'
  }

  function typeBadge(type) {
    const map = {
      database: 'badge-info',
      secret: 'badge-warning',
      manual: 'badge-ghost',
    }
    return map[type] ?? 'badge-ghost'
  }

  function formatDate(dateStr) {
    if (!dateStr) return '—'
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }
</script>

<div class="min-h-screen bg-base-200">
  <!-- Navbar -->
  <div class="navbar bg-base-100 shadow-sm px-6">
    <div class="flex-1">
      <span class="text-xl font-bold tracking-tight">karden</span>
      <span class="ml-2 text-sm text-base-content/50">secret lifecycle manager</span>
    </div>
  </div>

  <!-- Main content -->
  <div class="p-6 max-w-7xl mx-auto">
    <!-- Stats -->
    <div class="grid grid-cols-3 gap-4 mb-6">
      <div class="stat bg-base-100 rounded-box shadow-sm">
        <div class="stat-title">Managed Workloads</div>
        <div class="stat-value">{workloads.length}</div>
      </div>
      <div class="stat bg-base-100 rounded-box shadow-sm">
        <div class="stat-title">Active</div>
        <div class="stat-value text-success">
          {workloads.filter(w => w.status === 'active').length}
        </div>
      </div>
      <div class="stat bg-base-100 rounded-box shadow-sm">
        <div class="stat-title">Pending Rotation</div>
        <div class="stat-value text-warning">
          {workloads.filter(w => !w.last_rotated_at).length}
        </div>
      </div>
    </div>

    <!-- Table -->
    <div class="bg-base-100 rounded-box shadow-sm overflow-hidden">
      <div class="p-4 border-b border-base-200">
        <h2 class="font-semibold text-base">Workloads</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="table table-zebra">
          <thead>
            <tr>
              <th>Pod</th>
              <th>Namespace</th>
              <th>Secret</th>
              <th>Type</th>
              <th>Status</th>
              <th>Last Rotated</th>
              <th>Rotation Days</th>
            </tr>
          </thead>
          <tbody>
            {#each workloads as w}
              <tr class="cursor-pointer hover">
                <td class="font-mono text-sm">{w.pod_name}</td>
                <td class="text-sm">{w.namespace}</td>
                <td class="font-mono text-sm">{w.secret_name}</td>
                <td>
                  <span class="badge badge-sm {typeBadge(w.type)}">
                    {w.type}
                    {#if w.db_type}· {w.db_type}{/if}
                  </span>
                </td>
                <td>
                  <span class="badge badge-sm {statusBadge(w.status)}">
                    {w.status}
                  </span>
                </td>
                <td class="text-sm">{formatDate(w.last_rotated_at)}</td>
                <td class="text-sm">{w.rotation_days}d</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  </div>
</div>

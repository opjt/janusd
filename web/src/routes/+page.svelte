<script lang="ts">
  import { api, type Secret } from '$lib/api'
  import { daysUntilRotation, rotationStatus, formatDate } from '$lib/mock'
  import { Database, ExternalLink, RefreshCw, AlertTriangle, Eye, EyeOff, RotateCw, Clock, Box, Plus, Loader } from '@lucide/svelte'
  import * as Drawer from '$lib/components/ui/drawer/index.js'

  let secrets = $state<Secret[]>([])
  let loading = $state(true)
  let error = $state<string | null>(null)

  $effect(() => {
    api.secrets.list()
      .then(data => { secrets = data; loading = false })
      .catch(e => { error = e.message; loading = false })
  })

  let selected = $state<Secret | null>(null)
  let drawerOpen = $state(false)

  async function openDrawer(secret: Secret) {
    selected = secret
    drawerOpen = true
    try {
      selected = await api.secrets.get(secret.namespace, secret.name)
    } catch {}
  }

  let valueVisible = $state(false)
  let editMode = $state(false)
  let editValue = $state('')

  function startEdit() { editMode = true; editValue = '' }
  function cancelEdit() { editMode = false; editValue = '' }

  function statusBadgeClass(s: ReturnType<typeof rotationStatus>) {
    return { ok: 'badge-success', soon: 'badge-warning', overdue: 'badge-error', never: 'badge-error' }[s]
  }

  function statusLabel(lastRotatedAt: string | null, rotationDays: number) {
    const s = rotationStatus(lastRotatedAt, rotationDays)
    const days = daysUntilRotation(lastRotatedAt, rotationDays)
    if (s === 'never') return 'never rotated'
    if (s === 'overdue') return `${Math.abs(days!)}d overdue`
    return `${days}d left`
  }
</script>

<div class="bg-base-100 rounded-box overflow-hidden">
  <div class="px-5 py-4 border-b border-base-200 flex items-center justify-between">
    <h2 class="font-semibold text-sm">Secrets</h2>
    <button class="btn btn-primary btn-sm gap-2">
      <Plus size={13} />
      Add Secret
    </button>
  </div>

  {#if loading}
    <div class="flex items-center justify-center gap-2 py-16 text-base-content/40 text-sm">
      <Loader size={15} class="animate-spin" />
      Loading...
    </div>
  {:else if error}
    <div class="py-16 text-center text-error text-sm">{error}</div>
  {:else}
  <div class="overflow-x-auto">
    <table class="table table-zebra text-sm">
      <thead>
        <tr class="text-xs text-base-content/50">
          <th>Name</th>
          <th>Namespace</th>
          <th>Type</th>
          <th>Used by</th>
          <th>Last Rotated</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        {#each secrets as secret}
          {@const rStatus = rotationStatus(secret.last_rotated_at, secret.rotation_days)}
          <tr
            class="cursor-pointer hover"
            onclick={() => openDrawer(secret)}
          >
            <td class="font-mono font-medium">{secret.name}</td>
            <td class="text-base-content/50 text-xs">{secret.namespace}</td>
            <td>
              <div class="flex items-center gap-1.5 text-base-content/50">
                {#if secret.type === 'database'}
                  <Database size={13} />
                  <span class="text-xs">{secret.db_type}</span>
                {:else if secret.type === 'external'}
                  <ExternalLink size={13} />
                  <span class="text-xs">external</span>
                {:else}
                  <RefreshCw size={13} />
                  <span class="text-xs">generated</span>
                {/if}
              </div>
            </td>
            <td>
              <div class="flex items-center gap-2 flex-wrap">
                {#each secret.pods as pod}
                  <span class="font-mono text-xs text-base-content/60">{pod}</span>
                {/each}
              </div>
            </td>
            <td class="text-base-content/60">{formatDate(secret.last_rotated_at)}</td>
            <td>
              <div class="flex items-center gap-1.5">
                {#if rStatus === 'overdue' || rStatus === 'never'}
                  <AlertTriangle size={13} class="text-error" />
                {/if}
                <span class="badge {statusBadgeClass(rStatus)} badge-sm">
                  {statusLabel(secret.last_rotated_at, secret.rotation_days)}
                </span>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
  {/if}
</div>

<!-- Detail Drawer -->
<Drawer.Root bind:open={drawerOpen} direction="right">
  <Drawer.Content class="overflow-y-auto">
    {#if selected}
      <Drawer.Header class="mb-4">
        <Drawer.Title class="font-mono text-sm font-semibold">
          {selected.name}
        </Drawer.Title>
        <p class="text-xs text-base-content/40 mt-1 font-sans">
          {selected.namespace} · {selected.db_type ?? selected.type}
        </p>
      </Drawer.Header>

      <div class="space-y-4 px-10 pb-10">

        <!-- Data -->
        <div class="bg-base-200 rounded-lg divide-y divide-base-300">
          <div class="px-5 py-3 flex items-center justify-between">
            <span class="text-xs font-semibold text-base-content/50 uppercase tracking-wide">Data</span>
            <button class="btn btn-ghost btn-xs btn-circle" onclick={() => valueVisible = !valueVisible}>
              {#if valueVisible}<EyeOff size={13} />{:else}<Eye size={13} />{/if}
            </button>
          </div>
          {#each Object.entries(selected.data ?? {}) as [key, value]}
            <div class="px-5 py-3 flex flex-col gap-0.5">
              <span class="font-mono text-xs text-base-content/40">{key}</span>
              <span class="font-mono text-sm text-base-content/80 truncate">
                {valueVisible ? value : '••••••••••••••••'}
              </span>
            </div>
          {/each}
        </div>

        <!-- Rotation -->
        <div class="bg-base-200 rounded-lg divide-y divide-base-300">
          <div class="px-5 py-3 flex items-center justify-between">
            <span class="text-xs font-semibold text-base-content/50 uppercase tracking-wide">Rotation</span>
            {#if selected.type === 'database'}
              <button class="btn btn-ghost btn-xs gap-1"><RotateCw size={11} />Rotate now</button>
            {/if}
          </div>
          <div class="px-5 py-4 grid grid-cols-2 gap-6 text-sm">
            <div>
              <div class="text-xs text-base-content/40 mb-1">Last rotated</div>
              <div>{formatDate(selected.last_rotated_at)}</div>
            </div>
            <div>
              <div class="text-xs text-base-content/40 mb-1">Interval</div>
              <div>Every {selected.rotation_days} days</div>
            </div>
          </div>
        </div>

        <!-- Used by -->
        <div class="bg-base-200 rounded-lg divide-y divide-base-300">
          <div class="px-5 py-3">
            <span class="text-xs font-semibold text-base-content/50 uppercase tracking-wide">Used by</span>
          </div>
          <ul class="divide-y divide-base-300">
            {#each selected.pods as pod}
              <li class="flex items-center gap-3 px-5 py-3">
                <Box size={13} class="text-base-content/40" />
                <span class="font-mono text-sm">{pod}</span>
              </li>
            {/each}
          </ul>
        </div>

        <!-- History -->
        {#await api.audit.list({ secret: selected.name, namespace: selected.namespace })}
          <div class="flex items-center gap-2 py-6 px-5 text-base-content/40 text-xs">
            <Loader size={13} class="animate-spin" /> Loading history...
          </div>
        {:then logs}
          {#if logs.length > 0}
            <div class="bg-base-200 rounded-lg divide-y divide-base-300">
              <div class="px-5 py-3">
                <span class="text-xs font-semibold text-base-content/50 uppercase tracking-wide">History</span>
              </div>
              <ul class="divide-y divide-base-300">
                {#each logs as entry}
                  <li class="flex items-center gap-3 px-5 py-3">
                    <Clock size={12} class="text-base-content/30 shrink-0" />
                    <span class="text-xs text-base-content/50 w-32 shrink-0">{formatDate(entry.created_at)}</span>
                    <span class="font-mono text-xs text-base-content/60">{entry.actor}</span>
                    <span class="text-xs text-base-content/40 ml-auto">{entry.action}</span>
                  </li>
                {/each}
              </ul>
            </div>
          {/if}
        {:catch}
          <div class="flex items-center gap-2 py-6 px-5 text-base-content/40 text-xs">
            <AlertTriangle size={13} class="text-error" /> Failed to load history
          </div>
        {/await}

      </div>
    {/if}
  </Drawer.Content>
</Drawer.Root>

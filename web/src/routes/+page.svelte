<script lang="ts">
  import { secrets, rotationStatus, daysUntilRotation, formatDate, type Secret } from '$lib/mock'
  import { Database, ExternalLink, RefreshCw, AlertTriangle, Eye, EyeOff, RotateCw, Clock, Box, KeyRound } from '@lucide/svelte'
  import * as Drawer from '$lib/components/ui/drawer/index.js'

  let selected = $state<Secret | null>(null)
  let drawerOpen = $state(false)

  function openDrawer(secret: Secret) {
    selected = secret
    drawerOpen = true
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

  const history = [
    { date: '2026-03-12T00:00:00Z', actor: 'karden', note: 'auto-rotated' },
    { date: '2025-12-01T00:00:00Z', actor: 'opjt',   note: 'manually set' },
  ]
</script>

<div class="bg-base-100 rounded-box shadow-sm overflow-hidden">
  <div class="px-5 py-4 border-b border-base-200 flex items-center justify-between">
    <h2 class="font-semibold text-sm">Secrets</h2>
    <button class="btn btn-primary btn-sm gap-2">
      <KeyRound size={13} />
      Add Secret
    </button>
  </div>

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
            <td>
              <span class="badge badge-ghost badge-sm">{secret.namespace}</span>
            </td>
            <td>
              <div class="flex items-center gap-1.5">
                {#if secret.type === 'database'}
                  <Database size={13} class="text-info" />
                  <span class="badge badge-info badge-sm">{secret.db_type}</span>
                {:else if secret.type === 'external'}
                  <ExternalLink size={13} class="text-warning" />
                  <span class="badge badge-warning badge-sm">external</span>
                {:else}
                  <RefreshCw size={13} class="text-success" />
                  <span class="badge badge-success badge-sm">generated</span>
                {/if}
              </div>
            </td>
            <td>
              <div class="flex items-center gap-1 flex-wrap">
                {#each secret.pods as pod}
                  <span class="badge badge-outline badge-sm font-mono">{pod}</span>
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
</div>

<!-- Detail Drawer -->
<Drawer.Root bind:open={drawerOpen} direction="right">
  <Drawer.Content class="overflow-y-auto">
    {#if selected}
      <Drawer.Header class="mb-4">
        <Drawer.Title class="font-mono text-sm font-semibold flex items-center gap-2 flex-wrap">
          {selected.name}
          <span class="badge badge-ghost badge-sm font-sans font-normal">{selected.namespace}</span>
          {#if selected.db_type}
            <span class="badge badge-info badge-sm font-sans font-normal">{selected.db_type}</span>
          {:else}
            <span class="badge badge-ghost badge-sm font-sans font-normal">{selected.type}</span>
          {/if}
        </Drawer.Title>
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
          {#each Object.entries(selected.data) as [key, value]}
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
        <div class="bg-base-200 rounded-lg divide-y divide-base-300">
          <div class="px-5 py-3">
            <span class="text-xs font-semibold text-base-content/50 uppercase tracking-wide">History</span>
          </div>
          <ul class="divide-y divide-base-300">
            {#each history as entry}
              <li class="flex items-center gap-3 px-5 py-3">
                <Clock size={12} class="text-base-content/30 shrink-0" />
                <span class="text-xs text-base-content/50 w-32 shrink-0">{formatDate(entry.date)}</span>
                <span class="font-mono text-xs text-base-content/60">{entry.actor}</span>
                <span class="text-xs text-base-content/40 ml-auto">{entry.note}</span>
              </li>
            {/each}
          </ul>
        </div>

      </div>
    {/if}
  </Drawer.Content>
</Drawer.Root>

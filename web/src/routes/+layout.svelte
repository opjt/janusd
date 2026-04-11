<script lang="ts">
  import '../app.css'
  import { page } from '$app/stores'
  import { KeyRound, Moon, Sun, ScrollText, Settings } from '@lucide/svelte'

  let theme = $state<'karden' | 'karden-dark'>('karden')

  function toggleTheme() {
    theme = theme === 'karden' ? 'karden-dark' : 'karden'
    document.documentElement.setAttribute('data-theme', theme)
  }

  let { children } = $props()
</script>

<div class="min-h-screen flex flex-col bg-base-200">

  <!-- Topbar -->
  <header class="bg-base-100 border-b border-base-200 sticky top-0 z-50">
    <div class="max-w-6xl mx-auto px-6 h-14 flex items-center gap-6">

      <!-- Logo -->
      <a href="/" class="flex items-center gap-2 shrink-0">
        <!-- <KeyRound class="text-primary" size={17} /> -->
        <span class="font-bold tracking-tight text-sm">karden</span>
      </a>

      <!-- Nav -->
      <nav class="flex items-center gap-1 flex-1">
        <a
          href="/"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors
            {$page.url.pathname === '/'
              ? 'bg-primary/10 text-primary font-medium'
              : 'text-base-content/70 hover:bg-base-200'}"
        >
          Secrets
        </a>

        <a
          href="/audit"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors
            {$page.url.pathname === '/audit'
              ? 'bg-primary/10 text-primary font-medium'
              : 'text-base-content/70 hover:bg-base-200'}"
        >
          <ScrollText size={14} />
          Audit Log
        </a>
      </nav>

      <!-- Right: theme toggle + settings -->
      <div class="flex items-center gap-1">
        <button
          class="btn btn-ghost btn-sm btn-circle"
          onclick={toggleTheme}
          aria-label="Toggle theme"
        >
          {#if theme === 'karden'}
            <Moon size={15} />
          {:else}
            <Sun size={15} />
          {/if}
        </button>

        <a
          href="/settings"
          class="btn btn-ghost btn-sm btn-circle {$page.url.pathname === '/settings' ? 'text-primary' : ''}"
          aria-label="Settings"
        >
          <Settings size={15} />
        </a>
      </div>

    </div>
  </header>

  <!-- Page content -->
  <main class="flex-1 p-6 max-w-6xl w-full mx-auto">
    {@render children()}
  </main>

</div>

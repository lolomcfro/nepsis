<script lang="ts">
  import { getApps, getKnownStores, hideApp, showApp, uninstallApp } from '../lib/wails'
  import type { App } from '../lib/wails'

  export let connected: boolean

  let apps: App[] = []
  let knownStores = new Set<string>()
  let loading = false
  let error = ''
  let search = ''
  let acting = new Set<string>()
  let confirmDelete: App | null = null
  let deleting = false

  $: storeApps = apps.filter(a => knownStores.has(a.package))
  $: regularApps = apps.filter(a => !knownStores.has(a.package))

  $: filteredStoreApps = storeApps.filter(a =>
    a.label.toLowerCase().includes(search.toLowerCase()) ||
    a.package.toLowerCase().includes(search.toLowerCase())
  )
  $: filteredRegularApps = regularApps.filter(a =>
    a.label.toLowerCase().includes(search.toLowerCase()) ||
    a.package.toLowerCase().includes(search.toLowerCase())
  )

  $: totalFiltered = filteredStoreApps.length + filteredRegularApps.length

  let visibilityFilter: 'all' | 'visible' | 'hidden' = 'all'

  $: visibleFilteredRegularApps = filteredRegularApps.filter(a => {
    if (visibilityFilter === 'visible') return !a.hidden
    if (visibilityFilter === 'hidden') return a.hidden
    return true
  })

  $: visibleFilteredStoreApps = filteredStoreApps.filter(a => {
    if (visibilityFilter === 'visible') return !a.hidden
    if (visibilityFilter === 'hidden') return a.hidden
    return true
  })

  function cycleFilter() {
    if (visibilityFilter === 'all') visibilityFilter = 'visible'
    else if (visibilityFilter === 'visible') visibilityFilter = 'hidden'
    else visibilityFilter = 'all'
  }

  async function load() {
    if (!connected) return
    loading = true
    error = ''
    try {
      const [fetchedApps, stores] = await Promise.all([getApps(), getKnownStores()])
      apps = fetchedApps
      knownStores = new Set(stores)
    } catch (e: any) {
      error = e?.message ?? String(e)
    } finally {
      loading = false
    }
  }

  async function hide(app: App) {
    if (acting.has(app.package)) return
    acting = new Set([...acting, app.package])
    error = ''
    try {
      await hideApp(app.package)
      app.hidden = true
      apps = [...apps]
    } catch (e: any) {
      error = `Failed to hide ${app.label}: ${e?.message ?? e}`
    } finally {
      acting = new Set([...acting].filter(p => p !== app.package))
    }
  }

  async function show(app: App) {
    if (acting.has(app.package)) return
    acting = new Set([...acting, app.package])
    error = ''
    try {
      await showApp(app.package)
      app.hidden = false
      const fresh = await getApps()
      const updated = fresh.find(a => a.package === app.package)
      if (updated) app.icon = updated.icon
      apps = [...apps]
    } catch (e: any) {
      error = `Failed to show ${app.label}: ${e?.message ?? e}`
    } finally {
      acting = new Set([...acting].filter(p => p !== app.package))
    }
  }

  async function doDelete() {
    if (!confirmDelete || deleting) return
    deleting = true
    error = ''
    const app = confirmDelete
    try {
      await uninstallApp(app.package)
      apps = apps.filter(a => a.package !== app.package)
      confirmDelete = null
    } catch (e: any) {
      error = `Failed to delete ${app.label}: ${e?.message ?? e}`
    } finally {
      deleting = false
    }
  }

  $: if (connected) load()
</script>

<div class="apps-tab">
  <div class="tab-hero">
    <div class="hero-title">Manage Apps</div>
    <div class="hero-controls">
      <div class="search-wrap">
        <svg class="search-icon" width="12" height="12" viewBox="0 0 12 12" fill="none">
          <circle cx="5" cy="5" r="3.5" stroke="currentColor" stroke-width="1.3"/>
          <path d="M8 8l2 2" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
        </svg>
        <input
          class="search-input"
          type="text"
          placeholder="Search apps…"
          bind:value={search}
        />
      </div>
      <button class="filter-pill" on:click={cycleFilter}>
        {visibilityFilter === 'all' ? 'All' : visibilityFilter === 'visible' ? 'Visible' : 'Hidden'}
      </button>
      <button class="refresh-btn" on:click={load} disabled={loading || !connected}>
        {loading ? 'Loading…' : 'Refresh'}
      </button>
    </div>
  </div>

  {#if error}
    <div class="error-banner">
      {error}
      <button on:click={load}>Retry</button>
    </div>
  {/if}

  {#if loading}
    <p class="hint">Loading apps from phone…</p>
  {:else if totalFiltered === 0 && apps.length > 0}
    <p class="hint">No apps match "{search}"</p>
  {:else if apps.length === 0}
    <p class="hint">No apps found.</p>
  {:else}

    {#if visibleFilteredStoreApps.length > 0}
      <div class="section-header">App Stores — Auto-hidden</div>
      <ul class="app-list">
        {#each visibleFilteredStoreApps as app (app.package)}
          <li class="app-row" class:hidden-app={app.hidden}>
            <div class="app-icon-placeholder">{app.label.charAt(0).toUpperCase()}</div>
            <span class="app-name">{app.label}</span>
            <span class="store-tag">Store</span>
            <span class="badge" class:badge-visible={!app.hidden} class:badge-hidden={app.hidden}>
              {app.hidden ? 'Hidden' : 'Visible'}
            </span>
            {#if app.hidden}
              <button
                class="btn-show"
                disabled={acting.has(app.package)}
                on:click={() => show(app)}
              >
                {acting.has(app.package) ? '…' : 'Show'}
              </button>
            {:else}
              <button
                class="btn-hide"
                disabled={acting.has(app.package)}
                on:click={() => hide(app)}
              >
                {acting.has(app.package) ? '…' : 'Hide'}
              </button>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}

    {#if visibleFilteredRegularApps.length > 0}
      <div class="section-header">Apps</div>
      <ul class="app-list">
        {#each visibleFilteredRegularApps as app (app.package)}
          <li class="app-row" class:hidden-app={app.hidden}>
            <div class="app-icon-placeholder">{app.label.charAt(0).toUpperCase()}</div>
            <span class="app-name">{app.label}</span>
            <span class="badge" class:badge-visible={!app.hidden} class:badge-hidden={app.hidden}>
              {app.hidden ? 'Hidden' : 'Visible'}
            </span>
            {#if app.hidden}
              <button
                class="btn-show"
                disabled={acting.has(app.package)}
                on:click={() => show(app)}
              >
                {acting.has(app.package) ? '…' : 'Show'}
              </button>
            {:else}
              <button
                class="btn-hide"
                disabled={acting.has(app.package)}
                on:click={() => hide(app)}
              >
                {acting.has(app.package) ? '…' : 'Hide'}
              </button>
            {/if}
            <button
              class="btn-delete"
              disabled={acting.has(app.package)}
              on:click={() => confirmDelete = app}
            >
              Delete
            </button>
          </li>
        {/each}
      </ul>
    {/if}

  {/if}
</div>

{#if confirmDelete}
  <div class="modal-overlay" on:click|self={() => { if (!deleting) confirmDelete = null }}>
    <div class="modal-dialog">
      <p class="modal-title">Are you sure you want to delete {confirmDelete.label}?</p>
      <p class="modal-body">
        This will completely uninstall the app from the device.<br />
        It cannot be restored without reinstalling.
      </p>
      <div class="modal-actions">
        <button
          class="modal-cancel"
          disabled={deleting}
          on:click={() => confirmDelete = null}
        >
          Cancel
        </button>
        <button
          class="modal-delete"
          disabled={deleting}
          on:click={doDelete}
        >
          {deleting ? 'Deleting…' : 'Delete'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .apps-tab { max-width: 700px; }

  .tab-hero {
    background: linear-gradient(135deg, var(--bg-hero-start), var(--bg-hero-end));
    padding: 16px 18px 14px;
    border-bottom: 1px solid var(--border-hero);
    flex-shrink: 0;
  }
  .hero-title {
    font-size: 15px;
    font-weight: 800;
    letter-spacing: -0.3px;
    color: var(--text-primary);
    margin-bottom: 10px;
  }
  .hero-controls {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .search-wrap {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 6px;
    background: rgba(255,255,255,0.6);
    border: 1px solid var(--border-hero);
    border-radius: 7px;
    height: 28px;
    padding: 0 10px;
    color: var(--text-muted);
  }
  :global([data-theme="dark"]) .search-wrap {
    background: rgba(255,255,255,0.05);
  }
  .search-icon { flex-shrink: 0; }
  .search-input {
    border: none;
    background: transparent;
    font-size: 12px;
    color: var(--text-primary);
    outline: none;
    width: 100%;
  }
  .search-input::placeholder { color: var(--text-muted); }
  .filter-pill {
    height: 28px;
    padding: 0 12px;
    border: 1px solid var(--border-hero);
    border-radius: 14px;
    background: var(--accent-bg);
    color: var(--accent);
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
  }
  .refresh-btn {
    height: 28px;
    padding: 0 12px;
    background: #1f1f2e;
    border: 1px solid #2a2a38;
    border-radius: 6px;
    color: #6b7280;
    cursor: pointer;
    font-size: 12px;
    white-space: nowrap;
    transition: background 0.15s;
  }
  .refresh-btn:hover:not(:disabled) { background: #2a2a3e; }
  .refresh-btn:disabled { opacity: 0.5; cursor: default; }

  .error-banner {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: #1f1515;
    border: 1px solid #7f1d1d;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
    margin-bottom: 16px;
  }
  .error-banner button {
    padding: 4px 10px;
    background: transparent;
    border: 1px solid #7f1d1d;
    border-radius: 4px;
    color: #f87171;
    cursor: pointer;
    font-size: 12px;
  }
  .error-banner button:hover { background: #2a1515; }

  .hint { color: #4b5563; font-size: 14px; margin-top: 8px; }

  .section-header {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #4b5563;
    margin: 16px 0 6px;
  }

  .app-list { list-style: none; padding: 0; margin: 0; }
  .app-row {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 36px;
    padding: 0 14px;
    border-bottom: 1px solid var(--border);
    transition: opacity 0.12s;
  }
  .app-row.hidden-app { opacity: 0.55; }
  .app-icon-placeholder {
    width: 20px;
    height: 20px;
    border-radius: 5px;
    background: var(--accent-bg);
    color: var(--accent);
    font-size: 10px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .app-name {
    flex: 1;
    font-size: 12px;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .badge {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 7px;
    border-radius: 10px;
    flex-shrink: 0;
  }
  .badge-visible {
    background: #f0fdf4;
    color: #166534;
    border: 1px solid #bbf7d0;
  }
  .badge-hidden {
    background: #fef3c7;
    color: #92400e;
    border: 1px solid #fde68a;
  }
  :global([data-theme="dark"]) .badge-visible {
    background: #052e16;
    color: #4ade80;
    border-color: #166534;
  }
  :global([data-theme="dark"]) .badge-hidden {
    background: #451a03;
    color: #fbbf24;
    border-color: #78350f;
  }

  .store-tag {
    font-size: 10px;
    background: #1e1b4b;
    color: #a78bfa;
    border: 1px solid #312e81;
    border-radius: 4px;
    padding: 2px 6px;
    flex-shrink: 0;
  }

  .btn-hide, .btn-show, .btn-delete {
    padding: 5px 12px;
    border-radius: 5px;
    font-size: 13px;
    cursor: pointer;
    flex-shrink: 0;
    transition: background 0.12s;
  }
  .btn-hide:disabled, .btn-show:disabled, .btn-delete:disabled {
    opacity: 0.5;
    cursor: default;
  }

  .btn-hide {
    background: #1f1f2e;
    border: 1px solid #2a2a38;
    color: #9ca3af;
  }
  .btn-hide:hover:not(:disabled) { background: #2a2a3e; }

  .btn-show {
    background: #1a2a1f;
    border: 1px solid #166534;
    color: #4ade80;
  }
  .btn-show:hover:not(:disabled) { background: #1f3328; }

  .btn-delete {
    background: #1f1515;
    border: 1px solid #7f1d1d;
    color: #f87171;
  }
  .btn-delete:hover:not(:disabled) { background: #2a1515; }

  /* Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }
  .modal-dialog {
    background: #16161e;
    border: 1px solid #2a2a38;
    border-radius: 10px;
    padding: 28px 28px 24px;
    max-width: 380px;
    width: 90%;
  }
  .modal-title {
    font-size: 15px;
    font-weight: 600;
    color: #e2e2e8;
    margin-bottom: 12px;
  }
  .modal-body {
    font-size: 13px;
    color: #9ca3af;
    line-height: 1.6;
    margin-bottom: 24px;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }
  .modal-cancel {
    padding: 8px 18px;
    background: #1f1f2e;
    border: 1px solid #2a2a38;
    border-radius: 6px;
    color: #9ca3af;
    font-size: 14px;
    cursor: pointer;
    transition: background 0.12s;
  }
  .modal-cancel:hover:not(:disabled) { background: #2a2a3e; }
  .modal-cancel:disabled { opacity: 0.5; cursor: default; }

  .modal-delete {
    padding: 8px 18px;
    background: #7f1d1d;
    border: none;
    border-radius: 6px;
    color: #f87171;
    font-size: 14px;
    cursor: pointer;
    transition: background 0.12s;
  }
  .modal-delete:hover:not(:disabled) { background: #991b1b; }
  .modal-delete:disabled { opacity: 0.5; cursor: default; }
</style>

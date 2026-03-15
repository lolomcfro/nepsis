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
  <div class="toolbar">
    <input
      type="search"
      placeholder="Search apps…"
      bind:value={search}
    />
    <button on:click={load} disabled={loading || !connected}>
      {loading ? 'Loading…' : 'Refresh'}
    </button>
  </div>

  {#if error}
    <div class="error-banner">
      {error}
      <button on:click={load}>Retry</button>
    </div>
  {/if}

  {#if loading}
    <p class="hint">Loading apps from phone… (may take up to 10 seconds)</p>
  {:else if totalFiltered === 0 && apps.length > 0}
    <p class="hint">No apps match "{search}"</p>
  {:else if apps.length === 0}
    <p class="hint">No apps found.</p>
  {:else}

    {#if filteredStoreApps.length > 0}
      <div class="section-header">App Stores — Auto-hidden</div>
      <ul class="app-list">
        {#each filteredStoreApps as app (app.package)}
          <li class="app-item" class:faded={app.hidden}>
            {#if app.icon}
              <img src="data:image/png;base64,{app.icon}" alt="" class="app-icon" />
            {:else}
              <div class="app-icon placeholder"></div>
            {/if}
            <div class="app-info">
              <span class="app-label">{app.label}</span>
              <span class="app-package">{app.package}</span>
            </div>
            <span class="store-tag">Store</span>
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

    {#if filteredRegularApps.length > 0}
      <div class="section-header">Apps</div>
      <ul class="app-list">
        {#each filteredRegularApps as app (app.package)}
          <li class="app-item" class:faded={app.hidden}>
            {#if app.icon}
              <img src="data:image/png;base64,{app.icon}" alt="" class="app-icon" />
            {:else}
              <div class="app-icon placeholder"></div>
            {/if}
            <div class="app-info">
              <span class="app-label">{app.label}</span>
              <span class="app-package">{app.package}</span>
            </div>
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

  .toolbar { display: flex; gap: 12px; margin-bottom: 16px; }
  .toolbar input {
    flex: 1;
    padding: 8px 12px;
    background: #1f1f2e;
    border: 1px solid #2a2a38;
    border-radius: 6px;
    font-size: 14px;
    color: #e2e2e8;
    outline: none;
    transition: border-color 0.15s;
  }
  .toolbar input:focus { border-color: #a78bfa; }
  .toolbar input::placeholder { color: #4b5563; }
  .toolbar button {
    padding: 8px 16px;
    background: #1f1f2e;
    border: 1px solid #2a2a38;
    border-radius: 6px;
    color: #6b7280;
    cursor: pointer;
    font-size: 14px;
    transition: background 0.15s;
  }
  .toolbar button:hover:not(:disabled) { background: #2a2a3e; }
  .toolbar button:disabled { opacity: 0.5; cursor: default; }

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
  .app-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 10px;
    border-radius: 6px;
    border-bottom: 1px solid #2a2a38;
    transition: background 0.12s, opacity 0.2s;
  }
  .app-item:hover { background: #1f1f2e; }
  .app-item.faded { opacity: 0.45; }

  .app-icon { width: 40px; height: 40px; border-radius: 8px; object-fit: cover; flex-shrink: 0; }
  .app-icon.placeholder { background: #2a2a38; }

  .app-info { flex: 1; min-width: 0; }
  .app-label { display: block; font-size: 14px; font-weight: 500; color: #e2e2e8; }
  .app-package { display: block; font-size: 11px; color: #4b5563; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

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

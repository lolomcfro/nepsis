<script lang="ts">
  import { onMount } from 'svelte'
  import { getApps, hideApp, showApp } from '../lib/wails'
  import type { App } from '../lib/wails'

  export let connected: boolean

  let apps: App[] = []
  let loading = false
  let error = ''
  let search = ''
  let toggling = new Set<string>()

  $: filtered = apps.filter(a =>
    a.label.toLowerCase().includes(search.toLowerCase()) ||
    a.package.toLowerCase().includes(search.toLowerCase())
  )

  async function load() {
    if (!connected) return
    loading = true
    error = ''
    try {
      apps = await getApps()
    } catch (e: any) {
      error = e?.message ?? String(e)
    } finally {
      loading = false
    }
  }

  async function toggle(app: App) {
    if (toggling.has(app.package)) return
    toggling = new Set([...toggling, app.package])
    error = ''
    try {
      if (app.hidden) {
        await showApp(app.package)
        app.hidden = false
      } else {
        await hideApp(app.package)
        app.hidden = true
      }
      apps = [...apps]
    } catch (e: any) {
      error = `Failed to toggle ${app.label}: ${e?.message ?? e}`
    } finally {
      toggling.delete(app.package)
      toggling = new Set(toggling)
    }
  }

  onMount(load)
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
    <div class="error-banner">{error} <button on:click={load}>Retry</button></div>
  {/if}

  {#if loading}
    <p class="hint">Loading apps from phone… (may take up to 10 seconds)</p>
  {:else if filtered.length === 0 && apps.length > 0}
    <p class="hint">No apps match "{search}"</p>
  {:else if apps.length === 0}
    <p class="hint">No apps found.</p>
  {:else}
    <ul class="app-list">
      {#each filtered as app (app.package)}
        <li class="app-item" class:hidden={app.hidden}>
          {#if app.icon}
            <img src="data:image/png;base64,{app.icon}" alt="" class="app-icon" />
          {:else}
            <div class="app-icon placeholder"></div>
          {/if}
          <div class="app-info">
            <span class="app-label">{app.label}</span>
            <span class="app-package">{app.package}</span>
          </div>
          <label class="toggle" title={app.hidden ? 'Hidden — click to show' : 'Visible — click to hide'}>
            <input
              type="checkbox"
              checked={!app.hidden}
              disabled={toggling.has(app.package)}
              on:change={() => toggle(app)}
            />
            <span class="slider"></span>
          </label>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .apps-tab { max-width: 700px; }
  .toolbar { display: flex; gap: 12px; margin-bottom: 16px; }
  .toolbar input {
    flex: 1; padding: 8px 12px;
    border: 1px solid #ddd; border-radius: 4px;
    font-size: 14px; outline: none;
  }
  .toolbar input:focus { border-color: #1976d2; }
  .toolbar button {
    padding: 8px 16px; background: #f5f5f5;
    border: 1px solid #ddd; border-radius: 4px; cursor: pointer;
  }
  .toolbar button:disabled { opacity: 0.5; cursor: default; }

  .hint { color: #888; font-size: 14px; margin-top: 8px; }

  .app-list { list-style: none; padding: 0; margin: 0; }
  .app-item {
    display: flex; align-items: center; gap: 12px;
    padding: 10px 0; border-bottom: 1px solid #f0f0f0;
    transition: opacity 0.2s;
  }
  .app-item.hidden { opacity: 0.45; }
  .app-icon { width: 40px; height: 40px; border-radius: 8px; object-fit: cover; flex-shrink: 0; }
  .app-icon.placeholder { background: #e0e0e0; }
  .app-info { flex: 1; min-width: 0; }
  .app-label { display: block; font-size: 14px; font-weight: 500; }
  .app-package { display: block; font-size: 11px; color: #9e9e9e; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

  .toggle { position: relative; display: inline-block; width: 44px; height: 24px; cursor: pointer; flex-shrink: 0; }
  .toggle input { opacity: 0; width: 0; height: 0; position: absolute; }
  .slider {
    position: absolute; inset: 0;
    background: #ccc; border-radius: 24px; transition: background 0.2s;
  }
  .slider::before {
    content: ''; position: absolute;
    height: 18px; width: 18px; left: 3px; bottom: 3px;
    background: white; border-radius: 50%; transition: transform 0.2s;
  }
  input:checked ~ .slider { background: #1976d2; }
  input:checked ~ .slider::before { transform: translateX(20px); }
  input:disabled ~ .slider { opacity: 0.5; cursor: default; }

  .error-banner {
    padding: 12px 16px; background: #ffebee;
    border-radius: 4px; color: #c62828;
    margin-bottom: 16px; display: flex; align-items: center; gap: 12px;
  }
  .error-banner button { cursor: pointer; }
</style>

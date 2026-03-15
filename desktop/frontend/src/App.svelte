<script lang="ts">
  import { onMount } from 'svelte'
  import { getConnectionStatus, isDeviceOwnerInstalled, onConnectionChange, onAdminVersionMismatch, updateAdmin, getKnownStores, hideApp } from './lib/wails'
  import type { ConnectionStatus } from './lib/wails'
  import SetupTab from './components/SetupTab.svelte'
  import AppsTab from './components/AppsTab.svelte'
  import InstallTab from './components/InstallTab.svelte'

  let activeTab = 'apps'
  let connected = false
  let serial = ''
  let deviceOwnerInstalled = false

  let versionMismatch: { installedVersion: number, bundledVersion: number } | null = null
  let updateState: 'idle' | 'updating' | 'success' | 'error' = 'idle'
  let updateError = ''

  async function handleUpdate() {
    updateState = 'updating'
    try {
      await updateAdmin()
      updateState = 'success'
      setTimeout(() => { versionMismatch = null; updateState = 'idle' }, 2000)
    } catch (e: any) {
      updateError = e?.message ?? String(e)
      updateState = 'error'
    }
  }

  async function handleSetupComplete() {
    deviceOwnerInstalled = true
    activeTab = 'apps'
    // Auto-hide known stores (fire-and-forget, ignore errors)
    try {
      const stores = await getKnownStores()
      await Promise.allSettled(stores.map(pkg => hideApp(pkg)))
    } catch {}
  }

  onMount(async () => {
    const status = await getConnectionStatus()
    connected = status.connected
    serial = status.serial

    if (connected) {
      deviceOwnerInstalled = await isDeviceOwnerInstalled()
      activeTab = deviceOwnerInstalled ? 'apps' : 'setup'
    }

    onAdminVersionMismatch((info) => {
      versionMismatch = info
      updateState = 'idle'
    })

    onConnectionChange(async (status: ConnectionStatus) => {
      connected = status.connected
      serial = status.serial
      if (connected) {
        deviceOwnerInstalled = await isDeviceOwnerInstalled()
        activeTab = deviceOwnerInstalled ? 'apps' : 'setup'
      } else {
        deviceOwnerInstalled = false
      }
    })
  })
</script>

<div class="app">
  <div class="titlebar">
    <div class="brand">
      <span class="brand-icon">N</span>
      <span class="brand-name">nepsis</span>
    </div>
    <div class="conn-badge">
      <span class="conn-dot" class:connected={connected}></span>
      {connected ? serial : 'Not connected'}
    </div>
  </div>

  <nav class="tabs">
    <button class:active={activeTab === 'setup'} on:click={() => activeTab = 'setup'}>
      Setup
    </button>
    <button
      class:active={activeTab === 'apps'}
      on:click={() => activeTab = 'apps'}
      disabled={!connected}
    >
      Apps
    </button>
    <button
      class:active={activeTab === 'install'}
      on:click={() => activeTab = 'install'}
      disabled={!connected}
    >
      Install
    </button>
  </nav>

  {#if versionMismatch}
    <div class="update-banner">
      {#if updateState === 'idle'}
        SoberAdmin update available (v{versionMismatch.installedVersion} → v{versionMismatch.bundledVersion})
        <button on:click={handleUpdate}>Update</button>
        <button class="dismiss" on:click={() => versionMismatch = null}>✕</button>
      {:else if updateState === 'updating'}
        Updating SoberAdmin…
      {:else if updateState === 'success'}
        SoberAdmin updated successfully.
      {:else if updateState === 'error'}
        Update failed: {updateError}
        <button on:click={handleUpdate}>Retry</button>
        <button class="dismiss" on:click={() => versionMismatch = null}>✕</button>
      {/if}
    </div>
  {/if}

  <main class="content">
    {#if activeTab === 'setup'}
      <SetupTab {connected} {deviceOwnerInstalled} on:setupcomplete={handleSetupComplete} />
    {:else if activeTab === 'apps'}
      <AppsTab {connected} />
    {:else if activeTab === 'install'}
      <InstallTab {connected} />
    {/if}
  </main>


</div>

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) { font-family: system-ui, -apple-system, sans-serif; background: #0f0f13; color: #e2e2e8; }

  .app { display: flex; flex-direction: column; height: 100vh; background: #16161e; }

  /* Title bar */
  .titlebar {
    display: flex;
    align-items: center;
    padding: 0 16px;
    height: 44px;
    background: #16161e;
    border-bottom: 1px solid #2a2a38;
    flex-shrink: 0;
  }
  .brand { display: flex; align-items: center; gap: 8px; }
  .brand-icon {
    background: linear-gradient(135deg, #7c6af7, #a78bfa);
    border-radius: 7px;
    width: 22px;
    height: 22px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 700;
    color: white;
  }
  .brand-name { font-size: 15px; font-weight: 600; color: #e2e2e8; }

  .conn-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: #6b7280;
    background: #1f1f2e;
    border: 1px solid #2a2a38;
    border-radius: 20px;
    padding: 4px 10px;
    margin-left: auto;
  }
  .conn-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: #4b5563;
    flex-shrink: 0;
  }
  .conn-dot.connected {
    background: #22c55e;
    box-shadow: 0 0 6px #22c55e88;
  }

  /* Tabs */
  .tabs { display: flex; border-bottom: 1px solid #2a2a38; padding: 0 16px; background: #1a1a24; flex-shrink: 0; }
  .tabs button {
    padding: 12px 20px;
    border: none;
    background: none;
    cursor: pointer;
    font-size: 14px;
    color: #6b7280;
    border-bottom: 2px solid transparent;
    transition: color 0.15s, border-color 0.15s;
  }
  .tabs button.active { color: #a78bfa; border-bottom-color: #a78bfa; }
  .tabs button:disabled { opacity: 0.4; cursor: default; }

  /* Content */
  .content { flex: 1; overflow-y: auto; padding: 24px; }

  /* Update banner */
  .update-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: #1a1a2e;
    border-bottom: 1px solid #312e81;
    font-size: 13px;
    color: #c4b5fd;
    flex-shrink: 0;
  }
  .update-banner button {
    padding: 4px 10px;
    border: 1px solid #312e81;
    border-radius: 4px;
    background: #1f1f2e;
    color: #a78bfa;
    cursor: pointer;
    font-size: 12px;
  }
  .update-banner button:hover { background: #2a2a3e; }
  .update-banner button.dismiss { border-color: #2a2a38; color: #6b7280; margin-left: auto; }
</style>

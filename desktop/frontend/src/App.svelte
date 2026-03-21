<!-- desktop/frontend/src/App.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import { getConnectionStatus, isDeviceOwnerInstalled, onConnectionChange, onAdminVersionMismatch, updateAdmin, getKnownStores, hideApp } from './lib/wails'
  import type { ConnectionStatus } from './lib/wails'
  import Sidebar from './components/Sidebar.svelte'
  import StatusBar from './components/StatusBar.svelte'
  import SetupTab from './components/SetupTab.svelte'
  import AppsTab from './components/AppsTab.svelte'
  import InstallTab from './components/InstallTab.svelte'
  import DocsTab from './components/DocsTab.svelte'

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
    try {
      const stores = await getKnownStores()
      await Promise.allSettled(stores.map(pkg => hideApp(pkg)))
    } catch {}
  }

  function handleResetComplete() {
    deviceOwnerInstalled = false
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
  <Sidebar {activeTab} on:navigate={(e) => activeTab = e.detail} />

  <div class="main">
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

    <div class="content">
      {#if activeTab === 'setup'}
        <SetupTab {connected} {deviceOwnerInstalled} on:setupcomplete={handleSetupComplete} on:resetcomplete={handleResetComplete} />
      {:else if activeTab === 'apps'}
        <AppsTab {connected} />
      {:else if activeTab === 'install'}
        <InstallTab {connected} />
      {:else if activeTab === 'docs'}
        <DocsTab />
      {/if}
    </div>

    <StatusBar {connected} {serial} />
  </div>
</div>

<style>
  .app {
    display: flex;
    height: 100vh;
    background: var(--bg-app);
    overflow: hidden;
  }

  .main {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .content {
    flex: 1;
    overflow-y: auto;
    background: var(--bg-content);
    display: flex;
    flex-direction: column;
  }

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

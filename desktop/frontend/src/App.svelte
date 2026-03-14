<script lang="ts">
  import { onMount } from 'svelte'
  import { getConnectionStatus, isDeviceOwnerInstalled, onConnectionChange } from './lib/wails'
  import type { ConnectionStatus } from './lib/wails'
  import StatusBar from './components/StatusBar.svelte'
  import SetupTab from './components/SetupTab.svelte'
  import AppsTab from './components/AppsTab.svelte'
  import InstallTab from './components/InstallTab.svelte'

  let activeTab = 'setup'
  let connected = false
  let serial = ''
  let deviceOwnerInstalled = false

  onMount(async () => {
    const status = await getConnectionStatus()
    connected = status.connected
    serial = status.serial

    if (connected) {
      deviceOwnerInstalled = await isDeviceOwnerInstalled()
      activeTab = deviceOwnerInstalled ? 'apps' : 'setup'
    }

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

  <main class="content">
    {#if activeTab === 'setup'}
      <SetupTab {connected} {deviceOwnerInstalled} />
    {:else if activeTab === 'apps'}
      <AppsTab {connected} />
    {:else if activeTab === 'install'}
      <InstallTab {connected} />
    {/if}
  </main>

  <StatusBar {connected} {serial} />
</div>

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) { font-family: system-ui, -apple-system, sans-serif; }
  .app { display: flex; flex-direction: column; height: 100vh; }
  .tabs { display: flex; border-bottom: 1px solid #e0e0e0; padding: 0 16px; background: #fafafa; }
  .tabs button {
    padding: 12px 20px;
    border: none;
    background: none;
    cursor: pointer;
    font-size: 14px;
    color: #555;
    border-bottom: 2px solid transparent;
    transition: color 0.15s, border-color 0.15s;
  }
  .tabs button.active { color: #1976d2; border-bottom-color: #1976d2; }
  .tabs button:disabled { opacity: 0.4; cursor: default; }
  .content { flex: 1; overflow-y: auto; padding: 24px; }
</style>

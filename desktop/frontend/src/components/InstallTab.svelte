<script lang="ts">
  import { installAPK, openFileDialog } from '../lib/wails'

  export let connected: boolean

  let selectedPath = ''
  let status: 'idle' | 'installing' | 'success' | 'error' = 'idle'
  let errorMessage = ''

  async function pickFile() {
    try {
      const path = await openFileDialog()
      if (path) selectedPath = path
    } catch (e) {
      // User cancelled dialog — ignore
    }
  }

  async function install() {
    if (!selectedPath) return
    status = 'installing'
    errorMessage = ''
    try {
      await installAPK(selectedPath)
      status = 'success'
    } catch (e: any) {
      errorMessage = e?.message ?? String(e)
      status = 'error'
    }
  }

  function reset() {
    status = 'idle'
    selectedPath = ''
    errorMessage = ''
  }
</script>

<div class="install-tab">
  <h2>Install APK</h2>
  <p class="description">
    Since the Play Store is hidden, this is the only way to install new apps.
    Select an APK file from your computer to install it on your phone.
  </p>

  <div class="file-picker">
    <input type="text" readonly value={selectedPath} placeholder="No file selected" />
    <button on:click={pickFile} disabled={status === 'installing'}>Browse…</button>
  </div>

  <button
    class="install-btn"
    disabled={!selectedPath || !connected || status === 'installing'}
    on:click={install}
  >
    {#if status === 'installing'}
      Installing…
    {:else if !connected}
      No phone connected
    {:else}
      Install to Phone
    {/if}
  </button>

  {#if status === 'success'}
    <div class="banner success">
      Installed successfully! The app should now appear on your phone.
      <button on:click={reset}>Install Another</button>
    </div>
  {:else if status === 'error'}
    <div class="banner error">
      <strong>Install failed:</strong> {errorMessage}
      <button on:click={() => status = 'idle'}>Try Again</button>
    </div>
  {/if}
</div>

<style>
  .install-tab { max-width: 600px; }
  h2 { margin-bottom: 12px; font-size: 20px; }
  .description { color: #555; margin-bottom: 20px; line-height: 1.6; }
  .file-picker { display: flex; gap: 8px; margin-bottom: 12px; }
  .file-picker input {
    flex: 1; padding: 8px 12px;
    border: 1px solid #ddd; border-radius: 4px;
    font-size: 14px; background: #f9f9f9; color: #333;
  }
  .file-picker button {
    padding: 8px 16px; cursor: pointer;
    border: 1px solid #ddd; border-radius: 4px; background: #f5f5f5;
  }
  .file-picker button:disabled { opacity: 0.5; cursor: default; }
  .install-btn {
    padding: 12px 32px; background: #1976d2; color: white;
    border: none; border-radius: 4px; font-size: 15px; cursor: pointer;
    transition: background 0.15s;
  }
  .install-btn:hover:not(:disabled) { background: #1565c0; }
  .install-btn:disabled { background: #bdbdbd; cursor: default; }
  .banner {
    margin-top: 16px; padding: 16px; border-radius: 4px;
    display: flex; align-items: center; gap: 12px; flex-wrap: wrap; line-height: 1.5;
  }
  .banner.success { background: #e8f5e9; border-left: 4px solid #4caf50; color: #2e7d32; }
  .banner.error { background: #ffebee; border-left: 4px solid #f44336; color: #c62828; }
  .banner button { padding: 4px 12px; cursor: pointer; white-space: nowrap; }
</style>

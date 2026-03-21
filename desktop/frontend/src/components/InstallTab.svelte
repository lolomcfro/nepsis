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
  <div class="tab-hero">
    <div class="hero-title">Install APK</div>
    <div class="hero-subtitle">Select an APK or bundle file to sideload</div>
  </div>
  <p class="description">
    Since the Play Store is hidden, this is the only way to install new apps.
    Select an app package from your computer to install it on your phone.
    Supports plain APKs (<code>.apk</code>) and split bundles (<code>.apkm</code>, <code>.xapk</code>, <code>.apks</code>).
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
  .tab-hero {
    background: linear-gradient(135deg, var(--bg-hero-start), var(--bg-hero-end));
    padding: 18px 22px 16px;
    border-bottom: 1px solid var(--border-hero);
    flex-shrink: 0;
  }
  .hero-title {
    font-size: 15px;
    font-weight: 800;
    letter-spacing: -0.3px;
    color: var(--text-primary);
    margin-bottom: 4px;
  }
  .hero-subtitle {
    font-size: 11px;
    font-weight: 500;
    color: var(--accent);
  }
  .install-tab { max-width: 600px; }
  .description { color: var(--text-secondary); margin-bottom: 20px; line-height: 1.6; }
  .file-picker { display: flex; gap: 8px; margin-bottom: 12px; }
  .file-picker input {
    flex: 1; padding: 8px 12px;
    border: 1px solid var(--border-input); border-radius: 4px;
    font-size: 14px; background: var(--bg-input); color: var(--text-input);
  }
  .file-picker button {
    padding: 8px 16px; cursor: pointer;
    border: 1px solid var(--border-btn-secondary); border-radius: 4px;
    background: var(--bg-btn-secondary); color: var(--text-btn-secondary);
  }
  .file-picker button:disabled { opacity: 0.5; cursor: default; }
  .install-btn {
    padding: 12px 32px;
    background: linear-gradient(135deg, var(--accent), var(--accent-subtle));
    color: white;
    border: none; border-radius: 6px; font-size: 15px; cursor: pointer;
    transition: opacity 0.15s;
  }
  .install-btn:hover:not(:disabled) { opacity: 0.88; }
  .install-btn:disabled { opacity: 0.45; cursor: default; }
  .banner {
    margin-top: 16px; padding: 16px; border-radius: 6px;
    display: flex; align-items: center; gap: 12px; flex-wrap: wrap; line-height: 1.5;
  }
  .banner.success {
    background: var(--color-success-bg);
    border-left: 4px solid var(--color-success-border);
    color: var(--color-success-text);
  }
  .banner.error {
    background: var(--color-error-bg);
    border-left: 4px solid var(--color-error-border);
    color: var(--color-error-text);
  }
  .banner button { padding: 4px 12px; cursor: pointer; white-space: nowrap; }
</style>

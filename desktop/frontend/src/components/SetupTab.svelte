<script lang="ts">
  import { runSetup } from '../lib/wails'

  export let connected: boolean
  export let deviceOwnerInstalled: boolean

  let step: 'instructions' | 'running' | 'done' | 'error' = 'instructions'
  let errorMessage = ''

  async function startSetup() {
    step = 'running'
    try {
      await runSetup()
      step = 'done'
      deviceOwnerInstalled = true
    } catch (e: any) {
      errorMessage = e?.message ?? String(e)
      step = 'error'
    }
  }
</script>

<div class="setup">
  <h2>Setup</h2>

  {#if deviceOwnerInstalled}
    <div class="banner success">
      SoberAdmin is installed and active as Device Owner.
      Your phone is locked down.
    </div>
  {:else if step === 'instructions'}
    <p>Before continuing, complete these steps on your phone:</p>
    <ol>
      <li>
        <strong>Remove all Google accounts</strong><br>
        Settings → Accounts → Google → Remove account<br>
        <em>Required for Device Owner — you can re-add them after setup if needed</em>
      </li>
      <li>
        <strong>Enable Developer Mode</strong><br>
        Settings → About Phone → tap <em>Build Number</em> 7 times
      </li>
      <li>
        <strong>Enable USB Debugging</strong><br>
        Settings → Developer Options → USB Debugging → On
      </li>
      <li>
        <strong>Plug your phone into this computer via USB</strong><br>
        Tap <em>Allow</em> on the USB Debugging prompt on your phone
      </li>
    </ol>

    <button class="primary" disabled={!connected} on:click={startSetup}>
      {connected ? 'Begin Setup' : 'Waiting for phone…'}
    </button>

  {:else if step === 'running'}
    <div class="progress">
      <div class="spinner"></div>
      <p>Setting up SoberAdmin — do not unplug your phone…</p>
    </div>

  {:else if step === 'done'}
    <div class="banner success">
      Setup complete! Your phone is now locked down.<br>
      Switch to the <strong>Apps</strong> tab to manage app visibility.
    </div>

  {:else if step === 'error'}
    <div class="banner error">
      <strong>Setup failed:</strong> {errorMessage}
      <button on:click={() => step = 'instructions'}>Try Again</button>
    </div>
  {/if}
</div>

<style>
  .setup { max-width: 600px; }
  h2 { margin-bottom: 16px; font-size: 20px; }
  ol { margin: 16px 0 24px 20px; line-height: 1.8; }
  ol li { margin-bottom: 12px; }
  em { color: #888; font-size: 13px; }
  .primary {
    padding: 12px 32px;
    background: #1976d2;
    color: white;
    border: none;
    border-radius: 4px;
    font-size: 15px;
    cursor: pointer;
    transition: background 0.15s;
  }
  .primary:hover:not(:disabled) { background: #1565c0; }
  .primary:disabled { background: #bdbdbd; cursor: default; }
  .progress { display: flex; align-items: center; gap: 16px; margin-top: 24px; }
  .spinner {
    width: 24px; height: 24px; flex-shrink: 0;
    border: 3px solid #e0e0e0; border-top-color: #1976d2;
    border-radius: 50%; animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  .banner {
    padding: 16px; border-radius: 4px; margin-top: 16px;
    line-height: 1.6;
  }
  .banner.success { background: #e8f5e9; border-left: 4px solid #4caf50; color: #2e7d32; }
  .banner.error {
    background: #ffebee; border-left: 4px solid #f44336; color: #c62828;
    display: flex; align-items: center; gap: 16px; flex-wrap: wrap;
  }
  .banner.error button { padding: 4px 12px; cursor: pointer; white-space: nowrap; }
</style>

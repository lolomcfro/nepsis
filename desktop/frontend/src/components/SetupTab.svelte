<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { runSetup } from '../lib/wails'

  export let connected: boolean
  export let deviceOwnerInstalled: boolean

  const dispatch = createEventDispatcher()

  let step: 'instructions' | 'running' | 'done' | 'error' = 'instructions'
  let errorMessage = ''

  async function startSetup() {
    step = 'running'
    try {
      await runSetup()
      step = 'done'
      deviceOwnerInstalled = true
      dispatch('setupcomplete')
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
  .setup { max-width: 600px; color: #e2e2e8; }
  h2 { margin-bottom: 16px; font-size: 20px; color: #e2e2e8; }
  ol { margin: 16px 0 24px 20px; line-height: 1.8; color: #e2e2e8; }
  ol li { margin-bottom: 12px; }
  ol li::marker { color: #a78bfa; font-weight: 600; }
  em { color: #9ca3af; font-size: 13px; }
  p { color: #9ca3af; }
  .primary {
    padding: 12px 32px;
    background: linear-gradient(135deg, #7c6af7, #a78bfa);
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 15px;
    cursor: pointer;
    transition: opacity 0.15s;
  }
  .primary:hover:not(:disabled) { opacity: 0.88; }
  .primary:disabled {
    background: #1f1f2e;
    color: #4b5563;
    border: 1px solid #2a2a38;
    cursor: default;
  }
  .progress { display: flex; align-items: center; gap: 16px; margin-top: 24px; color: #9ca3af; }
  .spinner {
    width: 24px; height: 24px; flex-shrink: 0;
    border: 3px solid #2a2a38; border-top-color: #a78bfa;
    border-radius: 50%; animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  .banner {
    padding: 16px; border-radius: 6px; margin-top: 16px;
    line-height: 1.6;
  }
  .banner.success { background: #1a2a1f; border: 1px solid #166534; border-left: 4px solid #166534; color: #4ade80; }
  .banner.error {
    background: #1f1515; border: 1px solid #7f1d1d; border-left: 4px solid #7f1d1d; color: #f87171;
    display: flex; align-items: center; gap: 16px; flex-wrap: wrap;
  }
  .banner.error button {
    padding: 4px 12px; cursor: pointer; white-space: nowrap;
    background: #1f1f2e; color: #f87171; border: 1px solid #7f1d1d; border-radius: 4px;
  }
  .banner.error button:hover { background: #2a2020; }
</style>

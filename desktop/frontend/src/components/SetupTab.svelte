<script lang="ts">
  import { onDestroy, createEventDispatcher } from 'svelte'
  import {
    getGoogleAccountCount, openAccountSettings, exportContactsToDesktop,
    getContactsBackupInfo, runInstall, runReset, importContactsFromBackup
  } from '../lib/wails'
  import type { ContactsBackupInfo } from '../lib/wails'

  export let connected: boolean
  export let deviceOwnerInstalled: boolean

  const dispatch = createEventDispatcher()

  // ── Wizard state (setup mode) ───────────────────────────────────────────
  type WizardStep =
    | 'detect'
    | 'backup-consent'
    | 'backing-up'
    | 'guide-removal'
    | 'installing'
    | 'success'
    | 'error'

  let wizardStep: WizardStep = 'detect'
  let accountCount = 0
  let backupPath = ''
  let errorMessage = ''
  let pollInterval: ReturnType<typeof setInterval> | null = null
  let disconnectedDuringPoll = false
  let isDetecting = false  // guard against re-entrant detectAccounts() calls

  // ── Reset state (post-setup mode) ───────────────────────────────────────
  type ResetState = 'idle' | 'confirm' | 'resetting' | 'restore-prompt' | 'restoring' | 'reset-done'
  let resetState: ResetState = 'idle'
  let resetError = ''
  let backupInfo: ContactsBackupInfo | null = null

  // ── Auto-detect on connect ───────────────────────────────────────────────
  // Guard against re-entrancy: if connected toggles rapidly while detectAccounts()
  // is awaiting, only one invocation should run at a time.
  $: if (connected && !deviceOwnerInstalled && wizardStep === 'detect' && !isDetecting) {
    detectAccounts()
  }

  // Only track disconnect when it can meaningfully pause the wizard.
  $: if (!connected && (wizardStep === 'guide-removal' || wizardStep === 'detect')) {
    disconnectedDuringPoll = true
    stopPoll()
  }

  $: if (connected && disconnectedDuringPoll) {
    disconnectedDuringPoll = false
    if (wizardStep === 'guide-removal') startPoll()
    if (wizardStep === 'detect') detectAccounts()
  }

  async function detectAccounts() {
    isDetecting = true
    try {
      accountCount = await getGoogleAccountCount()
      if (accountCount === 0) {
        // No accounts — skip backup and account-removal steps, go straight to install.
        await doInstall()
      } else {
        wizardStep = 'backup-consent'
      }
    } catch (e: any) {
      // Ignore — phone may have disconnected; reactive block will handle reconnect
    } finally {
      isDetecting = false
    }
  }

  async function skipBackupAndProceed() {
    wizardStep = 'guide-removal'
    startPoll()
    openAccountSettings().catch(() => {})
  }

  async function doBackup() {
    wizardStep = 'backing-up'
    try {
      backupPath = await exportContactsToDesktop()
      wizardStep = 'guide-removal'
      startPoll()
      openAccountSettings().catch(() => {})
    } catch (e: any) {
      errorMessage = e?.message ?? String(e)
      wizardStep = 'error'
    }
  }

  function startPoll() {
    stopPoll()
    pollInterval = setInterval(async () => {
      if (!connected) return
      try {
        accountCount = await getGoogleAccountCount()
        if (accountCount === 0) {
          stopPoll()
          await doInstall()
        }
      } catch {
        // Phone disconnected — reactive block handles resume
      }
    }, 2000)
  }

  function stopPoll() {
    if (pollInterval !== null) {
      clearInterval(pollInterval)
      pollInterval = null
    }
  }

  async function doInstall() {
    wizardStep = 'installing'
    try {
      await runInstall()
      wizardStep = 'success'
      deviceOwnerInstalled = true
      dispatch('setupcomplete')
    } catch (e: any) {
      errorMessage = e?.message ?? String(e)
      wizardStep = 'error'
    }
  }

  function retryFromStart() {
    stopPoll()
    errorMessage = ''
    wizardStep = 'detect'
  }

  // ── Reset flow ────────────────────────────────────────────────────────────
  async function startReset() {
    resetState = 'resetting'
    resetError = ''
    try {
      await runReset()
      backupInfo = await getContactsBackupInfo()
      if (backupInfo) {
        resetState = 'restore-prompt'
      } else {
        resetState = 'reset-done'
        deviceOwnerInstalled = false
        dispatch('resetcomplete')
      }
    } catch (e: any) {
      resetError = e?.message ?? String(e)
      resetState = 'idle'
    }
  }

  async function doRestore() {
    resetState = 'restoring'
    try {
      await importContactsFromBackup()
    } catch {
      // Non-fatal — contacts restore is best-effort
    }
    resetState = 'reset-done'
    deviceOwnerInstalled = false
    dispatch('resetcomplete')
  }

  function skipRestore() {
    resetState = 'reset-done'
    deviceOwnerInstalled = false
    dispatch('resetcomplete')
  }

  onDestroy(stopPoll)
</script>

<div class="setup">
  <h2>Setup</h2>

  {#if deviceOwnerInstalled}
    <!-- ── Post-setup state ── -->
    <div class="banner success">
      SoberAdmin is installed and active as Device Owner. Your phone is locked down.
    </div>

    {#if resetState === 'idle'}
      <div class="reset-section">
        {#if resetError}
          <div class="banner error" style="margin-bottom: 12px">{resetError}</div>
        {/if}
        <button class="danger" on:click={() => resetState = 'confirm'}>
          Reset Phone
        </button>
      </div>

    {:else if resetState === 'confirm'}
      <div class="banner warning">
        <p>This will remove all restrictions, show all hidden apps, and remove SoberAdmin as Device Owner.</p>
        <div class="button-row">
          <button class="danger" on:click={startReset}>Reset Everything</button>
          <button class="secondary" on:click={() => resetState = 'idle'}>Cancel</button>
        </div>
      </div>

    {:else if resetState === 'resetting'}
      <div class="progress">
        <div class="spinner"></div>
        <p>Resetting phone — do not unplug…</p>
      </div>

    {:else if resetState === 'restore-prompt'}
      <div class="banner info">
        <p>Device Owner removed. We have a contacts backup from <strong>{backupInfo?.date}</strong>.</p>
        <p class="hint">Restore it to your phone?</p>
        <div class="button-row">
          <button class="primary" on:click={doRestore}>Restore contacts</button>
          <button class="secondary" on:click={skipRestore}>Skip</button>
        </div>
      </div>

    {:else if resetState === 'restoring'}
      <div class="progress">
        <div class="spinner"></div>
        <p>Restoring contacts…</p>
      </div>

    {:else if resetState === 'reset-done'}
      <div class="banner success">
        Your phone has been fully restored. SoberAdmin is no longer active.
      </div>
    {/if}

  {:else}
    <!-- ── Setup wizard ── -->

    {#if wizardStep === 'detect'}
      <div class="progress">
        <div class="spinner"></div>
        <p>{connected ? 'Checking phone…' : 'Waiting for phone connection…'}</p>
      </div>

    {:else if wizardStep === 'backup-consent'}
      <div class="wizard-step">
        <p class="step-lead">Before you remove your Google accounts, we can save a backup of your contacts to this computer.</p>
        <div class="info-box">
          This file stays on your computer only and is never uploaded anywhere.
        </div>
        <div class="button-col">
          <button class="primary" on:click={doBackup}>Save backup and continue</button>
          <button class="secondary" on:click={skipBackupAndProceed}>Skip — I'll take my chances</button>
        </div>
        <p class="hint">Skipping means if you accidentally tap "Remove data" during account removal, your local contacts won't be recoverable.</p>
      </div>

    {:else if wizardStep === 'backing-up'}
      <div class="progress">
        <div class="spinner"></div>
        <p>Saving contacts backup…</p>
      </div>

    {:else if wizardStep === 'guide-removal'}
      <div class="wizard-step">
        {#if disconnectedDuringPoll}
          <div class="banner warning">Phone disconnected — plug it back in to continue.</div>
        {:else}
          <p class="step-lead">
            {accountCount === 1 ? '1 Google account' : `${accountCount} Google accounts`}
            {accountCount === 1 ? 'needs' : 'need'} to be removed before setup can continue.
          </p>

          <div class="warn-box">
            <strong>Important:</strong> When Android asks what to do with your data — choose <strong>Keep</strong>.
            {backupPath ? ' Your contacts are also backed up to this computer, so they\'re safe either way.' : ''}
          </div>

          <button class="primary" on:click={() => openAccountSettings().catch(() => {})}>
            Open Account Settings on my phone
          </button>

          <div class="poll-status">
            <div class="spinner small"></div>
            Waiting… {accountCount} {accountCount === 1 ? 'account' : 'accounts'} remaining
          </div>

          {#if backupPath}
            <p class="hint">Contacts backup saved to: {backupPath}</p>
          {/if}
        {/if}
      </div>

    {:else if wizardStep === 'installing'}
      <div class="progress">
        <div class="spinner"></div>
        <p>Setting up SoberAdmin — do not unplug your phone…</p>
      </div>

    {:else if wizardStep === 'success'}
      <div class="banner success">
        Setup complete! Your phone is now locked down.<br>
        Switch to the <strong>Apps</strong> tab to manage app visibility.
      </div>
      <div class="readd-section">
        <p>You can now re-add your Google account. Your synced contacts will return automatically.</p>
        <button class="secondary" on:click={() => openAccountSettings().catch(() => {})}>
          Open Account Settings
        </button>
      </div>

    {:else if wizardStep === 'error'}
      <div class="banner error">
        <strong>Setup failed:</strong> {errorMessage}
        <button on:click={retryFromStart}>Try Again</button>
      </div>
    {/if}
  {/if}
</div>

<style>
  .setup { max-width: 600px; color: #e2e2e8; }
  h2 { margin-bottom: 16px; font-size: 20px; color: #e2e2e8; }

  .wizard-step { display: flex; flex-direction: column; gap: 16px; }
  .step-lead { color: #e2e2e8; line-height: 1.6; }
  .hint { color: #6b7280; font-size: 13px; line-height: 1.5; }

  .info-box {
    padding: 12px 16px; background: #1a1a2e; border: 1px solid #312e81;
    border-left: 4px solid #7c6af7; border-radius: 6px; color: #c4b5fd; font-size: 14px;
  }
  .warn-box {
    padding: 12px 16px; background: #1f1a0f; border: 1px solid #92400e;
    border-left: 4px solid #d97706; border-radius: 6px; color: #fcd34d; font-size: 14px; line-height: 1.5;
  }

  .button-col { display: flex; flex-direction: column; gap: 10px; }
  .button-row { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 12px; }

  .poll-status {
    display: flex; align-items: center; gap: 10px;
    color: #9ca3af; font-size: 14px; margin-top: 8px;
  }

  .reset-section { margin-top: 20px; }
  .readd-section { margin-top: 20px; display: flex; flex-direction: column; gap: 12px; color: #9ca3af; font-size: 14px; }

  .primary {
    padding: 11px 28px; background: linear-gradient(135deg, #7c6af7, #a78bfa);
    color: white; border: none; border-radius: 6px; font-size: 15px; cursor: pointer;
    transition: opacity 0.15s; align-self: flex-start;
  }
  .primary:hover { opacity: 0.88; }

  .secondary {
    padding: 10px 20px; background: #1f1f2e; color: #9ca3af;
    border: 1px solid #2a2a38; border-radius: 6px; font-size: 14px; cursor: pointer;
    transition: background 0.15s; align-self: flex-start;
  }
  .secondary:hover { background: #2a2a3a; }

  .danger {
    padding: 10px 20px; background: #1f1515; color: #f87171;
    border: 1px solid #7f1d1d; border-radius: 6px; font-size: 14px; cursor: pointer;
    transition: background 0.15s;
  }
  .danger:hover { background: #2a1515; }

  .progress { display: flex; align-items: center; gap: 16px; margin-top: 8px; color: #9ca3af; }

  .spinner {
    width: 24px; height: 24px; flex-shrink: 0;
    border: 3px solid #2a2a38; border-top-color: #a78bfa;
    border-radius: 50%; animation: spin 0.8s linear infinite;
  }
  .spinner.small { width: 16px; height: 16px; border-width: 2px; }
  @keyframes spin { to { transform: rotate(360deg); } }

  .banner {
    padding: 16px; border-radius: 6px; margin-top: 8px; line-height: 1.6;
  }
  .banner.success { background: #1a2a1f; border: 1px solid #166534; border-left: 4px solid #166534; color: #4ade80; }
  .banner.error {
    background: #1f1515; border: 1px solid #7f1d1d; border-left: 4px solid #7f1d1d; color: #f87171;
    display: flex; align-items: center; gap: 16px; flex-wrap: wrap;
  }
  .banner.error button {
    padding: 4px 12px; cursor: pointer; background: #1f1f2e; color: #f87171;
    border: 1px solid #7f1d1d; border-radius: 4px;
  }
  .banner.warning { background: #1f1a0f; border: 1px solid #92400e; border-left: 4px solid #d97706; color: #fcd34d; }
  .banner.info { background: #1a1a2e; border: 1px solid #312e81; border-left: 4px solid #7c6af7; color: #c4b5fd; }
  .banner p { margin-bottom: 6px; }
  .banner .hint { color: #9ca3af; font-size: 13px; }
</style>

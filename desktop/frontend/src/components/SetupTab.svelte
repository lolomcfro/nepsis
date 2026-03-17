<script lang="ts">
  import { onDestroy, createEventDispatcher } from 'svelte'
  import {
    getGoogleAccountCount, openAccountSettings, exportContactsToDesktop,
    getContactsBackupInfo, runInstall, runReset, importContactsFromBackup, onResetStep
  } from '../lib/wails'
  import type { ContactsBackupInfo } from '../lib/wails'
  // @ts-ignore
  import { Quit } from '../../wailsjs/runtime/runtime'

  export let connected: boolean
  export let deviceOwnerInstalled: boolean

  const dispatch = createEventDispatcher()

  // ── Wizard state (setup mode) ───────────────────────────────────────────
  type WizardStep =
    | 'detect'
    | 'backup-consent'
    | 'backing-up'
    | 'guide-removal'
    | 'ready-to-install'
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
  type ResetState = 'idle' | 'confirm' | 'progress' | 'restore-prompt' | 'restoring' | 'choose-next'
  let resetState: ResetState = 'idle'
  let resetError = ''
  let backupInfo: ContactsBackupInfo | null = null
  let confirmInput = ''
  let resetSteps: { label: string; status: 'pending' | 'running' | 'done' | 'error' }[] = [
    { label: 'Unhiding all apps', status: 'pending' },
    { label: 'Deactivating Accountability Mode', status: 'pending' },
  ]

  // ── Auto-detect on connect ───────────────────────────────────────────────
  // Guard against re-entrancy: if connected toggles rapidly while detectAccounts()
  // is awaiting, only one invocation should run at a time.
  $: if (connected && !deviceOwnerInstalled && wizardStep === 'detect' && !isDetecting) {
    detectAccounts()
  }

  // Only track disconnect when it can meaningfully pause the wizard.
  $: if (!connected && (wizardStep === 'guide-removal' || wizardStep === 'detect' || wizardStep === 'installing' || wizardStep === 'ready-to-install')) {
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
        // No accounts — skip backup and account-removal steps, go to install prompt.
        wizardStep = 'ready-to-install'
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
          wizardStep = 'ready-to-install'
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
    resetState = 'progress'
    resetError = ''
    confirmInput = ''
    resetSteps = [
      { label: 'Unhiding all apps', status: 'pending' },
      { label: 'Deactivating Accountability Mode', status: 'pending' },
    ]
    const off = onResetStep((e) => {
      const map: Record<string, number> = { 'unhide': 0, 'device-owner': 1 }
      const i = map[e.step]
      if (i !== undefined) {
        resetSteps[i] = { ...resetSteps[i], status: e.status }
        resetSteps = [...resetSteps]
      }
    })
    try {
      await runReset()
      off()
      backupInfo = await getContactsBackupInfo()
      if (backupInfo) {
        resetState = 'restore-prompt'
      } else {
        resetState = 'choose-next'
      }
    } catch (e: any) {
      off()
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
    resetState = 'choose-next'
  }

  function skipRestore() {
    resetState = 'choose-next'
  }

  function restartWizard() {
    resetState = 'idle'
    deviceOwnerInstalled = false
    dispatch('resetcomplete')
    wizardStep = 'detect'
    // reactive statement fires detectAccounts()
  }

  function leaveUnrestricted() {
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
      Accountability Manager is active. Your phone is in Accountability Mode.
    </div>

    {#if resetState === 'idle'}
      <div class="reset-section">
        {#if resetError}
          <div class="banner error" style="margin-bottom: 12px">{resetError}</div>
        {/if}
        <button class="danger" on:click={() => resetState = 'confirm'}>
          Remove Accountability Mode
        </button>
      </div>

    {:else if resetState === 'confirm'}
      <div class="banner warning">
        <p>This will show all hidden apps and remove Accountability Mode from your phone.</p>
        <p>Type <strong>RESET</strong> to confirm:</p>
        <input
          class="confirm-input"
          type="text"
          bind:value={confirmInput}
          placeholder="RESET"
          autocomplete="off"
        />
        <div class="button-row">
          <button class="danger" on:click={startReset} disabled={confirmInput !== 'RESET'}>
            Reset Everything
          </button>
          <button class="secondary" on:click={() => { resetState = 'idle'; confirmInput = '' }}>
            Cancel
          </button>
        </div>
      </div>

    {:else if resetState === 'progress'}
      <div class="progress">
        <p>Removing Accountability Mode — do not unplug…</p>
        <ul class="reset-steps">
          {#each resetSteps as step}
            <li class="reset-step {step.status}">
              {#if step.status === 'running'}
                <span class="spinner-sm"></span>
              {:else if step.status === 'done'}
                ✓
              {:else if step.status === 'error'}
                ✗
              {:else}
                ·
              {/if}
              {step.label}
            </li>
          {/each}
        </ul>
      </div>

    {:else if resetState === 'restore-prompt'}
      <div class="banner info">
        <p>Accountability Mode removed. We have a contacts backup from <strong>{backupInfo?.date}</strong>.</p>
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

    {:else if resetState === 'choose-next'}
      <div class="wizard-step">
        <p class="step-lead">Done. Accountability Mode has been removed from your phone.</p>
        <div class="button-col">
          <button class="primary" on:click={restartWizard}>Set up again</button>
          <button class="secondary" on:click={leaveUnrestricted}>Leave phone unrestricted</button>
          <button class="secondary" on:click={() => Quit()}>Quit</button>
        </div>
      </div>
    {/if}

  {:else}
    <!-- ── Setup wizard ── -->

    {#if wizardStep === 'detect'}
      {#if connected}
        <div class="progress">
          <div class="spinner"></div>
          <p>Checking phone…</p>
        </div>
      {:else}
        <div class="wizard-step">
          <p class="step-lead">Plug your phone into this computer with a USB cable to get started.</p>
          <details class="trust-section">
            <summary>How does this work?</summary>
            <div class="trust-body">
              <p><strong>What Sober does:</strong></p>
              <ul>
                <li>Hides or locks apps you choose</li>
                <li>Prevents installing new apps</li>
                <li>Requires a USB cable and computer to make changes — adding friction between you and impulsive decisions</li>
                <li>Backs up your contacts to this computer at your request</li>
              </ul>
              <p><strong>What Sober cannot do:</strong></p>
              <ul>
                <li>Read your messages or emails</li>
                <li>Access your photos or files</li>
                <li>Track your location</li>
                <li>Access your passwords or accounts</li>
                <li>Send any data to a server — it works entirely offline, no account required</li>
              </ul>
            </div>
          </details>
        </div>
      {/if}

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
            You have {accountCount === 1 ? '1 Google account' : `${accountCount} Google accounts`} on this phone.
            Android requires these to be removed before Accountability Mode can be activated — your Google data stays safe and accessible from any browser.
          </p>

          <div class="warn-box">
            <strong>When prompted on your phone:</strong> tap <strong>Remove account</strong>.
            {backupPath ? ' Your contacts are backed up to this computer.' : ' Your contacts stored on the phone will not be deleted.'}
          </div>

          <button class="primary" on:click={() => openAccountSettings().catch(() => {})}>
            Open Account Settings on Phone
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

    {:else if wizardStep === 'ready-to-install'}
      <div class="wizard-step">
        <p class="step-lead">Ready to activate Accountability Mode on your phone.</p>
        <div class="button-col">
          <button class="primary" on:click={doInstall}>Activate</button>
        </div>
      </div>

    {:else if wizardStep === 'installing'}
      {#if disconnectedDuringPoll}
        <div class="banner error">
          Phone disconnected during setup. Please retry.
          <button on:click={retryFromStart}>Try Again</button>
        </div>
      {:else}
        <div class="progress">
          <div class="spinner"></div>
          <p>Setting up Accountability Mode — do not unplug your phone…</p>
        </div>
      {/if}

    {:else if wizardStep === 'success'}
      <div class="banner success">
        Done! Accountability Mode is active.<br>
        Switch to the <strong>Apps</strong> tab to choose which apps to hide.
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

  .confirm-input {
    width: 100%;
    margin: 8px 0;
    padding: 6px 10px;
    font-size: 1rem;
    border: 1px solid #92400e;
    border-radius: 4px;
    background: #1a1207;
    color: #fcd34d;
    box-sizing: border-box;
  }

  .reset-steps {
    list-style: none;
    padding: 0;
    margin: 12px 0 0;
    text-align: left;
  }

  .reset-step {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 0;
    color: #6b7280;
  }

  .reset-step.running { color: #e2e2e8; }
  .reset-step.done    { color: #4ade80; }
  .reset-step.error   { color: #f87171; }

  .spinner-sm {
    display: inline-block;
    width: 12px;
    height: 12px;
    flex-shrink: 0;
    border: 2px solid currentColor;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  .danger:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .trust-section {
    border: 1px solid #2a2a38;
    border-radius: 6px;
    overflow: hidden;
  }
  .trust-section summary {
    padding: 10px 14px;
    cursor: pointer;
    color: #9ca3af;
    font-size: 14px;
    user-select: none;
  }
  .trust-section summary:hover { color: #c4b5fd; }
  .trust-section[open] summary { color: #c4b5fd; border-bottom: 1px solid #2a2a38; }
  .trust-section summary:focus-visible { outline: 2px solid #7c6af7; outline-offset: 2px; }
  .trust-body {
    padding: 12px 16px;
    background: #1a1a2e;
    font-size: 13px;
    color: #9ca3af;
    line-height: 1.7;
  }
  .trust-body p { margin: 8px 0 4px; color: #e2e2e8; }
  .trust-body ul { margin: 0 0 8px; padding-left: 20px; }
  .trust-body li { margin: 3px 0; }
</style>

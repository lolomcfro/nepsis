<!-- desktop/frontend/src/components/StatusBar.svelte -->
<script lang="ts">
  export let connected: boolean = false
  export let serial: string = ''
  export let deviceModel: string = ''
  export let androidVersion: string = ''

  $: displayName = deviceModel || serial
  $: osLabel = androidVersion ? `Android ${androidVersion}` : 'Android'
</script>

<div class="status-bar">
  {#if connected}
    <span class="dot connected"></span>
    <span class="device-name">{displayName}</span>
    <span class="divider"></span>
    <span class="os-info">{osLabel}</span>
    <span class="spacer"></span>
    <span class="conn-label connected">Connected</span>
  {:else}
    <span class="dot"></span>
    <span class="no-device">No device connected</span>
  {/if}
</div>

<style>
  .status-bar {
    height: 28px;
    padding: 0 14px;
    background: var(--status-bar-bg);
    border-top: 1px solid var(--status-bar-border);
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--text-muted);
    flex-shrink: 0;
  }

  .dot.connected {
    background: #22c55e;
    box-shadow: 0 0 6px rgba(34, 197, 94, 0.55);
  }

  .device-name {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .divider {
    width: 1px;
    height: 10px;
    background: var(--border);
  }

  .os-info {
    font-size: 10px;
    color: var(--text-muted);
  }

  .spacer { flex: 1; }

  .conn-label {
    font-size: 10px;
    font-weight: 600;
    color: var(--text-muted);
  }

  .conn-label.connected { color: #4ade80; }

  .no-device {
    font-size: 11px;
    color: var(--text-muted);
  }
</style>

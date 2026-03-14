<script lang="ts">
  export let connected: boolean = false
  export let serial: string = ''
</script>

<div class="status-bar">
  {#if connected}
    <span class="dot connected"></span>
    Connected: {serial}
  {:else}
    <span class="dot disconnected"></span>
    No phone connected — plug in via USB
  {/if}
</div>

<style>
  .status-bar {
    padding: 6px 16px;
    font-size: 13px;
    border-top: 1px solid #e0e0e0;
    display: flex;
    align-items: center;
    gap: 8px;
    color: #555;
  }
  .dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; flex-shrink: 0; }
  .connected { background: #4caf50; }
  .disconnected { background: #9e9e9e; }
</style>

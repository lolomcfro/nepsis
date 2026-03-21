<!-- desktop/frontend/src/components/Sidebar.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte'

  export let activeTab: string = 'setup'

  const dispatch = createEventDispatcher()

  let darkMode = document.documentElement.getAttribute('data-theme') === 'dark'

  function navigate(tab: string) {
    dispatch('navigate', tab)
  }

  function toggleTheme() {
    darkMode = !darkMode
    const theme = darkMode ? 'dark' : 'light'
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', theme)
  }

  const tabs = [
    { id: 'setup',   label: 'Setup',   icon: 'setup'   },
    { id: 'apps',    label: 'Apps',    icon: 'apps'    },
    { id: 'install', label: 'Install', icon: 'install' },
    { id: 'docs',    label: 'Docs',    icon: 'docs'    },
  ]
</script>

<aside class="sidebar">
  <div class="logo" title="nepsis">
    <span>N</span>
  </div>

  <nav class="nav">
    {#each tabs as tab}
      <button
        class="nav-btn"
        class:active={activeTab === tab.id}
        title={tab.label}
        on:click={() => navigate(tab.id)}
        aria-label={tab.label}
      >
        {#if tab.icon === 'setup'}
          <svg width="17" height="17" viewBox="0 0 16 16" fill="none">
            <rect x="1" y="1" width="6" height="6" rx="1.5" fill="currentColor"/>
            <rect x="9" y="1" width="6" height="6" rx="1.5" fill="currentColor" opacity=".4"/>
            <rect x="1" y="9" width="6" height="6" rx="1.5" fill="currentColor" opacity=".4"/>
            <rect x="9" y="9" width="6" height="6" rx="1.5" fill="currentColor" opacity=".4"/>
          </svg>
        {:else if tab.icon === 'apps'}
          <svg width="17" height="17" viewBox="0 0 16 16" fill="none">
            <rect x="1" y="1" width="6" height="6" rx="1.5" fill="currentColor"/>
            <rect x="9" y="1" width="6" height="6" rx="1.5" fill="currentColor"/>
            <rect x="1" y="9" width="6" height="6" rx="1.5" fill="currentColor"/>
            <rect x="9" y="9" width="6" height="6" rx="1.5" fill="currentColor"/>
          </svg>
        {:else if tab.icon === 'install'}
          <svg width="17" height="17" viewBox="0 0 16 16" fill="none">
            <path d="M8 2v8M5 7l3 3 3-3M2 12h12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        {:else if tab.icon === 'docs'}
          <svg width="17" height="17" viewBox="0 0 16 16" fill="none">
            <path d="M4 2h6l3 3v9H4V2z" stroke="currentColor" stroke-width="1.3"/>
            <path d="M10 2v3h3" stroke="currentColor" stroke-width="1.3"/>
            <path d="M6 7h5M6 9.5h4M6 12h3" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
          </svg>
        {/if}
      </button>
    {/each}
  </nav>

  <div class="bottom">
    <button
      class="theme-toggle"
      title={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
      on:click={toggleTheme}
      aria-label="Toggle theme"
    >
      <div class="toggle-track" class:dark={darkMode}>
        <div class="toggle-thumb"></div>
      </div>
    </button>
  </div>
</aside>

<style>
  .sidebar {
    width: 46px;
    background: var(--bg-sidebar);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 10px 0;
    flex-shrink: 0;
    box-shadow: var(--shadow-sidebar);
  }

  .logo {
    width: 26px;
    height: 26px;
    background: linear-gradient(135deg, #7c6af7, #a78bfa);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 800;
    color: white;
    margin-bottom: 14px;
    flex-shrink: 0;
  }

  .nav {
    display: flex;
    flex-direction: column;
    gap: 4px;
    width: 100%;
    align-items: center;
  }

  .nav-btn {
    width: 34px;
    height: 34px;
    border: none;
    border-radius: 9px;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.12s, color 0.12s;
  }

  .nav-btn:hover {
    background: var(--accent-bg);
    color: var(--accent);
  }

  .nav-btn.active {
    background: var(--accent-bg);
    color: var(--accent);
  }

  .bottom {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding-bottom: 4px;
  }

  .theme-toggle {
    background: none;
    border: none;
    cursor: pointer;
    padding: 2px;
  }

  .toggle-track {
    width: 28px;
    height: 16px;
    background: var(--border);
    border-radius: 8px;
    display: flex;
    align-items: center;
    padding: 0 2px;
    transition: background 0.15s;
  }

  .toggle-track.dark {
    background: var(--accent);
    justify-content: flex-end;
  }

  .toggle-thumb {
    width: 12px;
    height: 12px;
    background: white;
    border-radius: 50%;
    box-shadow: 0 1px 3px rgba(0,0,0,0.2);
  }
</style>

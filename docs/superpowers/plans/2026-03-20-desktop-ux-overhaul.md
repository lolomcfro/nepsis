# Desktop UX Overhaul Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the horizontal-tab title bar UI with a slim icon sidebar, per-section hero areas, a bottom status bar, light/dark theming, and a new offline Docs tab.

**Architecture:** `App.svelte` becomes a flex shell: 46px `Sidebar` on the left + active tab component filling the right + `StatusBar` pinned at the bottom. Each tab owns its own hero gradient block at the top of its content. Theme tokens live in CSS custom properties on `:root` / `:root[data-theme="dark"]`. The `Sidebar` manages the active tab and dark/light toggle, communicating up via Svelte events.

**Tech Stack:** Svelte 3, TypeScript, Vite 3, Wails (Go backend — untouched), `marked` (new: Markdown rendering for Docs tab)

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `desktop/frontend/src/style.css` | Modify | CSS custom property tokens for light + dark theme |
| `desktop/frontend/src/App.svelte` | Modify | App shell: sidebar + tab router + status bar |
| `desktop/frontend/src/components/Sidebar.svelte` | **Create** | Icon nav rail, dark/light toggle |
| `desktop/frontend/src/components/StatusBar.svelte` | Modify | Full-width device info strip |
| `desktop/frontend/src/components/SetupTab.svelte` | Modify | Add hero area at top |
| `desktop/frontend/src/components/AppsTab.svelte` | Modify | Move search to hero, restyle rows |
| `desktop/frontend/src/components/InstallTab.svelte` | Modify | Add hero area at top |
| `desktop/frontend/src/components/DocsTab.svelte` | **Create** | Offline Markdown viewer |
| `desktop/frontend/src/docs/index.json` | **Create** | Nav tree manifest |
| `desktop/frontend/src/docs/getting-started/introduction.md` | **Create** | |
| `desktop/frontend/src/docs/getting-started/quick-setup.md` | **Create** | |
| `desktop/frontend/src/docs/setup-guide/accounts.md` | **Create** | |
| `desktop/frontend/src/docs/setup-guide/backup.md` | **Create** | |
| `desktop/frontend/src/docs/app-management/hiding-apps.md` | **Create** | |
| `desktop/frontend/src/docs/app-management/deleting-apps.md` | **Create** | |
| `desktop/frontend/src/docs/troubleshooting/common-errors.md` | **Create** | |
| `desktop/frontend/src/docs/troubleshooting/no-device.md` | **Create** | |
| `desktop/frontend/package.json` | Modify | Add `marked` dependency |

> **Note on device name:** `ConnectionStatus` from the backend only exposes `serial` (ADB device serial string, e.g. `HT6750100234`). The status bar will display `serial` as the device identifier — no backend changes required.

> **Note on testing:** No test framework exists in this project. Verification is visual via `npm run dev` in `desktop/frontend/`. Each task includes a "Start dev server and verify" step.

---

## Task 1: CSS Theme Tokens

**Files:**
- Modify: `desktop/frontend/src/style.css`

- [ ] **Step 1: Replace `style.css` with theme token system**

```css
/* desktop/frontend/src/style.css */
:root {
  --bg-app: #eeeef4;
  --bg-sidebar: #ffffff;
  --bg-content: #ffffff;
  --bg-hero-start: #ede9fe;
  --bg-hero-end: #f5f3ff;
  --bg-row: #f9fafb;
  --border: #e5e7eb;
  --border-hero: #f0ebff;
  --text-primary: #1a1a2e;
  --text-secondary: #374151;
  --text-muted: #6b7280;
  --accent: #7c6af7;
  --accent-subtle: #a78bfa;
  --accent-bg: #ede9fe;
  --status-bar-bg: #ffffff;
  --status-bar-border: #e5e7eb;
  --shadow-sidebar: 0 2px 8px rgba(0,0,0,0.07);
  --shadow-content: 0 2px 8px rgba(0,0,0,0.06);
}

:root[data-theme="dark"] {
  --bg-app: #080810;
  --bg-sidebar: #16161e;
  --bg-content: #13131e;
  --bg-hero-start: #1e1830;
  --bg-hero-end: #1a1628;
  --bg-row: #1f1f2e;
  --border: #23233a;
  --border-hero: #2a2a3e;
  --text-primary: #e2e2e8;
  --text-secondary: #9ca3af;
  --text-muted: #6b7280;
  --accent: #a78bfa;
  --accent-subtle: #c4b5fd;
  --accent-bg: #2a2040;
  --status-bar-bg: #16161e;
  --status-bar-border: #23233a;
  --shadow-sidebar: none;
  --shadow-content: none;
}

*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

body {
  font-family: system-ui, -apple-system, sans-serif;
  background: var(--bg-app);
  color: var(--text-primary);
}

@font-face {
  font-family: "Nunito";
  font-style: normal;
  font-weight: 400;
  src: local(""), url("assets/fonts/nunito-v16-latin-regular.woff2") format("woff2");
}
```

- [ ] **Step 2: Apply default theme on page load**

In `desktop/frontend/index.html`, add a `data-theme` init script before the `</head>` so the correct theme loads before paint (avoids flash):

```html
<script>
  (function() {
    var t = localStorage.getItem('theme');
    if (t === 'dark' || (!t && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
      document.documentElement.setAttribute('data-theme', 'dark');
    }
  })();
</script>
```

- [ ] **Step 3: Commit**

```bash
git add desktop/frontend/src/style.css desktop/frontend/index.html
git commit -m "feat: add CSS theme token system with light/dark support"
```

---

## Task 2: Sidebar Component

**Files:**
- Create: `desktop/frontend/src/components/Sidebar.svelte`

- [ ] **Step 1: Create `Sidebar.svelte`**

```svelte
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
```

- [ ] **Step 2: Commit**

```bash
git add desktop/frontend/src/components/Sidebar.svelte
git commit -m "feat: add Sidebar component with icon nav and theme toggle"
```

---

## Task 3: App Shell Refactor

**Files:**
- Modify: `desktop/frontend/src/App.svelte`

Replace the titlebar + horizontal tabs layout with the sidebar + content + status bar shell. All existing business logic (connection handling, version mismatch, setup routing) is preserved exactly — only the template and shell styles change.

- [ ] **Step 1: Rewrite `App.svelte`**

```svelte
<!-- desktop/frontend/src/App.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import { getConnectionStatus, isDeviceOwnerInstalled, onConnectionChange, onAdminVersionMismatch, updateAdmin, getKnownStores, hideApp } from './lib/wails'
  import type { ConnectionStatus } from './lib/wails'
  import Sidebar from './components/Sidebar.svelte'
  import StatusBar from './components/StatusBar.svelte'
  import SetupTab from './components/SetupTab.svelte'
  import AppsTab from './components/AppsTab.svelte'
  import InstallTab from './components/InstallTab.svelte'
  import DocsTab from './components/DocsTab.svelte'

  let activeTab = 'apps'
  let connected = false
  let serial = ''
  let deviceOwnerInstalled = false

  let versionMismatch: { installedVersion: number, bundledVersion: number } | null = null
  let updateState: 'idle' | 'updating' | 'success' | 'error' = 'idle'
  let updateError = ''

  async function handleUpdate() {
    updateState = 'updating'
    try {
      await updateAdmin()
      updateState = 'success'
      setTimeout(() => { versionMismatch = null; updateState = 'idle' }, 2000)
    } catch (e: any) {
      updateError = e?.message ?? String(e)
      updateState = 'error'
    }
  }

  async function handleSetupComplete() {
    deviceOwnerInstalled = true
    activeTab = 'apps'
    try {
      const stores = await getKnownStores()
      await Promise.allSettled(stores.map(pkg => hideApp(pkg)))
    } catch {}
  }

  function handleResetComplete() {
    deviceOwnerInstalled = false
  }

  onMount(async () => {
    const status = await getConnectionStatus()
    connected = status.connected
    serial = status.serial

    if (connected) {
      deviceOwnerInstalled = await isDeviceOwnerInstalled()
      activeTab = deviceOwnerInstalled ? 'apps' : 'setup'
    }

    onAdminVersionMismatch((info) => {
      versionMismatch = info
      updateState = 'idle'
    })

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
  <Sidebar {activeTab} on:navigate={(e) => activeTab = e.detail} />

  <div class="main">
    {#if versionMismatch}
      <div class="update-banner">
        {#if updateState === 'idle'}
          SoberAdmin update available (v{versionMismatch.installedVersion} → v{versionMismatch.bundledVersion})
          <button on:click={handleUpdate}>Update</button>
          <button class="dismiss" on:click={() => versionMismatch = null}>✕</button>
        {:else if updateState === 'updating'}
          Updating SoberAdmin…
        {:else if updateState === 'success'}
          SoberAdmin updated successfully.
        {:else if updateState === 'error'}
          Update failed: {updateError}
          <button on:click={handleUpdate}>Retry</button>
          <button class="dismiss" on:click={() => versionMismatch = null}>✕</button>
        {/if}
      </div>
    {/if}

    <div class="content">
      {#if activeTab === 'setup'}
        <SetupTab {connected} {deviceOwnerInstalled} on:setupcomplete={handleSetupComplete} on:resetcomplete={handleResetComplete} />
      {:else if activeTab === 'apps'}
        <AppsTab {connected} />
      {:else if activeTab === 'install'}
        <InstallTab {connected} />
      {:else if activeTab === 'docs'}
        <DocsTab />
      {/if}
    </div>

    <StatusBar {connected} {serial} />
  </div>
</div>

<style>
  .app {
    display: flex;
    height: 100vh;
    background: var(--bg-app);
    overflow: hidden;
  }

  .main {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .content {
    flex: 1;
    overflow-y: auto;
    background: var(--bg-content);
    display: flex;
    flex-direction: column;
  }

  .update-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: #1a1a2e;
    border-bottom: 1px solid #312e81;
    font-size: 13px;
    color: #c4b5fd;
    flex-shrink: 0;
  }
  .update-banner button {
    padding: 4px 10px;
    border: 1px solid #312e81;
    border-radius: 4px;
    background: #1f1f2e;
    color: #a78bfa;
    cursor: pointer;
    font-size: 12px;
  }
  .update-banner button:hover { background: #2a2a3e; }
  .update-banner button.dismiss { border-color: #2a2a38; color: #6b7280; margin-left: auto; }
</style>
```

- [ ] **Step 2: Start dev server and verify layout**

```bash
cd desktop/frontend && npm run dev
```

Expected: App loads with sidebar on the left, content on the right, status bar at bottom. All 4 nav icons visible. Theme toggle at bottom of sidebar. Clicking icons switches tabs (Docs tab will show blank until Task 8).

- [ ] **Step 3: Commit**

```bash
git add desktop/frontend/src/App.svelte
git commit -m "feat: refactor App shell to sidebar + content + status bar layout"
```

---

## Task 4: StatusBar Upgrade

**Files:**
- Modify: `desktop/frontend/src/components/StatusBar.svelte`

- [ ] **Step 1: Rewrite `StatusBar.svelte`**

```svelte
<!-- desktop/frontend/src/components/StatusBar.svelte -->
<script lang="ts">
  export let connected: boolean = false
  export let serial: string = ''
</script>

<div class="status-bar">
  {#if connected}
    <span class="dot connected"></span>
    <span class="device-name">{serial}</span>
    <span class="divider"></span>
    <span class="os-info">Android device</span>
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
```

- [ ] **Step 2: Verify in dev server**

With no device: status bar shows gray dot + "No device connected".
With device connected: shows green glowing dot + serial + "Connected".

- [ ] **Step 3: Commit**

```bash
git add desktop/frontend/src/components/StatusBar.svelte
git commit -m "feat: upgrade StatusBar to show device serial and connection state"
```

---

## Task 5: SetupTab Hero Area

**Files:**
- Modify: `desktop/frontend/src/components/SetupTab.svelte`

The SetupTab has a complex multi-step wizard. We add a hero block at the top that updates its subtitle based on the current wizard step. All existing logic is unchanged.

- [ ] **Step 1: Read `SetupTab.svelte` and confirm variable names**

Read `desktop/frontend/src/components/SetupTab.svelte`. Confirm:
- The wizard step variable is named `wizardStep` (type `WizardStep`)
- The reset state variable is named `resetState` (type `ResetState`)
- The root template element (where to insert the hero block as first child)

If the names differ from above, adjust the reactive expression in Step 2 accordingly.

- [ ] **Step 2: Add hero markup**

At the top of the `<script>` block, add a reactive subtitle:

```typescript
// Add near the top of the existing script block, after existing let declarations:
$: heroSubtitle = (() => {
  if (!connected) return 'Connect your device via USB to begin'
  if (deviceOwnerInstalled) {
    if (resetState === 'idle') return 'Device is set up · Manage or reset below'
    if (resetState === 'confirm') return 'Confirm reset to remove accountability mode'
    if (resetState === 'progress') return 'Resetting device…'
    return 'Setup complete'
  }
  const stepLabels: Record<WizardStep, string> = {
    'detect': 'Checking device accounts…',
    'backup-consent': 'Step 1 · Back up your contacts',
    'backing-up': 'Backing up contacts…',
    'guide-removal': 'Step 2 · Remove Google accounts',
    'ready-to-install': 'Step 3 · Ready to install',
    'installing': 'Installing SoberAdmin…',
    'success': 'Setup complete',
    'error': 'Something went wrong',
  }
  return stepLabels[wizardStep] ?? ''
})()
```

Add the hero block as the first element inside the tab's root element:

```html
<div class="tab-hero">
  <div class="hero-title">Setup Your Device</div>
  <div class="hero-subtitle">{heroSubtitle}</div>
</div>
```

Add scoped styles:

```css
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
```

- [ ] **Step 3: Verify hero appears and subtitle updates with wizard steps**

Run dev server. Connect/disconnect device and advance through the wizard — subtitle should update at each step.

- [ ] **Step 4: Commit**

```bash
git add desktop/frontend/src/components/SetupTab.svelte
git commit -m "feat: add hero area to SetupTab with reactive step subtitle"
```

---

## Task 6: AppsTab Hero + List Restyle

**Files:**
- Modify: `desktop/frontend/src/components/AppsTab.svelte`

Move the search bar into the hero area. Add a visibility filter pill. Restyle app rows.

- [ ] **Step 1: Add `visibilityFilter` state and filtered reactive**

In the script block, add after existing `let` declarations:

```typescript
let visibilityFilter: 'all' | 'visible' | 'hidden' = 'all'

$: visibleFilteredRegularApps = filteredRegularApps.filter(a => {
  if (visibilityFilter === 'visible') return !a.hidden
  if (visibilityFilter === 'hidden') return a.hidden
  return true
})

$: visibleFilteredStoreApps = filteredStoreApps.filter(a => {
  if (visibilityFilter === 'visible') return !a.hidden
  if (visibilityFilter === 'hidden') return a.hidden
  return true
})

function cycleFilter() {
  if (visibilityFilter === 'all') visibilityFilter = 'visible'
  else if (visibilityFilter === 'visible') visibilityFilter = 'hidden'
  else visibilityFilter = 'all'
}
```

- [ ] **Step 2: Replace the existing search input and section headers with a hero block**

Replace whatever top-level search/header markup exists with:

```html
<div class="tab-hero">
  <div class="hero-title">Manage Apps</div>
  <div class="hero-controls">
    <div class="search-wrap">
      <svg class="search-icon" width="12" height="12" viewBox="0 0 12 12" fill="none">
        <circle cx="5" cy="5" r="3.5" stroke="currentColor" stroke-width="1.3"/>
        <path d="M8 8l2 2" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
      </svg>
      <input
        class="search-input"
        type="text"
        placeholder="Search apps…"
        bind:value={search}
      />
    </div>
    <button class="filter-pill" on:click={cycleFilter}>
      {visibilityFilter === 'all' ? 'All' : visibilityFilter === 'visible' ? 'Visible' : 'Hidden'}
    </button>
  </div>
</div>
```

- [ ] **Step 3: Update app row markup**

Replace the existing app list rows to use the new row style. Each row in the `{#each}` block should render as:

```html
<div class="app-row" class:hidden-app={app.hidden}>
  <div class="app-icon-placeholder">{app.label.charAt(0).toUpperCase()}</div>
  <span class="app-name">{app.label}</span>
  <span class="badge" class:badge-visible={!app.hidden} class:badge-hidden={app.hidden}>
    {app.hidden ? 'Hidden' : 'Visible'}
  </span>
  <!-- keep existing action buttons (hide/show/delete) here -->
</div>
```

- [ ] **Step 4: Add scoped styles**

```css
.tab-hero {
  background: linear-gradient(135deg, var(--bg-hero-start), var(--bg-hero-end));
  padding: 16px 18px 14px;
  border-bottom: 1px solid var(--border-hero);
  flex-shrink: 0;
}
.hero-title {
  font-size: 15px;
  font-weight: 800;
  letter-spacing: -0.3px;
  color: var(--text-primary);
  margin-bottom: 10px;
}
.hero-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}
.search-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(255,255,255,0.6);
  border: 1px solid var(--border-hero);
  border-radius: 7px;
  height: 28px;
  padding: 0 10px;
  color: var(--text-muted);
}
:root[data-theme="dark"] .search-wrap {
  background: rgba(255,255,255,0.05);
}
.search-icon { flex-shrink: 0; }
.search-input {
  border: none;
  background: transparent;
  font-size: 12px;
  color: var(--text-primary);
  outline: none;
  width: 100%;
}
.search-input::placeholder { color: var(--text-muted); }
.filter-pill {
  height: 28px;
  padding: 0 12px;
  border: 1px solid var(--border-hero);
  border-radius: 14px;
  background: var(--accent-bg);
  color: var(--accent);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
}
.app-row {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 36px;
  padding: 0 14px;
  border-bottom: 1px solid var(--border);
  transition: opacity 0.12s;
}
.app-row.hidden-app { opacity: 0.55; }
.app-icon-placeholder {
  width: 20px;
  height: 20px;
  border-radius: 5px;
  background: var(--accent-bg);
  color: var(--accent);
  font-size: 10px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.app-name {
  flex: 1;
  font-size: 12px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.badge {
  font-size: 10px;
  font-weight: 600;
  padding: 2px 7px;
  border-radius: 10px;
  flex-shrink: 0;
}
.badge-visible {
  background: #f0fdf4;
  color: #166534;
  border: 1px solid #bbf7d0;
}
.badge-hidden {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fde68a;
}
:root[data-theme="dark"] .badge-visible {
  background: #052e16;
  color: #4ade80;
  border-color: #166534;
}
:root[data-theme="dark"] .badge-hidden {
  background: #451a03;
  color: #fbbf24;
  border-color: #78350f;
}
```

- [ ] **Step 5: Verify**

Run dev server with device connected. Search filters list. Filter pill cycles All/Visible/Hidden. Hidden apps are dimmed.

- [ ] **Step 6: Commit**

```bash
git add desktop/frontend/src/components/AppsTab.svelte
git commit -m "feat: move search to hero area and restyle app rows in AppsTab"
```

---

## Task 7: InstallTab Hero Area

**Files:**
- Modify: `desktop/frontend/src/components/InstallTab.svelte`

- [ ] **Step 1: Read `InstallTab.svelte` to understand current structure**

- [ ] **Step 2: Add hero block as first element in template**

```html
<div class="tab-hero">
  <div class="hero-title">Install APK</div>
  <div class="hero-subtitle">Select an APK or bundle file to sideload</div>
</div>
```

Add scoped styles:

```css
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
```

- [ ] **Step 3: Verify**

Run dev server. Navigate to Install tab — hero appears above the file picker.

- [ ] **Step 4: Commit**

```bash
git add desktop/frontend/src/components/InstallTab.svelte
git commit -m "feat: add hero area to InstallTab"
```

---

## Task 8: Install marked + DocsTab

**Files:**
- Modify: `desktop/frontend/package.json`
- Create: `desktop/frontend/src/components/DocsTab.svelte`

- [ ] **Step 1: Install `marked`**

```bash
cd desktop/frontend && npm install marked
```

Verify `package.json` now lists `"marked"` in `dependencies`.

- [ ] **Step 2: Create `DocsTab.svelte`**

```svelte
<!-- desktop/frontend/src/components/DocsTab.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import { marked } from 'marked'

  interface Article {
    id: string
    title: string
    file: string
  }

  interface Category {
    id: string
    title: string
    articles: Article[]
  }

  let categories: Category[] = []
  let activeArticle: Article | null = null
  let activeCategory: Category | null = null
  let renderedHtml = ''
  let loading = false

  // Import all markdown files eagerly so Vite bundles them
  const mdModules = import.meta.glob('../docs/**/*.md', { as: 'raw', eager: true }) as Record<string, string>
  // Import index manifest
  import indexJson from '../docs/index.json'

  onMount(() => {
    categories = indexJson.categories
    if (categories.length > 0 && categories[0].articles.length > 0) {
      selectArticle(categories[0], categories[0].articles[0])
    }
  })

  async function selectArticle(category: Category, article: Article) {
    activeCategory = category
    activeArticle = article
    loading = true
    try {
      const raw = mdModules[`../docs/${article.file}`] ?? '# Not found\n\nThis article could not be loaded.'
      const dirty = await marked.parse(raw)
      // Strip scripts from rendered HTML
      renderedHtml = dirty.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
    } finally {
      loading = false
    }
  }
</script>

<div class="docs-tab">
  <div class="tab-hero">
    <div class="hero-title">{activeArticle?.title ?? 'Documentation'}</div>
    <div class="hero-subtitle">
      {#if activeCategory && activeArticle}
        {activeCategory.title} › {activeArticle.title}
      {:else}
        Offline documentation
      {/if}
    </div>
  </div>

  <div class="docs-body">
    <nav class="docs-nav">
      {#each categories as category}
        <div class="nav-category">
          <div class="category-label">{category.title}</div>
          {#each category.articles as article}
            <button
              class="article-link"
              class:active={activeArticle?.id === article.id}
              on:click={() => selectArticle(category, article)}
            >
              {article.title}
            </button>
          {/each}
        </div>
      {/each}
    </nav>

    <div class="docs-content">
      {#if loading}
        <div class="loading">Loading…</div>
      {:else}
        <!-- eslint-disable-next-line svelte/no-at-html-tags -->
        {@html renderedHtml}
      {/if}
    </div>
  </div>
</div>

<style>
  .docs-tab {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

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

  .docs-body {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  .docs-nav {
    width: 160px;
    flex-shrink: 0;
    border-right: 1px solid var(--border);
    overflow-y: auto;
    padding: 12px 8px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    background: var(--bg-content);
  }

  .nav-category { display: flex; flex-direction: column; gap: 2px; }

  .category-label {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-muted);
    padding: 0 8px;
    margin-bottom: 4px;
  }

  .article-link {
    display: block;
    width: 100%;
    text-align: left;
    padding: 6px 8px;
    border: none;
    border-radius: 6px;
    background: transparent;
    font-size: 12px;
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 0.1s, color 0.1s;
  }
  .article-link:hover { background: var(--accent-bg); color: var(--accent); }
  .article-link.active { background: var(--accent-bg); color: var(--accent); font-weight: 600; }

  .docs-content {
    flex: 1;
    overflow-y: auto;
    padding: 24px 28px;
    background: var(--bg-content);
    color: var(--text-primary);
    font-size: 13px;
    line-height: 1.7;
  }

  .loading { color: var(--text-muted); font-size: 12px; }

  /* Markdown content styles */
  .docs-content :global(h1) { font-size: 18px; font-weight: 800; margin-bottom: 16px; color: var(--text-primary); }
  .docs-content :global(h2) { font-size: 14px; font-weight: 700; margin: 20px 0 10px; color: var(--text-primary); }
  .docs-content :global(h3) { font-size: 13px; font-weight: 600; margin: 16px 0 8px; color: var(--text-primary); }
  .docs-content :global(p) { margin-bottom: 12px; }
  .docs-content :global(ul), .docs-content :global(ol) { padding-left: 20px; margin-bottom: 12px; }
  .docs-content :global(li) { margin-bottom: 4px; }
  .docs-content :global(code) { background: var(--bg-row); padding: 2px 5px; border-radius: 4px; font-size: 11px; font-family: monospace; }
  .docs-content :global(pre) { background: var(--bg-row); border: 1px solid var(--border); border-radius: 8px; padding: 12px; overflow-x: auto; margin-bottom: 12px; }
  .docs-content :global(pre code) { background: none; padding: 0; }
  .docs-content :global(hr) { border: none; border-top: 1px solid var(--border); margin: 20px 0; }
  .docs-content :global(a) { color: var(--accent); text-decoration: none; }
  .docs-content :global(a:hover) { text-decoration: underline; }
  .docs-content :global(strong) { font-weight: 700; }
</style>
```

- [ ] **Step 3: Verify Docs tab loads offline**

Run dev server. Click Docs icon in sidebar — nav tree renders, first article loads, no network requests in browser devtools Network tab.

- [ ] **Step 4: Commit**

```bash
git add desktop/frontend/package.json desktop/frontend/src/components/DocsTab.svelte
git commit -m "feat: add offline DocsTab with marked Markdown renderer"
```

---

## Task 9: Docs Content Files

**Files:**
- Create: `desktop/frontend/src/docs/index.json`
- Create: 8 Markdown files

- [ ] **Step 1: Create `index.json` nav manifest**

```json
{
  "categories": [
    {
      "id": "getting-started",
      "title": "Getting Started",
      "articles": [
        { "id": "introduction",  "title": "Introduction",  "file": "getting-started/introduction.md" },
        { "id": "quick-setup",   "title": "Quick Setup",   "file": "getting-started/quick-setup.md" }
      ]
    },
    {
      "id": "setup-guide",
      "title": "Setup Guide",
      "articles": [
        { "id": "accounts", "title": "Google Accounts", "file": "setup-guide/accounts.md" },
        { "id": "backup",   "title": "Contacts Backup", "file": "setup-guide/backup.md" }
      ]
    },
    {
      "id": "app-management",
      "title": "App Management",
      "articles": [
        { "id": "hiding-apps",   "title": "Hiding Apps",   "file": "app-management/hiding-apps.md" },
        { "id": "deleting-apps", "title": "Deleting Apps", "file": "app-management/deleting-apps.md" }
      ]
    },
    {
      "id": "troubleshooting",
      "title": "Troubleshooting",
      "articles": [
        { "id": "common-errors", "title": "Common Errors", "file": "troubleshooting/common-errors.md" },
        { "id": "no-device",     "title": "No Device Found", "file": "troubleshooting/no-device.md" }
      ]
    }
  ]
}
```

- [ ] **Step 2: Create Markdown files**

Create each file with concise, accurate content. All files go under `desktop/frontend/src/docs/`.

**`getting-started/introduction.md`**
```markdown
# Introduction

Nepsis is a desktop tool that helps you set up and manage accountability mode on an Android device.

## What it does

- Guides you through removing Google accounts and installing SoberAdmin as device owner
- Lets you hide or remove apps from the device
- Allows sideloading APK files
- Backs up and restores contacts during setup

## Requirements

- A Windows, macOS, or Linux computer
- An Android device with USB debugging enabled
- A USB cable
```

**`getting-started/quick-setup.md`**
```markdown
# Quick Setup

1. Plug the Android device in via USB
2. On the device, allow USB debugging when prompted
3. Open Nepsis — it will detect the device automatically
4. Follow the Setup wizard to remove accounts and install SoberAdmin
5. Once complete, use the Apps tab to hide or delete unwanted apps
```

**`setup-guide/accounts.md`**
```markdown
# Google Accounts

SoberAdmin must be installed as device owner. Android requires no Google accounts on the device before this is possible.

## Removing accounts

The Setup wizard opens the Android Accounts settings page automatically. Remove all Google accounts listed there before continuing.

## Why this is required

Android's device owner policy does not allow installation when accounts are present. This is an Android system restriction, not a Nepsis limitation.
```

**`setup-guide/backup.md`**
```markdown
# Contacts Backup

Nepsis can export your contacts to a VCF file on your computer before setup, and restore them afterward.

## During setup

When the wizard detects contacts on the device, it will offer to back them up before you remove accounts.

## Restoring contacts

After setup completes, the wizard will prompt you to restore contacts from the saved backup file.

## Backup location

Backups are saved to your Desktop as `contacts-backup.vcf`.
```

**`app-management/hiding-apps.md`**
```markdown
# Hiding Apps

Hidden apps are removed from the device's app drawer and cannot be launched by the user, but their data is preserved.

## How to hide an app

1. Go to the **Apps** tab
2. Find the app using search or scroll
3. Click the action menu and select **Hide**

## Unhiding an app

Use the **Filter** pill to show Hidden apps, then select **Show** from the action menu.

## App stores

Known app stores (Google Play, Galaxy Store, etc.) are hidden automatically during setup.
```

**`app-management/deleting-apps.md`**
```markdown
# Deleting Apps

Deleting an app fully uninstalls it from the device. This cannot be undone without reinstalling.

## How to delete an app

1. Go to the **Apps** tab
2. Find the app
3. Click the action menu and select **Delete**
4. Confirm the deletion in the dialog

## When to delete vs hide

Use **Hide** when you want to restrict access but preserve data. Use **Delete** when the app should be fully removed.
```

**`troubleshooting/common-errors.md`**
```markdown
# Common Errors

## "No phone connected"

The app cannot detect a device. Check:
- USB cable is connected and not a charge-only cable
- USB debugging is enabled on the device (Settings → Developer options)
- You approved the USB debugging prompt on the device

## "Failed to install SoberAdmin"

- Make sure all Google accounts have been removed from the device
- Try unplugging and replugging the USB cable
- Restart the device and try again

## "Failed to hide / show app"

SoberAdmin must be installed as device owner for app management to work. Complete the Setup wizard first.
```

**`troubleshooting/no-device.md`**
```markdown
# No Device Found

If Nepsis shows "No device connected" even though a device is plugged in:

## Check USB debugging

1. On the device go to **Settings → About phone**
2. Tap **Build number** 7 times to enable Developer options
3. Go to **Settings → Developer options**
4. Enable **USB debugging**

## Check the USB prompt

When you connect the device, Android shows a prompt asking whether to trust this computer. Tap **Allow**.

## Try a different cable

Some USB cables only carry power and do not support data. Use a cable that came with the device or a known-good data cable.

## Restart ADB

Unplug the device, wait 5 seconds, plug it back in. Nepsis will reconnect automatically.
```

- [ ] **Step 3: Verify all articles load**

Click through every article in the Docs tab — all should render without errors.

- [ ] **Step 4: Commit**

```bash
git add desktop/frontend/src/docs/
git commit -m "feat: add offline documentation content and nav manifest"
```

---

## Final Verification Checklist

- [ ] `npm run dev` starts without errors
- [ ] All 4 sidebar icons navigate to correct tabs, active icon highlights
- [ ] Toggle pill switches light/dark; theme persists after page reload
- [ ] With device connected: status bar shows serial + "Connected" in green
- [ ] Without device: status bar shows "No device connected"
- [ ] SetupTab hero subtitle updates at each wizard step
- [ ] AppsTab search filters live; filter pill cycles All/Visible/Hidden; hidden apps are dimmed
- [ ] InstallTab hero appears above file picker
- [ ] Docs tab: all 8 articles render; no network requests in devtools
- [ ] Window resize: sidebar stays fixed, content scales
- [ ] Version mismatch banner still appears and update flow still works

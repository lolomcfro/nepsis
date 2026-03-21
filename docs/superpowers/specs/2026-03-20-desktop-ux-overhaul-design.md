# Desktop UX Overhaul — Design Spec

**Date:** 2026-03-20
**Scope:** `desktop/` folder only
**Branch:** `ux-overhaul`

---

## Problem

The current desktop UI uses a horizontal tab bar at the top inside a native title bar. It feels dated, is scale-dependent (fixed-width tabs break at small window sizes), and doesn't leverage the desktop form factor. There is no light mode, no documentation section, and the connection status is a single dot with no device context.

## Goal

A modern, sleek layout that:
- Works at any window size without breaking
- Supports light and dark modes with a persistent user preference
- Shows meaningful device connection context
- Adds an offline-capable documentation section

---

## Layout

```
┌──────────────────────────────────────────────────┐
│  [46px sidebar]  │  [content panel]              │
│                  │  ┌───────────────────────┐    │
│  logo            │  │  Hero gradient area   │    │
│                  │  │  Title + subtitle     │    │
│  [setup]         │  ├───────────────────────┤    │
│  [apps]          │  │                       │    │
│  [install]       │  │  Content body         │    │
│  [docs]          │  │                       │    │
│                  │  └───────────────────────┘    │
│  ────────        │                               │
│  [☾ toggle]      │                               │
├──────────────────┴───────────────────────────────┤
│  ● Pixel 9 Pro  │  Android 15        Connected   │
└──────────────────────────────────────────────────┘
```

---

## Components

### `Sidebar.svelte` (new)

A 46px-wide fixed column for the full window height (excluding status bar).

- **Logo** — 26×26px purple gradient square, 8px border-radius, top of sidebar
- **Nav icons** — 4 icons: Setup, Apps, Install, Docs
  - 34×34px tap target, 9px border-radius
  - Active: `--accent-bg` background, `--accent` colored icon
  - Inactive: transparent background, muted icon color
  - Tooltip on hover with section name
- **Dark/light toggle** — 28×16px pill at the very bottom
  - Writes `data-theme` attribute to `<html>` and persists to `localStorage`
- No connection indicator — removed from sidebar, handled by status bar

### `StatusBar.svelte` (modified)

Full-width strip fixed at the bottom of the window, ~28px tall.

**Connected state:**
```
● Pixel 9 Pro  |  Android 15                Connected
```
- Green pulsing dot, device model, separator, OS version, right-aligned "Connected" in green

**Disconnected state:**
```
○ No device connected
```
- Gray dot, muted text

### Hero Area (per tab, not a shared component)

Each tab renders its own hero block at the top of the content panel.

- **Background:** `linear-gradient(135deg, var(--bg-hero-start), var(--bg-hero-end))`
- **Title:** 15px, weight 800, tight letter-spacing, `--text-primary`
- **Subtitle:** 10px, `--accent`, weight 500 — used for step info, hints, or context
- **Bottom border:** `1px solid var(--border-hero)`
- Separated from content body by border only — no extra chrome

### `SetupTab.svelte` (modified)

- Add hero block: title "Setup Your Device", subtitle reflects current step (e.g. "Step 1 of 3 · Remove accounts")
- Remove any standalone section headers that duplicate the hero title
- Content body unchanged functionally

### `AppsTab.svelte` (modified)

- **Hero area:** title "Manage Apps" + search bar + filter pill (All / Visible / Hidden)
  - Search input: ghost style, 26px tall, sits inside the hero gradient
  - Filter pill: right-aligned, cycles between All / Visible / Hidden
- **App list rows:** 30px height
  - 18×18px app icon placeholder (colored by first letter or system icon)
  - App name (truncated)
  - Status badge: green "Visible" / amber "Hidden"
  - Action button (⋯ or chevron) on the right
  - Hidden apps rendered at 55% opacity

### `InstallTab.svelte` (modified)

- Add hero block: title "Install APK", subtitle "Select an APK or bundle file to sideload"
- File picker and status output in content body, unchanged functionally

### `DocsTab.svelte` (new)

Offline-first embedded documentation viewer. No network requests ever made.

**Layout:**
```
┌────────────────────────────────────────────┐
│  Hero: article title + breadcrumb          │
├──────────────────┬─────────────────────────┤
│  Nav tree        │  Rendered Markdown      │
│  (160px)         │  (flex: 1)              │
│                  │                         │
│  Getting Started │  # Article Title        │
│  > Introduction  │                         │
│  > Quick Setup   │  Body text...           │
│                  │                         │
│  Setup Guide     │  ## Section             │
│  > Accounts      │                         │
│  > Backup        │  More content...        │
└──────────────────┴─────────────────────────┘
```

- Markdown files live in `desktop/frontend/src/docs/` — bundled by Vite
- Use `marked` (MIT, ~50kb) to render Markdown to HTML
- Sanitize rendered HTML (strip scripts) before inserting
- Nav tree: categories as collapsible groups, articles as links within
- Active article highlighted in nav
- Hero subtitle shows breadcrumb: e.g. "Getting Started › Introduction"

**Initial doc structure:**
```
desktop/frontend/src/docs/
  index.json          # nav tree manifest
  getting-started/
    introduction.md
    quick-setup.md
  setup-guide/
    accounts.md
    backup.md
  app-management/
    hiding-apps.md
    deleting-apps.md
  troubleshooting/
    common-errors.md
    no-device.md
```

### `App.svelte` (modified)

- Remove horizontal tab bar and title bar chrome
- Compose: `<Sidebar>` (left) + active tab component (right) + `<StatusBar>` (bottom)
- Active tab driven by a `$activeTab` Svelte store
- Theme applied via `data-theme` attribute on `<html>` (set by sidebar toggle)

### `style.css` (modified)

Replace hard-coded color values with CSS custom properties:

```css
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
  --text-muted: #6b7280;
  --accent: #7c6af7;
  --accent-subtle: #a78bfa;
  --accent-bg: #ede9fe;
  --status-bar-bg: #ffffff;
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
  --text-muted: #6b7280;
  --accent: #a78bfa;
  --accent-subtle: #c4b5fd;
  --accent-bg: #2a2040;
  --status-bar-bg: #16161e;
}
```

---

## Dependencies

| Package | Purpose | Size | License |
|---------|---------|------|---------|
| `marked` | Markdown → HTML rendering for Docs tab | ~50kb | MIT |

---

## What Does Not Change

- All Wails Go backend bindings (`wailsjs/`) — untouched
- All business logic in `SetupTab`, `AppsTab`, `InstallTab` — only visual structure changes
- Wails window configuration (`wails.json`) — untouched
- ADB integration, error handling, friendlyError mapping — untouched

---

## Verification

1. `npm run dev` in `desktop/frontend/` — app loads with sidebar layout, no regressions
2. Click each of the 4 sidebar icons — hero area updates, active icon highlights
3. Toggle light/dark — theme switches immediately, persists after page reload
4. With ADB device connected: status bar shows device name + OS + "Connected"
5. Without device: status bar shows "No device connected" in muted style
6. Apps tab: type in search bar — list filters live; toggle filter pill — Visible/Hidden/All works
7. Docs tab: click articles in nav tree — Markdown renders offline, no network tab activity in devtools
8. Resize window to small size — sidebar stays fixed, content scales, nothing breaks

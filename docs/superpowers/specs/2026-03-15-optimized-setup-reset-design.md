# Optimized Setup & Reset Design

**Date:** 2026-03-15
**Status:** Approved

## Background

The current setup flow requires users to manually remove all Google accounts before Device Owner can be granted — a hard Android OS restriction. This causes two problems:

1. **Friction:** Users must navigate Settings manually, with no guidance or automation.
2. **Data loss risk:** Users who tap "Remove data" when removing a Google account lose local contacts permanently. This has already happened to at least one user.

The goal is to keep Device Owner (it enables `setApplicationHidden`, which hides apps while keeping background services — including Play Store updates — running) but make the setup experience as fast and safe as possible. A "Undo Everything" reset path is also needed so users can fully restore their phone.

## Target User

People who have owned their phone for a long time and want to simplify it — fewer distractions, no temptation apps, closer to a "dumb phone" experience without buying new hardware (e.g. Light Phone alternative). They are not starting from a factory reset.

## Architecture Constraint: Mode Abstraction

The commands layer must sit behind a common `AppManager` interface so that a future "non-owner" mode (direct ADB, no Device Owner required) can be added without touching the UI or app management tab. The setup wizard is the only place that is mode-aware. Hide/show/delete operations are mode-agnostic from the user's perspective.

```go
// AppManager is implemented by both DeviceOwnerManager and (future) DirectADBManager.
type AppManager interface {
    ListApps() ([]App, error)
    HideApp(pkg string) error
    ShowApp(pkg string) error
    UninstallApp(pkg string) error
}
```

Setup-specific operations (`CheckAccounts`, `SetDeviceOwner`, `ApplyRestrictions`, `ExportContacts`, `ClearDeviceOwner`) remain on the `Commands` struct and are not part of the `AppManager` interface — they are only called by the wizard and reset flows.

The active setup mode is persisted to a JSON config file at `os.UserConfigDir()/sober/config.json` (resolves to `~/.config/sober/` on Linux, `%APPDATA%\sober\` on Windows, `~/Library/Application Support/sober/` on macOS). If the file is missing at reset time, the app defaults to Device Owner mode.

```go
type Config struct {
    SetupMode    string `json:"setup_mode"`    // "device_owner" | "direct_adb"
    ContactsBackupPath string `json:"contacts_backup_path"` // absolute path or ""
}
```

```
AppManager interface
├── DeviceOwnerManager   (current: broadcasts to SoberAdmin)
└── DirectADBManager     (future: pm disable-user / pm enable)
```

## Feature 1: Redesigned Setup Wizard

Replaces the current static instruction list with a step-by-step wizard. Steps are skipped automatically when not needed (e.g. no accounts present).

### Step 1 — Account Detection (automatic)

Runs `dumpsys account` the moment the phone connects. Checks for `type=com.google` entries.

- If no accounts found: steps 1–3 are skipped entirely, wizard proceeds directly to the install phase.
- If accounts found: display count and proceed to Step 2.

### Step 2 — Contacts Backup (user consent required)

Present a consent screen:

> "Before you remove your Google accounts, we'd like to save a backup of your contacts to this computer. This file stays on your computer only and is never uploaded anywhere."
>
> **[Save backup and continue]** &nbsp;&nbsp; **[Skip — I'll take my chances]**

If the user skips, show a brief warning:
> "If you accidentally choose 'Remove data' during account removal, your local contacts won't be recoverable."

Then proceed regardless.

**Backup implementation:** mirrors the `ListApps` pattern exactly:
1. Desktop app deletes any stale file first: `adb shell run-as com.sober.admin rm -f cache/sober_contacts.vcf`
2. SoberAdmin receives an `EXPORT_CONTACTS` broadcast (`com.sober.EXPORT_CONTACTS`), writes a VCF to its private cache via `getCacheDir()` (not `getExternalCacheDir()`) — this is required for `run-as` access. Output path: `cache/sober_contacts.vcf` relative to the app's data root.
3. Desktop app polls via `adb shell run-as com.sober.admin cat cache/sober_contacts.vcf` with a 15-second deadline at 250ms intervals (same as `ListApps`).
4. On success, the VCF content is written to `<os.UserConfigDir()>/sober/contacts-backup-<timestamp>.vcf` on the desktop.
5. The full path is shown to the user. The path is also saved to `config.json` as `contacts_backup_path`.
6. Desktop app deletes the cache file: `adb shell run-as com.sober.admin rm -f cache/sober_contacts.vcf`

This requires a new broadcast action and handler in the SoberAdmin APK.

### Step 3 — Guided Account Removal

Display:
- How many accounts remain (live count, polled every 2 seconds via `dumpsys account`)
- A single **"Open Account Settings on my phone"** button that deep-links to the Android Accounts settings screen via `adb shell am start -a android.settings.SYNC_SETTINGS`
- A prominent warning box:

> "When Android asks what to do with your data — choose **Keep**. Your contacts are safe either way (we already backed them up), but keeping them avoids any extra steps."

The counter updates in real time: *"2 accounts remaining → 1 account remaining → All accounts removed ✓"*

When the count reaches zero, the wizard auto-proceeds to the install phase with no user action required.

**Disconnect handling:** If the phone disconnects while polling in Steps 1–3, the wizard pauses and shows: *"Phone disconnected — plug it back in to continue."* The wizard resumes from the same step when the phone reconnects. It does not restart from the beginning.

If the phone disconnects during the install phase (Steps 4–6), the wizard cannot safely resume mid-install. It shows an error: *"Phone disconnected during setup."* The "Try Again" path restarts from Step 1. This is safe because `InstallAPK` uses the `-r` (reinstall) flag, so re-running it on an already-partially-installed APK will not fail.

### Install Phase (Steps 4–6) — Single Spinner, No User Action

The UI shows a single spinner: *"Setting up SoberAdmin — do not unplug your phone…"*

Internally three operations run in sequence:
1. Install SoberAdmin APK
2. Set Device Owner (`dpm set-device-owner`)
3. Apply baseline restrictions (`com.sober.APPLY_RESTRICTIONS` broadcast)

These are not shown as separate UI steps — they are an atomic phase from the user's perspective. Any failure surfaces a clear error with a "Try Again" option that restarts from Step 1.

### Step 7 — Re-add Google Account

After setup completes, the wizard shows a success state:

> "Setup complete! You can now re-add your Google account."

A button opens Account Settings on the phone via `adb shell am start -a android.settings.SYNC_SETTINGS`. The app explains that Google-synced contacts will reappear automatically once signed back in.

The Setup tab then transitions to its persistent post-setup state (see Reset section below).

## Feature 2: Reset Flow ("Undo Everything")

Once setup is complete (Device Owner active), the Setup tab shows a persistent success banner and a **Reset Phone** button below it. The button is always visible in the post-setup state, not buried in a menu.

### Confirmation Dialog

> "This will remove all restrictions, show all hidden apps, and remove SoberAdmin as Device Owner. Are you sure?"
>
> **[Reset Everything]** &nbsp;&nbsp; **[Cancel]**

### Reset Steps (strictly ordered, automated, shown with progress)

Steps must execute in this order — Step 2 depends on Device Owner still being active:

1. **Show all hidden apps** — call `ListApps()`, filter by `Hidden: true`, call `ShowApp` on each. Must happen before Step 2 because `ListApps` requires Device Owner.
2. **Remove Device Owner** — send `com.sober.CLEAR_DEVICE_OWNER` broadcast to SoberAdmin, which calls `clearDeviceOwnerApp()` on itself and unregisters the admin receiver. This requires a new broadcast action and handler in the SoberAdmin APK.
3. **Restore contacts (optional)** — shown only if `config.json` has a non-empty `contacts_backup_path` pointing to an existing file:
   > "We have a contacts backup from [date]. Restore it to your phone?"
   > **[Restore]** &nbsp;&nbsp; **[Skip]**

   **Restore implementation:** Push the VCF to the phone (`adb push <path> /sdcard/sober_contacts_restore.vcf`), then broadcast `com.sober.IMPORT_CONTACTS` to SoberAdmin, which reads the file and inserts contacts via `ContactsContract`. This avoids the `file://` URI restriction (Android 7+ blocks `file://` URIs across process boundaries via Intent). SoberAdmin deletes `/sdcard/sober_contacts_restore.vcf` after import. This requires a third new broadcast action in the SoberAdmin APK (see table below). If the backup file is missing, this step is silently skipped.

4. **Confirm completion:**
   > "Your phone has been fully restored. SoberAdmin is no longer active."

   The Setup tab resets to its pre-setup wizard state.

### Mode Awareness

The reset flow reads `setup_mode` from `config.json` and executes the correct teardown. When the future non-owner mode is added, it provides its own teardown behind the same interface.

## What Is Not Changing

- The Apps tab (hide/show/delete) is unchanged in behavior; its backend is refactored to use the `AppManager` interface
- The Install tab is unchanged
- Device Owner remains the default and only setup mode for now
- The SoberAdmin APK's existing broadcast interface is extended (not replaced)

## SoberAdmin APK Changes Required

Two new broadcast actions must be added to the SoberAdmin APK:

| Action | Description |
|---|---|
| `com.sober.EXPORT_CONTACTS` | Reads all contacts via ContentResolver, writes VCF to `getCacheDir()/sober_contacts.vcf` (private cache, accessible via `run-as`) |
| `com.sober.IMPORT_CONTACTS` | Reads `/sdcard/sober_contacts_restore.vcf`, inserts contacts via `ContactsContract`, then deletes the file |
| `com.sober.CLEAR_DEVICE_OWNER` | Calls `DevicePolicyManager.clearDeviceOwnerApp()` and removes admin receiver |

## Out of Scope

- Non-owner (direct ADB) setup mode — architecture accommodates it, implementation deferred
- Multi-account-type support (non-Google accounts) — `CheckAccounts` intentionally only blocks on Google accounts
- Automated re-adding of Google account — not possible via ADB, user must do this manually

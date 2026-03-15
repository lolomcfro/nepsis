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

The active setup mode is persisted in app state so the reset flow knows exactly how to undo things.

```
AppManager interface
├── DeviceOwnerManager   (current: broadcasts to SoberAdmin)
└── DirectADBManager     (future: pm disable-user / pm enable)
```

## Feature 1: Redesigned Setup Wizard

Replaces the current static instruction list with a step-by-step wizard. Steps are skipped automatically when not needed (e.g. no accounts present).

### Step 1 — Account Detection (automatic)

Runs `dumpsys account` the moment the phone connects. Checks for `type=com.google` entries.

- If no accounts found: step is skipped, wizard proceeds to Step 4.
- If accounts found: display count and proceed to Step 2.

### Step 2 — Contacts Backup (user consent required)

Before the user removes any accounts, present a consent screen:

> "Before you remove your Google accounts, we'd like to save a backup of your contacts to this computer. This file stays on your computer only and is never uploaded anywhere."
>
> **[Save backup and continue]** &nbsp;&nbsp; **[Skip — I'll take my chances]**

If the user skips, show a brief warning:
> "If you accidentally choose 'Remove data' during account removal, your local contacts won't be recoverable."

Then proceed regardless.

**Backup implementation:** SoberAdmin receives an `EXPORT_CONTACTS` broadcast, writes a `.vcf` file to its cache directory, and the desktop app pulls it via `adb pull`. The file is stored in the desktop app's data directory with a timestamp. The path is shown to the user.

### Step 3 — Guided Account Removal

Display:
- How many accounts remain (live count, polled every 2 seconds via `dumpsys account`)
- A single **"Open Account Settings on my phone"** button that deep-links to the Android Accounts settings screen via ADB
- A prominent warning box:

> "When Android asks what to do with your data — choose **Keep**. Your contacts are safe either way (we already backed them up), but keeping them avoids any extra steps."

The counter updates in real time: *"2 accounts remaining → 1 account remaining → All accounts removed ✓"*

When the count reaches zero, the wizard auto-proceeds to Step 4 with no user action required.

### Steps 4–6 — Automated (no user action)

Runs silently with a spinner:

1. Install SoberAdmin APK
2. Set Device Owner (`dpm set-device-owner`)
3. Apply baseline restrictions (`APPLY_RESTRICTIONS` broadcast)

Any failure surfaces a clear error with a "Try Again" option.

### Step 7 — Re-add Google Account

After setup completes:

> "Setup complete! You can now re-add your Google account."

A button opens Account Settings on the phone via ADB. The app explains that Google-synced contacts will reappear automatically once signed back in.

## Feature 2: Reset Flow ("Undo Everything")

A **Reset Phone** button is shown on the Setup tab once setup is complete (Device Owner active). Positioned clearly but not prominently — below the success state, not in the primary action area.

### Confirmation Dialog

> "This will remove all restrictions, show all hidden apps, and remove SoberAdmin as Device Owner. Are you sure?"
>
> **[Reset Everything]** &nbsp;&nbsp; **[Cancel]**

### Reset Steps (automated, shown with progress)

1. **Show all hidden apps** — iterate the hidden app list and call `ShowApp` on each
2. **Remove Device Owner** — broadcast `CLEAR_DEVICE_OWNER` to SoberAdmin, which calls `clearDeviceOwnerApp()` on itself
3. **Restore contacts (optional)** — shown only if a backup file exists on this computer:
   > "We have a contacts backup from [date]. Restore it to your phone?"
   > **[Restore]** &nbsp;&nbsp; **[Skip]**
   If the backup is missing or already restored, this step is silently skipped.
4. **Confirm completion:**
   > "Your phone has been fully restored. SoberAdmin is no longer active."

### Mode Awareness

The reset flow reads the persisted setup mode and executes the correct teardown for that mode. When the future non-owner mode is added, it provides its own teardown implementation behind the same interface.

## What Is Not Changing

- The Apps tab (hide/show/delete) is unchanged
- The Install tab is unchanged
- Device Owner remains the default and only setup mode for now
- The SoberAdmin APK's existing broadcast interface is extended (not replaced)

## Out of Scope

- Non-owner (direct ADB) setup mode — architecture accommodates it, implementation deferred
- Multi-account-type support (non-Google accounts) — current `CheckAccounts` already only blocks on Google accounts, this is intentional
- Automated re-adding of Google account — not possible via ADB, user must do this manually

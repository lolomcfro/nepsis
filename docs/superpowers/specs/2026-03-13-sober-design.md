# Sober — Design Spec
**Date:** 2026-03-13

## Problem

Staying focused and avoiding addictive content on an Android phone is difficult when the phone itself provides unrestricted access to any app or website. Existing blocker apps (Accessibility Service or Usage Stats-based) are trivially defeated by disabling them from Settings. Deleting an app while the Play Store remains accessible leads to immediate reinstallation.

## Minimum Requirements

- **Android:** API 26 (Android 8.0 Oreo) or higher. `DISALLOW_INSTALL_UNKNOWN_SOURCES` per-app behavior and `setApplicationHidden()` semantics are stable from API 26 onward.
- **Laptop OS:** Windows 10+, macOS 11+, or Linux (any modern distro with ADB available)
- **ADB:** Bundled with the desktop app or expected on PATH

## Goal

A defeat-proof Android phone lockdown system controlled exclusively from a laptop over USB. Once set up, no changes can be made from the phone itself — not even by a determined, tempted user.

---

## Architecture

Two components, zero stored state:

```
┌─────────────────────────────┐         USB/ADB
│  Laptop                     │ ──────────────────► Android Phone
│                             │                     ┌────────────────────┐
│  Sober (Wails desktop app)  │  adb shell am       │ SoberAdmin APK     │
│  - stateless control panel  │  broadcast ───────► │ - Device Owner     │
│  - queries phone live       │                     │ - BroadcastReceiver│
│  - bundles SoberAdmin APK   │                     │ - No UI, no icon   │
│                             │                     │ - Applies DPM      │
└─────────────────────────────┘                     │   policies         │
                                                     └────────────────────┘
```

**No data is stored anywhere.** The phone's live state is always the source of truth. The Wails app queries it fresh on every connection.

---

## Component 1: Sober Desktop App (Wails + Go)

### Distribution
- Single downloadable binary per platform (Windows, Mac, Linux)
- Built with [Wails](https://wails.io/) — Go backend, web frontend
- SoberAdmin APK bundled inside the binary via Go's `embed` package
- Platform-specific ADB binary bundled via Go's `embed` package — no separate Android Platform Tools install required
- Expected download size: ~20-25 MB per platform
- No installer, no dependencies, no runtime required

### GUI Screens

**Setup Tab**
- Shown prominently if SoberAdmin is not yet installed as Device Owner
- Step-by-step wizard (see Setup Flow below)

**Apps Tab**
- Searchable list of all installed apps (name + icon)
- Toggle per app: Visible / Hidden
- Toggling immediately fires ADB broadcast to phone
- Disabled when no phone is connected
- Status bar shows connection state at all times
- App names and icons are fetched from the phone via the `LIST_APPS` protocol (see below)

**Install Tab**
- File picker to select an APK from the laptop
- "Install to Phone" button → `adb install <apk>`
- Only way to add new apps once Play Store is hidden
- Shows success/failure feedback

### Phone Connection Detection
- The desktop app polls `adb devices` on a short interval (every 2 seconds)
- On connection detected: fetch live app state and render Apps tab
- On disconnection detected: disable all controls, show disconnected status
- If an ADB command fails mid-operation (e.g. cable unplugged), show an error and re-poll

### Statelessness
- No database, no config file, no Wails storage
- On every USB connection: query phone live via ADB, render current state
- Changes are applied immediately and reflected in live state
- If another device installs or restores an app via ADB, Sober shows accurate state on next connection

---

## Component 2: SoberAdmin APK (Android)

### Properties
- No launcher icon
- No UI of any kind
- No activity, no service, no persistent process
- Cannot be uninstalled while Device Owner is active
- Removal requires factory reset or ADB from a laptop

### Components

**`AdminReceiver`** — extends `DeviceAdminReceiver`. Grants Device Owner status during setup.

**`CommandReceiver`** — `BroadcastReceiver` listening for ADB-issued intents. The receiver is **not exported** (`android:exported="false"`) and is protected with a signature-level permission (`android:permission="com.sober.SEND_COMMAND"`). Only the `adb shell am broadcast` invocation (which runs as the shell user, bypassing app-level permission checks) can trigger it. Installed apps on the phone cannot send these intents.

| Intent Action | Extras | Effect |
|---|---|---|
| `com.sober.HIDE_APP` | `package: String` | Calls `setApplicationHidden(pkg, true)` |
| `com.sober.SHOW_APP` | `package: String` | Calls `setApplicationHidden(pkg, false)` |
| `com.sober.LIST_APPS` | — | See LIST_APPS Protocol below |
| `com.sober.APPLY_RESTRICTIONS` | — | Calls `addUserRestriction(DISALLOW_INSTALL_UNKNOWN_SOURCES)` |

All policy enforcement is via `DevicePolicyManager.setApplicationHidden()` — enforced at the system level, cannot be overridden from the phone.

### LIST_APPS Protocol

When `com.sober.LIST_APPS` is received, `CommandReceiver`:

1. Queries all installed packages and their hidden state
2. Resolves human-readable app label and base64-encoded icon for each package
3. Writes a JSON file to `/data/local/tmp/sober_apps.json` in the format:
   ```json
   [
     {
       "package": "com.android.dialer",
       "label": "Phone",
       "icon": "<base64-encoded PNG, scaled to 48dp before encoding>",
       "hidden": false
     }
   ]
   ```
   Icons are scaled to 48×48dp before base64 encoding to keep file size manageable. On a typical phone with 200 apps this produces a file of roughly 1-3 MB, which is acceptable for a one-time-per-connection fetch over USB.

4. The desktop app polls for `/data/local/tmp/sober_apps.json` via `adb shell cat` with a 10-second timeout. If the file does not appear within 10 seconds, the desktop app shows an error ("Failed to read app list from phone") and allows the user to retry. This handles slow devices or APK failures gracefully.
5. The desktop app deletes the file after reading: `adb shell rm /data/local/tmp/sober_apps.json`

This is transient — the file exists only for the duration of a single query cycle. It does not constitute stored state.

### App List Filtering and Display

The LIST_APPS response includes all installed packages. The desktop app filters and sorts before display:

- **Filter:** Only show packages that have a launcher intent (`CATEGORY_LAUNCHER`) or are user-installed (`pm list packages -3`). Low-level system components (e.g. `com.android.providers.contacts`, `com.qualcomm.qti.*`) are excluded.
- **Sort:** Alphabetically by app label
- **Grouping:** User-installed apps first, then system apps with launchers, separated by a divider

This reduces ~200-300 raw packages to a manageable list of ~30-80 apps depending on the device.

### Hidden State After App Updates

`setApplicationHidden()` state can be reset on some Android versions when a package is updated. To handle this, `SoberAdmin` registers a `BroadcastReceiver` for `android.intent.action.PACKAGE_REPLACED`. When an installed package is updated, the handler calls `isApplicationHidden()` immediately upon receipt. If the system has reset the hidden flag (returns `false` for an app that was hidden), it re-applies `setApplicationHidden(pkg, true)`.

**Limitation:** `PACKAGE_REPLACED` fires after the update has completed. Whether `isApplicationHidden()` reflects the pre-update or post-update state at that moment is version-dependent. This handler is a best-effort mitigation. In the worst case, a previously-hidden app briefly reappears in the launcher after an update until the next time the user opens Sober and manually re-hides it. This is an accepted limitation.

### SoberAdmin Upgrades

If a new version of SoberAdmin is shipped with an updated Sober desktop app, the Setup tab will detect that the installed SoberAdmin version does not match the bundled version and prompt the user to upgrade. The upgrade is performed via `adb install -r` (replace existing installation). Device Owner status is preserved across `adb install -r` upgrades.

### Play Store
- Hidden (not disabled or uninstalled) via `setApplicationHidden()`
- Background auto-updates for existing apps continue to work
- Play Store UI is inaccessible from the phone
- New apps can only be installed via `adb install` from the Sober desktop app

### Unknown Sources Restriction
During setup, SoberAdmin applies `DISALLOW_INSTALL_UNKNOWN_SOURCES` via `DevicePolicyManager.addUserRestriction()`. This prevents sideloading APKs from the phone (e.g. from a file manager or browser). This restriction is applied as part of Step 5 of the setup flow (Apply Baseline Restrictions), immediately after Device Owner is granted.

### Multi-User and Work Profiles
`setApplicationHidden()` operates per-user. If the phone has a work profile or secondary user, hidden apps in the primary profile may remain visible in other profiles. Sober only manages the primary user (user 0). Work profiles and multi-user scenarios are out of scope.

---

## Setup Flow

Triggered from the Setup tab on first run:

```
1. Pre-flight instructions
   └─ Display step-by-step instructions:
       a. "Remove all Google accounts from your phone before continuing"
          (Settings → Accounts → Google → Remove account)
          NOTE: dpm set-device-owner fails if any accounts are present
       b. "Enable Developer Mode on your phone"
          - Settings → About Phone → tap Build Number 7 times
          - Settings → Developer Options → Enable USB Debugging
   └─ "Plug in your phone and click Continue"

2. Detect phone
   └─ adb devices
   └─ Fail with clear message if not found or not authorized
   └─ Detect if Google accounts are still present — fail with remediation message if so
   └─ Phone displays "Allow USB Debugging?" prompt → user taps Allow
   └─ Confirm authorization before proceeding

3. Install SoberAdmin APK
   └─ Extract bundled APK from embedded bytes to temp file
   └─ adb install sober-admin.apk

4. Grant Device Owner
   └─ adb shell dpm set-device-owner com.sober.admin/.AdminReceiver
   └─ Fail with clear message if accounts still present (second check)

5. Apply baseline restrictions
   └─ Broadcast com.sober.APPLY_RESTRICTIONS (DISALLOW_INSTALL_UNKNOWN_SOURCES)

6. Guided app review
   └─ Send com.sober.LIST_APPS → poll for sober_apps.json (10s timeout)
   └─ On timeout: show error with retry button; setup remains at this step
   └─ On success: display filtered, sorted app list
   └─ User toggles Visible/Hidden for each app
   └─ Sensible defaults:
       Visible: Phone, Messages, Camera, Maps, Calculator, Settings
       Hidden: Play Store, browser(s), social media (if detected)

7. Apply selections
   └─ Broadcast HIDE_APP for each hidden app
   └─ Confirm each policy applied successfully

8. Setup complete
   └─ Phone is now locked down
   └─ Transitions to Apps tab showing live state
```

### Laptop Migration

If the user gets a new laptop or needs to manage the phone from a different machine:

1. Download Sober on the new laptop
2. Enable USB Debugging on the phone (it may already be on)
3. Plug in — phone will prompt "Allow USB Debugging from this computer?" → tap Allow
4. Sober detects SoberAdmin is already installed as Device Owner and skips setup
5. Full control is restored from the new laptop

No factory reset required. ADB authorization is per-machine, so each new laptop requires the one-time phone prompt.

---

## Security Model

| Attack Vector | Mitigation |
|---|---|
| Open blocked app from phone | App is hidden — not visible in launcher or app drawer |
| Re-enable Play Store via Settings | Device Owner policy cannot be overridden from Settings |
| Uninstall SoberAdmin from phone | Blocked while Device Owner is active |
| Sideload APK from phone | `DISALLOW_INSTALL_UNKNOWN_SOURCES` enforced by Device Owner |
| Send malicious broadcast to CommandReceiver | Receiver is not exported; only ADB shell user can trigger it |
| Use ADB from another laptop | Requires physical USB access + explicit authorization prompt on the phone |
| Factory reset | Removes Device Owner — deliberate escape hatch, wipes the phone |
| App becomes visible after background update | `PACKAGE_REPLACED` receiver re-applies hidden state automatically |

Developer Options remain enabled after setup (USB Debugging stays on) — this is intentional to avoid friction. The primary security barrier is physical: you need a laptop, a USB cable, and ADB authorization.

---

## Out of Scope (Future)

- DNS-level content filtering
- Password-protected unlock (accountability partner model)
- Time-lock removal requests
- Multiple device profiles
- Scheduled lockdown windows
- Web-based control panel
- Work profile / multi-user management

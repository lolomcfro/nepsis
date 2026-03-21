# Design: Non-Technical UX Overhaul

**Date:** 2026-03-16
**Status:** Draft

---

## Context

The current Sober setup experience uses accurate but alienating technical terminology: "USB debugging," "Device Owner," "ADB commands," and error codes from failed shell commands. The target user — someone voluntarily locking their own phone to support addiction recovery — is motivated but non-technical. The terminology makes the app feel like spyware or a hacking tool, which creates fear and abandonment.

The goal of this design is to make the setup feel safe and understandable, with honest trust messaging, without any changes to Android code or backend logic.

---

## Framing Principles

The person doing setup IS the phone owner. They are locking their own phone. There is no required accountability partner. The laptop is the **friction layer** — the protection comes from requiring deliberate physical effort (USB cable, computer) to make changes, not from any cryptographic lock to a specific machine.

Accurate trust message:
> "Hiding or changing app restrictions requires your phone to be physically connected to a computer via USB cable — making it much harder to undo settings on impulse."

This is honest (any computer with the Sober app and USB access can manage it) while still communicating the meaningful friction that is the product's core value.

---

## 1. Language Overhaul

All technical terms are replaced throughout the UI:

| Current term | Replacement |
|---|---|
| USB debugging | "Computer connection" |
| Device Owner | "Accountability Mode" |
| SoberAdmin | "Accountability Manager" |
| Set device owner | "Activate" |
| Apply restrictions | "Lock settings" |
| ADB / adb | Never shown in UI |
| Error codes / stack traces | Plain-English messages per known failure |

---

## 2. Trust Transparency Screen

A new section shown before setup begins (or accessible from a persistent "How does this work?" link). Content must be accurate given the app's existing capabilities.

**What Sober does:**
- Hides or locks apps you choose
- Prevents installing new apps
- Requires a USB cable and computer to make changes — adding friction between you and impulsive decisions
- Backs up your contacts to this computer at your request (before setup, with your consent)

**What Sober cannot do:**
- Read your messages or emails
- Access your photos or files
- Track your location
- Access your passwords or accounts
- Send any data to a server (works entirely offline, no account required)

**Who can make changes:**
- Anyone with your phone physically in hand, a USB cable, and a computer with Sober installed

Note: Contacts are explicitly listed as a user-consented backup feature — not lumped under "cannot access" — because the app holds READ_CONTACTS / WRITE_CONTACTS permissions and the backup feature is real.

---

## 3. Per-Step Copy Rewrites

### Step: Back up contacts
> "Before we begin, let's make a backup of your contacts on this computer — just in case."

### Step: Connect phone via USB
> "Plug your phone into this computer with a USB cable. Make sure your phone is unlocked."

### Step: Enable computer connection (USB debugging)
> "Allow your phone to connect to this computer. This one-time step lets your computer install the accountability software. It can be turned off again after setup if you prefer."
>
> *How to do it: [inline step-by-step guide for Developer Options]*

### Step: Remove Google account
> "Android requires this as a security measure. Sign out of your Google account on the phone before continuing — your Gmail, Drive, and Google data stay safe and accessible from any browser or other devices."
>
> *This is Android's way of ensuring that only someone with physical access to your phone can activate deep restrictions.*

### Step: Install accountability app
> "Installing a small background app that enforces your settings. It won't appear in your app list on most phones."

Note: "on most phones" hedges the launcher-dependent behavior (`android:icon="@null"`, `android:label="@null"`). Some OEM launchers may show it without an icon.

### Step: Activate accountability mode
> "Activating. This gives the app system-level control so your restrictions can't be bypassed from the phone itself."

### Step: Choose apps to restrict
> "Select which apps to hide. You can change this any time from your computer."

---

## 4. Error Message Overhaul

Current error messages expose raw ADB output. New messages map known failure codes to plain English:

| Condition | New message |
|---|---|
| No device connected | "No phone detected. Make sure your USB cable is connected and your phone is unlocked." |
| ADB unauthorized | "Your phone is asking for permission. Check your phone screen and tap 'Allow'." |
| Device owner already set (SoberAdmin) | "Accountability Mode is already active on this phone." |
| Device owner already set (different app) | "Another app is controlling this phone. It must be removed before Sober can be set up." |
| Google accounts present | "Please sign out of your Google account first. Tap 'How to do this' for instructions." |
| Timeout waiting for result | "This is taking longer than expected. Make sure your phone is unlocked and try again." |

Note: `IllegalStateException` from `dpm set-device-owner` must be disambiguated — "SoberAdmin is already owner" vs "a different app is owner" — by checking `dpm list-owners` output before attempting to set. This is a Go-side change only (no Android code change).

---

## 5. Files to Modify

**Desktop Frontend only — no Android or Go backend changes:**
- `desktop/frontend/src/components/SetupTab.svelte` — rewrite all step copy, add trust transparency section, improve error display, distinguish device-owner-conflict errors

**Desktop Go (error disambiguation only):**
- `desktop/adb/commands.go` — pre-check `dpm list-owners` before `dpm set-device-owner` to distinguish "already set by SoberAdmin" from "set by different app", return typed errors the frontend can map to specific messages

---

## 6. Verification

- [ ] Walk through full setup flow end-to-end: confirm no technical terms appear at any step
- [ ] Trust transparency screen accurately describes contacts as a consented backup, not claimed as inaccessible
- [ ] Trigger no-device, ADB-unauthorized, Google-accounts-present error conditions: confirm correct friendly message shown
- [ ] Trigger device-owner-already-set with SoberAdmin: confirm "already active" message
- [ ] Trigger device-owner-already-set with a different app: confirm "another app is controlling" message (not the SoberAdmin message)

# Non-Technical UX Overhaul Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace all technical jargon in the Sober setup UI with plain-English copy, add a trust transparency section, and return friendly error messages for known failure modes.

**Architecture:** Two changes: (1) Go-side `SetDeviceOwner()` gains a pre-check that detects whether SoberAdmin or a different app is already Device Owner, returning friendly error strings. (2) The Svelte `SetupTab.svelte` gets its copy rewritten, a trust transparency panel added, and a `friendlyError()` mapper for the raw error strings that can still escape from Go.

**Tech Stack:** Go 1.23 (backend, `desktop/adb/`), Svelte + TypeScript (frontend, `desktop/frontend/`)

---

## Chunk 1: Go — SetDeviceOwner Error Disambiguation

### Task 1: Add failing tests for device-owner conflict detection

**Files:**
- Modify: `desktop/adb/commands_test.go`

- [ ] **Step 1: Update existing `fakeRunner` sub-tests and add two new failing test cases**

Open `desktop/adb/commands_test.go`. The existing `"success"` and `"output contains error"` sub-tests use `fakeRunner`, which returns the same output for every call. After Step 3 adds a `list-owners` pre-check to `SetDeviceOwner`, those tests will break because the pre-check receives the same output as the `set-device-owner` call. Fix them first by switching to `callTrackingRunner`.

Replace the `"success"` sub-test:
```go
t.Run("success", func(t *testing.T) {
    fake := &callTrackingRunner{
        responses: map[string]string{
            "list-owners":      "{}",
            "set-device-owner": "Success",
        },
    }
    c := adb.NewCommands(fake)
    if err := c.SetDeviceOwner(); err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
})
```

Replace the `"output contains error"` sub-test:
```go
t.Run("output contains error", func(t *testing.T) {
    fake := &callTrackingRunner{
        responses: map[string]string{
            "list-owners":      "{}",
            "set-device-owner": "Error: already set",
        },
    }
    c := adb.NewCommands(fake)
    if err := c.SetDeviceOwner(); err == nil {
        t.Fatal("expected error for error output, got nil")
    }
})
```

The `"runner error"` and `"accounts on device"` sub-tests use `fakeRunner` with an `err` set — `checkExistingDeviceOwner` fails open on runner error, so these still work unchanged.

Then after the existing `"accounts on device"` sub-test (line 197), add the two new cases:

```go
t.Run("sober-admin already device owner", func(t *testing.T) {
    fake := &callTrackingRunner{
        responses: map[string]string{
            "list-owners": "com.sober.admin/.AdminReceiver (User 0)",
        },
    }
    c := adb.NewCommands(fake)
    err := c.SetDeviceOwner()
    if err == nil {
        t.Fatal("expected error, got nil")
    }
    if !strings.Contains(err.Error(), "Accountability Mode is already active") {
        t.Errorf("expected friendly already-active message, got: %v", err)
    }
})

t.Run("different app is device owner", func(t *testing.T) {
    fake := &callTrackingRunner{
        responses: map[string]string{
            "list-owners": "com.other.mdm/.AdminReceiver (User 0)",
            "set-device-owner": "", // won't be reached
        },
    }
    c := adb.NewCommands(fake)
    err := c.SetDeviceOwner()
    if err == nil {
        t.Fatal("expected error, got nil")
    }
    if !strings.Contains(err.Error(), "Another app is controlling") {
        t.Errorf("expected other-owner message, got: %v", err)
    }
})
```

- [ ] **Step 2: Run the new tests to confirm they fail**

```bash
cd /home/logan/projects/sober/desktop && go test ./adb/... -run TestSetDeviceOwner -v
```

Expected: the two new sub-tests FAIL (function does not distinguish these cases yet).

---

### Task 2: Implement disambiguation in SetDeviceOwner

**Files:**
- Modify: `desktop/adb/commands.go`

- [ ] **Step 3: Add `checkExistingDeviceOwner` helper and update `SetDeviceOwner`**

In `commands.go`, add the helper function immediately before `SetDeviceOwner` (after `CheckAccounts`, around line 178):

```go
// checkExistingDeviceOwner returns a user-friendly error if any device owner is
// already set, nil otherwise. Fails open on runner error.
func (c *Commands) checkExistingDeviceOwner() error {
	out, err := c.runner.Run("shell", "dpm", "list-owners")
	if err != nil {
		return nil // can't check; the actual set-device-owner call will fail if needed
	}
	out = strings.TrimSpace(out)
	if out == "" || out == "{}" {
		return nil
	}
	if strings.Contains(out, "com.sober.admin") {
		return fmt.Errorf("Accountability Mode is already active on this phone.")
	}
	return fmt.Errorf("Another app is controlling this phone. It must be removed before Sober can be set up.")
}
```

Then update `SetDeviceOwner` to call it first. Replace the function signature + opening (lines 180–206) with:

```go
// SetDeviceOwner grants Device Owner to SoberAdmin.
func (c *Commands) SetDeviceOwner() error {
	if err := c.checkExistingDeviceOwner(); err != nil {
		return err
	}

	const maxRetries = 5
	var out string
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		out, err = c.runner.Run(
			"shell", "dpm", "set-device-owner",
			"com.sober.admin/.AdminReceiver",
		)
		if err == nil {
			break
		}
		if strings.Contains(err.Error(), "there are already some accounts on the device") && attempt < maxRetries-1 {
			time.Sleep(1 * time.Second)
			continue
		}
		if strings.Contains(err.Error(), "there are already some accounts on the device") {
			return fmt.Errorf("Google accounts are still on this device — " +
				"go to Settings › Accounts and remove them all, then try again")
		}
		return err
	}
	if strings.Contains(strings.ToLower(out), "error") {
		return fmt.Errorf("set-device-owner failed: %s", out)
	}
	return nil
}
```

- [ ] **Step 4: Run all tests to confirm new cases pass and nothing regressed**

```bash
cd /home/logan/projects/sober/desktop && go test ./adb/... -v
```

Expected: all tests PASS including the two new sub-tests.

- [ ] **Step 5: Commit**

```bash
cd /home/logan/projects/sober
git add desktop/adb/commands.go desktop/adb/commands_test.go
git commit -m "feat: distinguish sober-admin vs other-app device owner conflict in SetDeviceOwner"
```

---

## Chunk 2: Frontend — SetupTab UX Overhaul

### Task 3: Language overhaul + per-step copy rewrites

**Files:**
- Modify: `desktop/frontend/src/components/SetupTab.svelte`

This task rewrites all user-visible text in the file. No logic changes — only string content.

- [ ] **Step 6: Replace the post-setup success banner (line 211)**

Find:
```svelte
      SoberAdmin is installed and active as Device Owner. Your phone is locked down.
```
Replace with:
```svelte
      Accountability Manager is active. Your phone is in Accountability Mode.
```

- [ ] **Step 7: Replace the Reset button label (line 220)**

Find:
```svelte
          Reset Phone
```
Replace with:
```svelte
          Remove Accountability Mode
```

- [ ] **Step 8: Replace the reset confirmation text (lines 227–228)**

Find:
```svelte
        <p>This will remove all restrictions, show all hidden apps, and remove SoberAdmin as Device Owner.</p>
```
Replace with:
```svelte
        <p>This will show all hidden apps and remove Accountability Mode from your phone.</p>
```

- [ ] **Step 9: Replace the reset-complete message (line 285)**

Find:
```svelte
        <p class="step-lead">Reset complete. SoberAdmin is no longer active.</p>
```
Replace with:
```svelte
        <p class="step-lead">Done. Accountability Mode has been removed from your phone.</p>
```

- [ ] **Step 10: Replace the "guide-removal" step-lead and warn-box (lines 327–335)**

Find:
```svelte
          <p class="step-lead">
            {accountCount === 1 ? '1 Google account' : `${accountCount} Google accounts`}
            {accountCount === 1 ? 'needs' : 'need'} to be removed before setup can continue.
          </p>

          <div class="warn-box">
            <strong>Important:</strong> When Android asks to confirm account removal, just tap <strong>Remove account</strong>.
            {backupPath ? ' Your contacts are backed up to this computer and will be restored after setup.' : ' If your contacts are only stored locally on this phone, they won\'t be affected.'}
          </div>
```
Replace with:
```svelte
          <p class="step-lead">
            You have {accountCount === 1 ? '1 Google account' : `${accountCount} Google accounts`} on this phone.
            Android requires these to be removed before Accountability Mode can be activated — your Google data stays safe and accessible from any browser.
          </p>

          <div class="warn-box">
            <strong>When prompted on your phone:</strong> tap <strong>Remove account</strong>.
            {backupPath ? ' Your contacts are backed up to this computer.' : ''}
          </div>
```

- [ ] **Step 11: Replace the "Open Account Settings" button label (line 337)**

Find:
```svelte
            Open Account Settings on my phone
```
Replace with:
```svelte
            Open Account Settings on Phone
```

- [ ] **Step 12: Replace the "ready-to-install" step text and button (lines 354–357)**

Find:
```svelte
        <p class="step-lead">Ready to install SoberAdmin on your phone.</p>
        <div class="button-col">
          <button class="primary" on:click={doInstall}>Install SoberAdmin</button>
        </div>
```
Replace with:
```svelte
        <p class="step-lead">Ready to activate Accountability Mode on your phone.</p>
        <div class="button-col">
          <button class="primary" on:click={doInstall}>Set Up</button>
        </div>
```

- [ ] **Step 13: Replace the "installing" progress message (line 368)**

Find:
```svelte
          <p>Setting up SoberAdmin — do not unplug your phone…</p>
```
Replace with:
```svelte
          <p>Setting up Accountability Mode — do not unplug your phone…</p>
```

- [ ] **Step 14: Replace the success banner (lines 374–376)**

Find:
```svelte
      <div class="banner success">
        Setup complete! Your phone is now locked down.<br>
        Switch to the <strong>Apps</strong> tab to manage app visibility.
      </div>
```
Replace with:
```svelte
      <div class="banner success">
        Done! Accountability Mode is active.<br>
        Switch to the <strong>Apps</strong> tab to choose which apps to hide.
      </div>
```

- [ ] **Step 15: Replace the "resetting" progress message (line 249)**

Find:
```svelte
        <p>Resetting phone — do not unplug…</p>
```
Replace with:
```svelte
        <p>Removing Accountability Mode — do not unplug…</p>
```

- [ ] **Step 16: Verify visually**

Run the Wails dev server and step through the wizard:
```bash
cd /home/logan/projects/sober/desktop && wails dev
```
Confirm: no occurrences of "SoberAdmin", "Device Owner", "ADB", or "adb" appear in any visible UI text.

- [ ] **Step 17: Commit**

```bash
cd /home/logan/projects/sober
git add desktop/frontend/src/components/SetupTab.svelte
git commit -m "feat: replace technical jargon with plain-English copy in SetupTab"
```

---

### Task 4: Trust transparency section

**Files:**
- Modify: `desktop/frontend/src/components/SetupTab.svelte`

- [ ] **Step 18: Replace the `detect` step to show trust info when disconnected**

Find the entire `{#if wizardStep === 'detect'}` block (lines 297–301):
```svelte
    {#if wizardStep === 'detect'}
      <div class="progress">
        <div class="spinner"></div>
        <p>{connected ? 'Checking phone…' : 'Waiting for phone connection…'}</p>
      </div>
```
Replace with:
```svelte
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
              <p class="hint">Changes require your phone physically connected via USB cable — making it harder to undo settings on impulse.</p>
            </div>
          </details>
        </div>
      {/if}
```

- [ ] **Step 19: Add trust section styles**

At the end of the `<style>` block (before the closing `</style>`), add:

```css
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
  .trust-body {
    padding: 12px 16px;
    background: #12121e;
    font-size: 13px;
    color: #9ca3af;
    line-height: 1.7;
  }
  .trust-body p { margin: 8px 0 4px; color: #e2e2e8; }
  .trust-body ul { margin: 0 0 8px; padding-left: 20px; }
  .trust-body li { margin: 3px 0; }
```

- [ ] **Step 20: Verify trust section**

In dev mode, visit the Setup tab with no phone connected — confirm the "Plug your phone in" message and expandable "How does this work?" section appear. Plug in the phone — confirm it transitions to the spinner automatically.

- [ ] **Step 21: Commit**

```bash
cd /home/logan/projects/sober
git add desktop/frontend/src/components/SetupTab.svelte
git commit -m "feat: add trust transparency section to setup wizard idle state"
```

---

### Task 5: Error message mapper

**Files:**
- Modify: `desktop/frontend/src/components/SetupTab.svelte`

- [ ] **Step 22: Add `friendlyError` function to the script block**

In the `<script>` section of `SetupTab.svelte`, after the `leaveUnrestricted` function (around line 200), add:

```typescript
  function friendlyError(msg: string): string {
    if (!msg) return 'An unknown error occurred.'
    if (
      msg.includes('no devices') ||
      msg.includes('device not found') ||
      msg.includes('no devices/emulators found')
    ) return 'No phone detected. Make sure your USB cable is connected and your phone is unlocked.'
    if (msg.includes('device unauthorized') || msg.includes('unauthorized'))
      return "Your phone is asking for permission. Check your phone screen and tap 'Allow'."
    if (msg.includes('Accountability Mode is already active'))
      return 'Accountability Mode is already active on this phone.'
    if (msg.includes('Another app is controlling'))
      return 'Another app is controlling this phone. It must be removed before Sober can be set up.'
    if (
      msg.includes('Google accounts are still on this device') ||
      msg.includes('there are already some accounts')
    ) return 'Please sign out of your Google account first. Tap "Open Account Settings" for instructions.'
    if (msg.includes('timed out') || msg.includes('not removed within'))
      return 'This is taking longer than expected. Make sure your phone is unlocked and try again.'
    if (msg.includes('set-device-owner failed'))
      return 'Setup failed — make sure your phone is unlocked and try again.'
    return msg
  }
```

- [ ] **Step 23: Use `friendlyError` in both error display sites**

**Error banner in wizard** (line 386–389):

Find:
```svelte
    {:else if wizardStep === 'error'}
      <div class="banner error">
        <strong>Setup failed:</strong> {errorMessage}
        <button on:click={retryFromStart}>Try Again</button>
      </div>
```
Replace with:
```svelte
    {:else if wizardStep === 'error'}
      <div class="banner error">
        {friendlyError(errorMessage)}
        <button on:click={retryFromStart}>Try Again</button>
      </div>
```

**Reset error banner** (line 218):

Find:
```svelte
          <div class="banner error" style="margin-bottom: 12px">{resetError}</div>
```
Replace with:
```svelte
          <div class="banner error" style="margin-bottom: 12px">{friendlyError(resetError)}</div>
```

- [ ] **Step 24: Verify error messages**

With a phone connected but USB debugging NOT authorized, attempt setup — confirm "Your phone is asking for permission" message appears instead of a raw ADB error string.

- [ ] **Step 25: Final commit**

```bash
cd /home/logan/projects/sober
git add desktop/frontend/src/components/SetupTab.svelte
git commit -m "feat: map known error conditions to friendly messages in SetupTab"
```

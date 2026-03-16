# Optimized Setup & Reset Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the manual setup instructions with a guided wizard (account detection, optional contacts backup, live account-removal polling, automated Device Owner install) and add a "Reset Everything" flow to fully restore the phone.

**Architecture:** The Android APK gains three new broadcasts (EXPORT_CONTACTS, IMPORT_CONTACTS, CLEAR_DEVICE_OWNER). The Go backend gains an `AppManager` interface, a `config` package for persistence, and new `Commands` methods. The Svelte frontend's `SetupTab` is rewritten as a step-driven wizard.

**Tech Stack:** Kotlin/Android (API 26+, JUnit+Robolectric tests), Go 1.21 (table-driven tests with fake runner), Svelte + TypeScript (Wails bindings)

**Spec:** `docs/superpowers/specs/2026-03-15-optimized-setup-reset-design.md`

---

## Chunk 1: Android APK — Contacts broadcasts + CLEAR_DEVICE_OWNER

### Task 1: ContactsManager — export

**Files:**
- Create: `android/app/src/main/java/com/sober/admin/ContactsManager.kt`
- Create: `android/app/src/test/java/com/sober/admin/ContactsManagerTest.kt`

- [ ] **Step 1: Write the failing test**

`android/app/src/test/java/com/sober/admin/ContactsManagerTest.kt`:
```kotlin
package com.sober.admin

import android.content.ContentProviderOperation
import android.provider.ContactsContract
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import java.io.File

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [28])
class ContactsManagerTest {

    private lateinit var context: android.content.Context
    private lateinit var manager: ContactsManager

    @Before
    fun setUp() {
        context = ApplicationProvider.getApplicationContext()
        manager = ContactsManager(context)
    }

    @Test
    fun `exportToVcf writes empty string when no contacts`() {
        val file = File(context.cacheDir, "test_export.vcf")
        manager.exportToVcf(file)
        assertEquals("", file.readText())
    }

    @Test
    fun `exportToVcf includes contact name`() {
        insertContact(context, "Alice Smith", listOf("+15550001111"), emptyList())
        val file = File(context.cacheDir, "test_export2.vcf")
        manager.exportToVcf(file)
        val vcf = file.readText()
        assertTrue("Expected FN:Alice Smith in VCF", vcf.contains("FN:Alice Smith"))
        assertTrue("Expected BEGIN:VCARD", vcf.contains("BEGIN:VCARD"))
        assertTrue("Expected END:VCARD", vcf.contains("END:VCARD"))
    }

    @Test
    fun `exportToVcf includes phone number`() {
        insertContact(context, "Bob Jones", listOf("+15550002222"), emptyList())
        val file = File(context.cacheDir, "test_export3.vcf")
        manager.exportToVcf(file)
        assertTrue(file.readText().contains("+15550002222"))
    }

    @Test
    fun `exportToVcf includes email`() {
        insertContact(context, "Carol", emptyList(), listOf("carol@example.com"))
        val file = File(context.cacheDir, "test_export4.vcf")
        manager.exportToVcf(file)
        assertTrue(file.readText().contains("carol@example.com"))
    }

    private fun insertContact(
        ctx: android.content.Context,
        name: String,
        phones: List<String>,
        emails: List<String>
    ) {
        val ops = ArrayList<ContentProviderOperation>()
        ops.add(
            ContentProviderOperation.newInsert(ContactsContract.RawContacts.CONTENT_URI)
                .withValue(ContactsContract.RawContacts.ACCOUNT_TYPE, null)
                .withValue(ContactsContract.RawContacts.ACCOUNT_NAME, null)
                .build()
        )
        ops.add(
            ContentProviderOperation.newInsert(ContactsContract.Data.CONTENT_URI)
                .withValueBackReference(ContactsContract.Data.RAW_CONTACT_ID, 0)
                .withValue(ContactsContract.Data.MIMETYPE, ContactsContract.CommonDataKinds.StructuredName.CONTENT_ITEM_TYPE)
                .withValue(ContactsContract.CommonDataKinds.StructuredName.DISPLAY_NAME, name)
                .build()
        )
        for (phone in phones) {
            ops.add(
                ContentProviderOperation.newInsert(ContactsContract.Data.CONTENT_URI)
                    .withValueBackReference(ContactsContract.Data.RAW_CONTACT_ID, 0)
                    .withValue(ContactsContract.Data.MIMETYPE, ContactsContract.CommonDataKinds.Phone.CONTENT_ITEM_TYPE)
                    .withValue(ContactsContract.CommonDataKinds.Phone.NUMBER, phone)
                    .build()
            )
        }
        for (email in emails) {
            ops.add(
                ContentProviderOperation.newInsert(ContactsContract.Data.CONTENT_URI)
                    .withValueBackReference(ContactsContract.Data.RAW_CONTACT_ID, 0)
                    .withValue(ContactsContract.Data.MIMETYPE, ContactsContract.CommonDataKinds.Email.CONTENT_ITEM_TYPE)
                    .withValue(ContactsContract.CommonDataKinds.Email.ADDRESS, email)
                    .build()
            )
        }
        ctx.contentResolver.applyBatch(ContactsContract.AUTHORITY, ops)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /home/logan/projects/sober/android
./gradlew test --tests "com.sober.admin.ContactsManagerTest" 2>&1 | tail -20
```
Expected: FAIL — `ContactsManager` class not found.

- [ ] **Step 3: Create ContactsManager.kt with exportToVcf**

`android/app/src/main/java/com/sober/admin/ContactsManager.kt`:
```kotlin
package com.sober.admin

import android.content.ContentProviderOperation
import android.content.Context
import android.provider.ContactsContract
import java.io.File

class ContactsManager(private val context: Context) {

    /** Exports all contacts to a VCF 3.0 file. Writes an empty file if no contacts. */
    fun exportToVcf(outputFile: File) {
        val sb = StringBuilder()
        val contactsCursor = context.contentResolver.query(
            ContactsContract.Contacts.CONTENT_URI,
            arrayOf(ContactsContract.Contacts._ID, ContactsContract.Contacts.DISPLAY_NAME_PRIMARY),
            null, null, null
        )
        contactsCursor?.use { c ->
            while (c.moveToNext()) {
                val id = c.getString(c.getColumnIndexOrThrow(ContactsContract.Contacts._ID))
                val name = c.getString(c.getColumnIndexOrThrow(ContactsContract.Contacts.DISPLAY_NAME_PRIMARY)) ?: ""
                sb.append("BEGIN:VCARD\r\nVERSION:3.0\r\n")
                sb.append("FN:${escapeVcf(name)}\r\n")

                val phoneCursor = context.contentResolver.query(
                    ContactsContract.CommonDataKinds.Phone.CONTENT_URI,
                    arrayOf(ContactsContract.CommonDataKinds.Phone.NUMBER),
                    "${ContactsContract.CommonDataKinds.Phone.CONTACT_ID} = ?",
                    arrayOf(id), null
                )
                phoneCursor?.use { pc ->
                    while (pc.moveToNext()) {
                        val num = pc.getString(pc.getColumnIndexOrThrow(ContactsContract.CommonDataKinds.Phone.NUMBER))
                        if (!num.isNullOrBlank()) sb.append("TEL:${escapeVcf(num)}\r\n")
                    }
                }

                val emailCursor = context.contentResolver.query(
                    ContactsContract.CommonDataKinds.Email.CONTENT_URI,
                    arrayOf(ContactsContract.CommonDataKinds.Email.ADDRESS),
                    "${ContactsContract.CommonDataKinds.Email.CONTACT_ID} = ?",
                    arrayOf(id), null
                )
                emailCursor?.use { ec ->
                    while (ec.moveToNext()) {
                        val addr = ec.getString(ec.getColumnIndexOrThrow(ContactsContract.CommonDataKinds.Email.ADDRESS))
                        if (!addr.isNullOrBlank()) sb.append("EMAIL:${escapeVcf(addr)}\r\n")
                    }
                }

                sb.append("END:VCARD\r\n")
            }
        }
        outputFile.writeText(sb.toString())
    }

    /** Imports contacts from a VCF 3.0 file. Returns the number of contacts imported. */
    fun importFromVcf(vcfFile: File): Int {
        val content = vcfFile.readText()
        val cards = content.split("END:VCARD").filter { it.contains("BEGIN:VCARD") }
        var count = 0
        for (card in cards) {
            val name = card.lines()
                .firstOrNull { it.startsWith("FN:") }
                ?.removePrefix("FN:")?.trim()
                ?.let { unescapeVcf(it) }
                ?: continue

            val phones = card.lines()
                .filter { line -> line.startsWith("TEL:") || (line.startsWith("TEL;") && line.contains(":")) }
                .mapNotNull { line ->
                    val colon = line.lastIndexOf(':')
                    if (colon >= 0) unescapeVcf(line.substring(colon + 1).trim()) else null
                }
                .filter { it.isNotBlank() }

            val emails = card.lines()
                .filter { it.startsWith("EMAIL:") }
                .map { unescapeVcf(it.removePrefix("EMAIL:").trim()) }
                .filter { it.isNotBlank() }

            val ops = ArrayList<ContentProviderOperation>()
            ops.add(
                ContentProviderOperation.newInsert(ContactsContract.RawContacts.CONTENT_URI)
                    .withValue(ContactsContract.RawContacts.ACCOUNT_TYPE, null)
                    .withValue(ContactsContract.RawContacts.ACCOUNT_NAME, null)
                    .build()
            )
            ops.add(
                ContentProviderOperation.newInsert(ContactsContract.Data.CONTENT_URI)
                    .withValueBackReference(ContactsContract.Data.RAW_CONTACT_ID, 0)
                    .withValue(ContactsContract.Data.MIMETYPE, ContactsContract.CommonDataKinds.StructuredName.CONTENT_ITEM_TYPE)
                    .withValue(ContactsContract.CommonDataKinds.StructuredName.DISPLAY_NAME, name)
                    .build()
            )
            for (phone in phones) {
                ops.add(
                    ContentProviderOperation.newInsert(ContactsContract.Data.CONTENT_URI)
                        .withValueBackReference(ContactsContract.Data.RAW_CONTACT_ID, 0)
                        .withValue(ContactsContract.Data.MIMETYPE, ContactsContract.CommonDataKinds.Phone.CONTENT_ITEM_TYPE)
                        .withValue(ContactsContract.CommonDataKinds.Phone.NUMBER, phone)
                        .build()
                )
            }
            for (email in emails) {
                ops.add(
                    ContentProviderOperation.newInsert(ContactsContract.Data.CONTENT_URI)
                        .withValueBackReference(ContactsContract.Data.RAW_CONTACT_ID, 0)
                        .withValue(ContactsContract.Data.MIMETYPE, ContactsContract.CommonDataKinds.Email.CONTENT_ITEM_TYPE)
                        .withValue(ContactsContract.CommonDataKinds.Email.ADDRESS, email)
                        .build()
                )
            }
            try {
                context.contentResolver.applyBatch(ContactsContract.AUTHORITY, ops)
                count++
            } catch (e: Exception) {
                // Skip this contact and continue
            }
        }
        return count
    }

    private fun escapeVcf(s: String): String = s.replace("\\", "\\\\").replace(",", "\\,").replace("\n", "\\n")
    private fun unescapeVcf(s: String): String = s.replace("\\n", "\n").replace("\\,", ",").replace("\\\\", "\\")
}
```

- [ ] **Step 4: Add import tests to ContactsManagerTest.kt**

Add to `ContactsManagerTest.kt` inside the class:

```kotlin
@Test
fun `importFromVcf round-trip restores contact`() {
    insertContact(context, "Dave Export", listOf("+15550003333"), listOf("dave@example.com"))
    val exportFile = File(context.cacheDir, "roundtrip.vcf")
    manager.exportToVcf(exportFile)

    // Wipe all contacts
    context.contentResolver.delete(ContactsContract.RawContacts.CONTENT_URI, null, null)

    val count = manager.importFromVcf(exportFile)
    assertTrue("Expected at least 1 import, got $count", count >= 1)

    val cursor = context.contentResolver.query(
        ContactsContract.Contacts.CONTENT_URI,
        arrayOf(ContactsContract.Contacts.DISPLAY_NAME_PRIMARY),
        null, null, null
    )
    var found = false
    cursor?.use { while (it.moveToNext()) { if (it.getString(0) == "Dave Export") found = true } }
    assertTrue("Expected Dave Export to be restored", found)
}

@Test
fun `importFromVcf returns 0 for empty file`() {
    val empty = File(context.cacheDir, "empty.vcf")
    empty.writeText("")
    val count = manager.importFromVcf(empty)
    assertEquals(0, count)
}

@Test
fun `importFromVcf skips cards with no FN`() {
    val vcf = "BEGIN:VCARD\r\nVERSION:3.0\r\nTEL:+15550004444\r\nEND:VCARD\r\n"
    val file = File(context.cacheDir, "nofn.vcf")
    file.writeText(vcf)
    val count = manager.importFromVcf(file)
    assertEquals(0, count)
}

@Test
fun `importFromVcf does not match non-phone typed lines`() {
    // ADR and PHOTO lines must not be treated as phone numbers
    val vcf = "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Eve\r\n" +
              "ADR;TYPE=HOME:;;123 Main St;City;ST;12345;US\r\n" +
              "TEL:+15550005555\r\n" +
              "END:VCARD\r\n"
    val file = File(context.cacheDir, "nonphone.vcf")
    file.writeText(vcf)
    manager.importFromVcf(file)

    val phoneCursor = context.contentResolver.query(
        ContactsContract.CommonDataKinds.Phone.CONTENT_URI,
        arrayOf(ContactsContract.CommonDataKinds.Phone.NUMBER),
        null, null, null
    )
    var count = 0
    phoneCursor?.use { while (it.moveToNext()) count++ }
    assertEquals("Expected exactly 1 phone number (not the ADR line)", 1, count)
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /home/logan/projects/sober/android
./gradlew test --tests "com.sober.admin.ContactsManagerTest" 2>&1 | tail -20
```
Expected: BUILD SUCCESSFUL, all 8 tests pass.

- [ ] **Step 6: Commit**

```bash
git add android/app/src/main/java/com/sober/admin/ContactsManager.kt \
        android/app/src/test/java/com/sober/admin/ContactsManagerTest.kt
git commit -m "feat(android): add ContactsManager for VCF export/import"
```

---

### Task 2: Add 3 new broadcasts to CommandReceiver + update manifest

**Files:**
- Modify: `android/app/src/main/java/com/sober/admin/CommandReceiver.kt`
- Modify: `android/app/src/main/AndroidManifest.xml`

- [ ] **Step 1: Add the 3 new `when` branches to CommandReceiver.kt**

Add after the existing `"com.sober.APPLY_RESTRICTIONS"` branch:

```kotlin
"com.sober.EXPORT_CONTACTS" -> {
    val result = goAsync()
    Thread {
        val outFile = File(context.cacheDir, "sober_contacts.vcf")
        try {
            val manager = ContactsManager(context)
            manager.exportToVcf(outFile)
        } catch (e: Exception) {
            outFile.writeText("""{"error":${escapeJson(e.toString())}}""")
        } finally {
            result.finish()
        }
    }.start()
}
"com.sober.IMPORT_CONTACTS" -> {
    val result = goAsync()
    Thread {
        val resultFile = File(context.cacheDir, "sober_import_result.json")
        try {
            // Use the app's external files directory — no storage permission required,
            // accessible to both ADB push and this app on all API levels (26+).
            val vcfFile = File(context.getExternalFilesDir(null), "sober_contacts_restore.vcf")
            if (!vcfFile.exists()) throw Exception("source file not found: ${vcfFile.absolutePath}")
            val manager = ContactsManager(context)
            val count = manager.importFromVcf(vcfFile)
            vcfFile.delete()
            resultFile.writeText("""{"success":true,"count":$count}""")
        } catch (e: Exception) {
            resultFile.writeText("""{"error":${escapeJson(e.toString())}}""")
        } finally {
            result.finish()
        }
    }.start()
}
"com.sober.CLEAR_DEVICE_OWNER" -> {
    // goAsync() is required — without it the process is eligible for reclamation
    // before clearDeviceOwnerApp() executes on Android 8+ (API 26, this app's minSdk).
    // No result file is written: the desktop confirms success by polling
    // `dpm list-owners` until com.sober.admin no longer appears (10s deadline).
    val result = goAsync()
    Thread {
        try {
            dpm.clearDeviceOwnerApp(context.packageName)
        } finally {
            result.finish()
        }
    }.start()
}
```

- [ ] **Step 2: Update AndroidManifest.xml**

Add `READ_CONTACTS` and `WRITE_CONTACTS` permissions and the 3 new intent-filter actions.

Add these two lines after `QUERY_ALL_PACKAGES`:
```xml
<uses-permission android:name="android.permission.READ_CONTACTS"/>
<uses-permission android:name="android.permission.WRITE_CONTACTS"/>
```

Note: no `READ_EXTERNAL_STORAGE` is needed. IMPORT_CONTACTS uses `context.getExternalFilesDir(null)` which is the app's own external files directory — apps can read their own external files directory without any permission on all API levels.

Add these 3 actions inside the CommandReceiver's `<intent-filter>` (after `com.sober.APPLY_RESTRICTIONS`):
```xml
<action android:name="com.sober.EXPORT_CONTACTS"/>
<action android:name="com.sober.IMPORT_CONTACTS"/>
<action android:name="com.sober.CLEAR_DEVICE_OWNER"/>
```

- [ ] **Step 3: Run all Android tests to confirm nothing broken**

```bash
cd /home/logan/projects/sober/android
./gradlew test 2>&1 | tail -20
```
Expected: BUILD SUCCESSFUL.

- [ ] **Step 4: Commit**

```bash
git add android/app/src/main/java/com/sober/admin/CommandReceiver.kt \
        android/app/src/main/AndroidManifest.xml
git commit -m "feat(android): add EXPORT_CONTACTS, IMPORT_CONTACTS, CLEAR_DEVICE_OWNER broadcasts"
```

---

### Task 3: Build APK, bump version, copy to desktop

**Files:**
- Modify: `android/app/build.gradle` (versionCode 1 → 2)
- Modify: `desktop/embed.go` (BundledAdminVersion 1 → 2)
- Update: `desktop/assets/sober-admin.apk`

- [ ] **Step 1: Bump versionCode in build.gradle and guard debuggable**

In `android/app/build.gradle`, change:
```
versionCode 1
```
to:
```
versionCode 2
```

Also add a comment to the release build type so future maintainers don't remove `debuggable true`:
```gradle
buildTypes {
    release {
        // debuggable must remain true — the run-as contacts backup mechanism
        // (EXPORT_CONTACTS, IMPORT_CONTACTS) requires adb run-as access,
        // which only works on debuggable builds. Do not remove.
        debuggable true
        minifyEnabled true
        proguardFiles getDefaultProguardFile('proguard-android-optimize.txt')
        signingConfig signingConfigs.debug
    }
}
```

- [ ] **Step 2: Build release APK**

```bash
cd /home/logan/projects/sober/android
./gradlew assembleRelease 2>&1 | tail -10
```
Expected: `BUILD SUCCESSFUL`. APK at `app/build/outputs/apk/release/app-release.apk`.

- [ ] **Step 3: Copy APK to desktop assets**

```bash
cp /home/logan/projects/sober/android/app/build/outputs/apk/release/app-release.apk \
   /home/logan/projects/sober/desktop/assets/sober-admin.apk
```

- [ ] **Step 4: Bump BundledAdminVersion in embed.go**

In `desktop/embed.go`, change:
```go
const BundledAdminVersion = 1
```
to:
```go
const BundledAdminVersion = 2
```

- [ ] **Step 5: Verify desktop still compiles**

```bash
cd /home/logan/projects/sober/desktop
go build ./... 2>&1
```
Expected: no output (success).

- [ ] **Step 6: Commit**

```bash
git add android/app/build.gradle \
        desktop/embed.go \
        desktop/assets/sober-admin.apk
git commit -m "chore: bump SoberAdmin to versionCode 2 with new broadcasts"
```

---

## Chunk 2: Go backend — AppManager interface, Config, new Commands, new app.go methods

### Task 4: AppManager interface

**Files:**
- Create: `desktop/adb/app_manager.go`
- Modify: `desktop/app.go` (add `appManager` field, route 4 methods through it)

- [ ] **Step 1: Create app_manager.go**

`desktop/adb/app_manager.go`:
```go
package adb

// AppManager abstracts hide/show/list/uninstall operations so the UI layer
// is independent of whether Device Owner or direct-ADB mode is active.
// *Commands implements this interface — no adapter needed.
type AppManager interface {
	ListApps() ([]App, error)
	HideApp(pkg string) error
	ShowApp(pkg string) error
	UninstallApp(pkg string) error
}
```

- [ ] **Step 2: Verify *Commands satisfies the interface**

Add this to `desktop/adb/app_manager.go` (keep it permanently — this is idiomatic Go for documenting interface compliance):
```go
var _ AppManager = (*Commands)(nil)
```

```bash
cd /home/logan/projects/sober/desktop
go build ./adb/... 2>&1
```
Expected: no output (compilation failure here means Commands is missing a method).

- [ ] **Step 3: Add `appManager` field to App struct and wire it in startup**

In `desktop/app.go`, update the `App` struct — add the field after `commands`:
```go
type App struct {
    ctx        context.Context
    runner     *adb.Runner
    commands   *adb.Commands  // setup/teardown operations only
    appManager adb.AppManager // hide/show/list/uninstall (mode-agnostic)
    poller     *adb.Poller

    connected bool
    serial    string
}
```

In `startup()`, add after `a.commands = adb.NewCommands(runner)`:
```go
a.appManager = a.commands
```

- [ ] **Step 4: Route GetApps, HideApp, ShowApp, UninstallApp through appManager**

In `desktop/app.go`, change the 4 methods:

```go
func (a *App) GetApps() ([]adb.App, error) {
    if !a.connected {
        return nil, fmt.Errorf("no phone connected")
    }
    return a.appManager.ListApps()
}

func (a *App) HideApp(pkg string) error {
    if !a.connected {
        return fmt.Errorf("no phone connected")
    }
    return a.appManager.HideApp(pkg)
}

func (a *App) ShowApp(pkg string) error {
    if !a.connected {
        return fmt.Errorf("no phone connected")
    }
    return a.appManager.ShowApp(pkg)
}

func (a *App) UninstallApp(pkg string) error {
    if !a.connected {
        return fmt.Errorf("no phone connected")
    }
    return a.appManager.UninstallApp(pkg)
}
```

- [ ] **Step 5: Verify it compiles and existing tests pass**

```bash
cd /home/logan/projects/sober/desktop
go build ./... 2>&1
go test ./adb/... -v 2>&1 | tail -20
```
Expected: BUILD SUCCESS, all existing tests pass.

- [ ] **Step 6: Commit**

```bash
git add desktop/adb/app_manager.go desktop/app.go
git commit -m "refactor: introduce AppManager interface, route app ops through it"
```

---

### Task 5: Config persistence

**Files:**
- Create: `desktop/config/config.go`
- Create: `desktop/config/config_test.go`

- [ ] **Step 1: Write the failing tests**

`desktop/config/config_test.go`:
```go
package config_test

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/sober/desktop/config"
)

// isolateConfig redirects os.UserConfigDir() to a temp dir for all platforms.
// On Linux: sets XDG_CONFIG_HOME. On Windows: sets APPDATA. On macOS: sets HOME.
func isolateConfig(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    t.Setenv("HOME", dir)
    t.Setenv("XDG_CONFIG_HOME", dir) // Linux: takes precedence over $HOME/.config
    t.Setenv("APPDATA", dir)         // Windows: used by os.UserConfigDir()
    return dir
}

func TestLoadMissing(t *testing.T) {
    isolateConfig(t)
    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("Load should not error on missing file, got: %v", err)
    }
    if cfg.SetupMode != "device_owner" {
        t.Errorf("expected default setup_mode=device_owner, got: %s", cfg.SetupMode)
    }
    if cfg.ContactsBackupPath != "" {
        t.Errorf("expected empty contacts_backup_path, got: %s", cfg.ContactsBackupPath)
    }
}

func TestSaveAndLoad(t *testing.T) {
    isolateConfig(t)
    cfg := &config.Config{
        SetupMode:          "device_owner",
        ContactsBackupPath: "/tmp/contacts-backup-20260315-143022.vcf",
    }
    if err := config.Save(cfg); err != nil {
        t.Fatalf("Save error: %v", err)
    }
    loaded, err := config.Load()
    if err != nil {
        t.Fatalf("Load error: %v", err)
    }
    if loaded.SetupMode != cfg.SetupMode {
        t.Errorf("setup_mode: want %q, got %q", cfg.SetupMode, loaded.SetupMode)
    }
    if loaded.ContactsBackupPath != cfg.ContactsBackupPath {
        t.Errorf("contacts_backup_path: want %q, got %q", cfg.ContactsBackupPath, loaded.ContactsBackupPath)
    }
}

func TestLoadCorrupted(t *testing.T) {
    dir := isolateConfig(t)
    // Write a corrupted config file at the path os.UserConfigDir() will return.
    // With XDG_CONFIG_HOME=dir, os.UserConfigDir() returns dir on Linux.
    cfgDir := filepath.Join(dir, "sober")
    os.MkdirAll(cfgDir, 0700)
    os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte("not-json{{"), 0600)

    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("Load should not error on corrupted file, got: %v", err)
    }
    if cfg.SetupMode != "device_owner" {
        t.Errorf("expected default on corrupt, got: %s", cfg.SetupMode)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /home/logan/projects/sober/desktop
go test ./config/... 2>&1 | tail -10
```
Expected: FAIL — package not found.

- [ ] **Step 3: Create config.go**

`desktop/config/config.go`:
```go
package config

import (
    "encoding/json"
    "os"
    "path/filepath"
)

// Config persists state across app launches.
type Config struct {
    SetupMode          string `json:"setup_mode"`           // "device_owner" | "direct_adb"
    ContactsBackupPath string `json:"contacts_backup_path"` // absolute path or ""
}

func configPath() (string, error) {
    dir, err := os.UserConfigDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "sober", "config.json"), nil
}

// Load reads the config file. Returns sensible defaults if the file is missing or unreadable.
func Load() (*Config, error) {
    path, err := configPath()
    if err != nil {
        return defaultConfig(), nil
    }
    data, err := os.ReadFile(path)
    if err != nil {
        return defaultConfig(), nil
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return defaultConfig(), nil
    }
    return &cfg, nil
}

// Save writes the config file, creating directories as needed.
func Save(cfg *Config) error {
    path, err := configPath()
    if err != nil {
        return err
    }
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
        return err
    }
    data, err := json.Marshal(cfg)
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0600)
}

func defaultConfig() *Config {
    return &Config{SetupMode: "device_owner"}
}
```

- [ ] **Step 4: Add module path to go.mod**

Confirm the module path in `desktop/go.mod` — it should be `github.com/sober/desktop`. If so, the import `github.com/sober/desktop/config` is correct already.

```bash
head -3 /home/logan/projects/sober/desktop/go.mod
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /home/logan/projects/sober/desktop
go test ./config/... -v 2>&1
```
Expected: all 3 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add desktop/config/config.go desktop/config/config_test.go
git commit -m "feat: add config persistence for setup mode and contacts backup path"
```

---

### Task 6: New Commands methods

**Files:**
- Modify: `desktop/adb/commands.go`
- Modify: `desktop/adb/commands_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `desktop/adb/commands_test.go`:

```go
func TestCountGoogleAccounts(t *testing.T) {
    t.Run("returns count when accounts present", func(t *testing.T) {
        fake := &fakeRunner{output: "Account {name=a@gmail.com, type=com.google}\nAccount {name=b@gmail.com, type=com.google}\n"}
        c := adb.NewCommands(fake)
        n, err := c.CountGoogleAccounts()
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if n != 2 {
            t.Errorf("expected 2, got %d", n)
        }
    })

    t.Run("returns 0 when no accounts", func(t *testing.T) {
        fake := &fakeRunner{output: "Accounts: 0\n"}
        c := adb.NewCommands(fake)
        n, err := c.CountGoogleAccounts()
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if n != 0 {
            t.Errorf("expected 0, got %d", n)
        }
    })

    t.Run("returns 0 on runner error (fail open)", func(t *testing.T) {
        fake := &fakeRunner{err: fmt.Errorf("adb fail")}
        c := adb.NewCommands(fake)
        n, err := c.CountGoogleAccounts()
        if err != nil {
            t.Fatalf("fail-open: expected nil, got %v", err)
        }
        if n != 0 {
            t.Errorf("expected 0 on error, got %d", n)
        }
    })
}

func TestOpenAccountSettings(t *testing.T) {
    fake := &fakeRunner{}
    c := adb.NewCommands(fake)
    err := c.OpenAccountSettings()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    call := strings.Join(fake.calls[0], " ")
    if !strings.Contains(call, "android.settings.SYNC_SETTINGS") {
        t.Errorf("expected SYNC_SETTINGS intent, got: %s", call)
    }
}

func TestExportContacts(t *testing.T) {
    vcf := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Test\r\nEND:VCARD\r\n"

    t.Run("success with contacts", func(t *testing.T) {
        fake := &callTrackingRunner{
            responses: map[string]string{
                "broadcast": "",
                "cat":       vcf,
                "rm":        "",
            },
        }
        c := adb.NewCommands(fake)
        result, err := c.ExportContacts()
        if err != nil {
            t.Fatalf("ExportContacts error: %v", err)
        }
        if !strings.Contains(result, "BEGIN:VCARD") {
            t.Errorf("expected VCF content, got: %s", result)
        }
    })

    t.Run("success with no contacts (empty file)", func(t *testing.T) {
        fake := &callTrackingRunner{
            responses: map[string]string{
                "broadcast": "",
                "cat":       "",
                "rm":        "",
            },
        }
        c := adb.NewCommands(fake)
        result, err := c.ExportContacts()
        if err != nil {
            t.Fatalf("ExportContacts error: %v", err)
        }
        if result != "" {
            t.Errorf("expected empty string for no contacts, got: %s", result)
        }
    })

    t.Run("broadcast error", func(t *testing.T) {
        fake := &fakeRunner{err: fmt.Errorf("device not found")}
        c := adb.NewCommands(fake)
        _, err := c.ExportContacts()
        if err == nil {
            t.Fatal("expected error, got nil")
        }
    })

    t.Run("device error response", func(t *testing.T) {
        fake := &callTrackingRunner{
            responses: map[string]string{
                "broadcast": "",
                "cat":       `{"error":"permission denied"}`,
                "rm":        "",
            },
        }
        c := adb.NewCommands(fake)
        _, err := c.ExportContacts()
        if err == nil {
            t.Fatal("expected error from device, got nil")
        }
        if !strings.Contains(err.Error(), "EXPORT_CONTACTS failed") {
            t.Errorf("unexpected error: %v", err)
        }
    })
}

func TestImportContacts(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        fake := &callTrackingRunner{
            responses: map[string]string{
                "push":      "",
                "broadcast": "",
                "cat":       `{"success":true,"count":3}`,
                "rm":        "",
            },
        }
        c := adb.NewCommands(fake)
        err := c.ImportContacts("/tmp/backup.vcf")
        if err != nil {
            t.Fatalf("ImportContacts error: %v", err)
        }
    })

    t.Run("push error", func(t *testing.T) {
        fake := &fakeRunner{err: fmt.Errorf("push failed")}
        c := adb.NewCommands(fake)
        err := c.ImportContacts("/tmp/backup.vcf")
        if err == nil {
            t.Fatal("expected error on push failure")
        }
        if !strings.Contains(err.Error(), "push contacts") {
            t.Errorf("unexpected error: %v", err)
        }
    })

    t.Run("device error response", func(t *testing.T) {
        fake := &callTrackingRunner{
            responses: map[string]string{
                "push":      "",
                "broadcast": "",
                "cat":       `{"error":"file not found"}`,
                "rm":        "",
            },
        }
        c := adb.NewCommands(fake)
        err := c.ImportContacts("/tmp/backup.vcf")
        if err == nil {
            t.Fatal("expected error from device")
        }
        if !strings.Contains(err.Error(), "IMPORT_CONTACTS failed") {
            t.Errorf("unexpected error: %v", err)
        }
    })
}

func TestClearDeviceOwner(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        // After the broadcast, dpm list-owners returns no sober.admin
        fake := &callTrackingRunner{
            responses: map[string]string{
                "broadcast":   "",
                "list-owners": "{}",  // no device owner
            },
        }
        c := adb.NewCommands(fake)
        err := c.ClearDeviceOwner()
        if err != nil {
            t.Fatalf("ClearDeviceOwner error: %v", err)
        }
    })

    t.Run("broadcast error", func(t *testing.T) {
        fake := &fakeRunner{err: fmt.Errorf("device offline")}
        c := adb.NewCommands(fake)
        err := c.ClearDeviceOwner()
        if err == nil {
            t.Fatal("expected error, got nil")
        }
    })
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /home/logan/projects/sober/desktop
go test ./adb/... 2>&1 | tail -15
```
Expected: FAIL — methods not defined.

- [ ] **Step 3: Add the 5 new methods to commands.go**

Add to `desktop/adb/commands.go`:

```go
// CountGoogleAccounts returns the number of Google accounts on the device.
// Returns 0 on runner error (fail open — SetDeviceOwner will catch any remaining accounts).
func (c *Commands) CountGoogleAccounts() (int, error) {
	out, err := c.runner.Run("shell", "dumpsys", "account")
	if err != nil {
		return 0, nil
	}
	count := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Account {") && strings.Contains(line, "type=com.google") {
			count++
		}
	}
	return count, nil
}

// OpenAccountSettings opens the Android Accounts settings screen on the phone.
func (c *Commands) OpenAccountSettings() error {
	_, err := c.runner.Run("shell", "am", "start", "-a", "android.settings.SYNC_SETTINGS")
	return err
}

// ExportContacts exports all contacts from the phone as a VCF string.
// Returns an empty string (no error) if the device has no contacts.
func (c *Commands) ExportContacts() (string, error) {
	_, _ = c.runner.Run("shell", "run-as", "com.sober.admin", "rm", "-f", "cache/sober_contacts.vcf")

	_, err := c.runner.Run(
		"shell", "am", "broadcast",
		"-a", "com.sober.EXPORT_CONTACTS",
		"-n", "com.sober.admin/.CommandReceiver",
	)
	if err != nil {
		return "", fmt.Errorf("EXPORT_CONTACTS broadcast: %w", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		out, err := c.runner.Run("shell", "run-as", "com.sober.admin", "cat", "cache/sober_contacts.vcf")
		if err != nil {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		out = strings.TrimSpace(out)
		if strings.HasPrefix(out, `{"error"`) {
			return "", fmt.Errorf("EXPORT_CONTACTS failed on device: %s", out)
		}
		// File exists (either empty = no contacts, or VCF content)
		_, _ = c.runner.Run("shell", "run-as", "com.sober.admin", "rm", "-f", "cache/sober_contacts.vcf")
		return out, nil
	}
	return "", fmt.Errorf("EXPORT_CONTACTS timed out — device did not respond within 15 seconds")
}

// ImportContacts pushes a VCF file to the phone and imports it via SoberAdmin.
// The file is pushed to the app's external files directory (no storage permission required).
func (c *Commands) ImportContacts(vcfPath string) error {
	const destPath = "/sdcard/Android/data/com.sober.admin/files/sober_contacts_restore.vcf"
	_, err := c.runner.Run("push", vcfPath, destPath)
	if err != nil {
		return fmt.Errorf("push contacts: %w", err)
	}

	_, _ = c.runner.Run("shell", "run-as", "com.sober.admin", "rm", "-f", "cache/sober_import_result.json")

	_, err = c.runner.Run(
		"shell", "am", "broadcast",
		"-a", "com.sober.IMPORT_CONTACTS",
		"-n", "com.sober.admin/.CommandReceiver",
	)
	if err != nil {
		return fmt.Errorf("IMPORT_CONTACTS broadcast: %w", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		out, err := c.runner.Run("shell", "run-as", "com.sober.admin", "cat", "cache/sober_import_result.json")
		if err != nil {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		out = strings.TrimSpace(out)
		if strings.HasPrefix(out, `{"success"`) {
			return nil
		}
		if strings.HasPrefix(out, `{"error"`) {
			return fmt.Errorf("IMPORT_CONTACTS failed on device: %s", out)
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("IMPORT_CONTACTS timed out — device did not respond within 15 seconds")
}

// ClearDeviceOwner removes SoberAdmin as Device Owner.
// Polls until the removal is confirmed or times out.
func (c *Commands) ClearDeviceOwner() error {
	_, err := c.runner.Run(
		"shell", "am", "broadcast",
		"-a", "com.sober.CLEAR_DEVICE_OWNER",
		"-n", "com.sober.admin/.CommandReceiver",
	)
	if err != nil {
		return fmt.Errorf("CLEAR_DEVICE_OWNER broadcast: %w", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !c.IsDeviceOwnerInstalled() {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("device owner not removed within 10 seconds")
}
```

- [ ] **Step 4: Run tests**

```bash
cd /home/logan/projects/sober/desktop
go test ./adb/... -v 2>&1 | tail -30
```
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add desktop/adb/commands.go desktop/adb/commands_test.go
git commit -m "feat: add CountGoogleAccounts, ExportContacts, ImportContacts, ClearDeviceOwner commands"
```

---

### Task 7: New Wails methods in app.go

**Files:**
- Modify: `desktop/app.go`
- Modify: `desktop/embed.go` — add `"path/filepath"` to imports if not present

- [ ] **Step 1: Add new import paths to app.go**

`app.go` will need these additional imports:
```go
import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "time"

    "github.com/sober/desktop/adb"
    "github.com/sober/desktop/config"
    wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)
```

- [ ] **Step 2: Remove RunSetup, add RunInstall**

Delete the existing `RunSetup` method entirely (it's replaced by the wizard's individual steps).

Add `RunInstall`:
```go
// RunInstall installs SoberAdmin, sets Device Owner, and applies restrictions.
// This is the automated install phase of the setup wizard.
func (a *App) RunInstall() error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	tmp, err := os.CreateTemp("", "sober-admin-*.apk")
	if err != nil {
		return fmt.Errorf("create temp APK: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(soberAdminAPK); err != nil {
		tmp.Close()
		return fmt.Errorf("write APK: %w", err)
	}
	tmp.Close()
	if err := a.commands.InstallAPK(tmp.Name()); err != nil {
		return fmt.Errorf("install SoberAdmin: %w", err)
	}
	if err := a.commands.SetDeviceOwner(); err != nil {
		return fmt.Errorf("set device owner: %w", err)
	}
	if err := a.commands.ApplyRestrictions(); err != nil {
		return fmt.Errorf("apply restrictions: %w", err)
	}
	cfg, _ := config.Load()
	cfg.SetupMode = "device_owner"
	_ = config.Save(cfg)
	return nil
}
```

- [ ] **Step 3: Add wizard helper methods**

```go
// GetGoogleAccountCount returns how many Google accounts are on the device.
func (a *App) GetGoogleAccountCount() (int, error) {
	if !a.connected {
		return 0, fmt.Errorf("no phone connected")
	}
	return a.commands.CountGoogleAccounts()
}

// OpenAccountSettings opens the Android Accounts settings screen on the phone.
func (a *App) OpenAccountSettings() error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	return a.commands.OpenAccountSettings()
}

// ExportContactsToDesktop exports contacts from the phone and saves them locally.
// Returns the saved file path. Saves the path to config for later restore.
func (a *App) ExportContactsToDesktop() (string, error) {
	if !a.connected {
		return "", fmt.Errorf("no phone connected")
	}
	vcf, err := a.commands.ExportContacts()
	if err != nil {
		return "", err
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config dir: %w", err)
	}
	soberDir := filepath.Join(dir, "sober")
	if err := os.MkdirAll(soberDir, 0700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	timestamp := time.Now().Format("20060102-150405")
	path := filepath.Join(soberDir, fmt.Sprintf("contacts-backup-%s.vcf", timestamp))
	if err := os.WriteFile(path, []byte(vcf), 0600); err != nil {
		return "", fmt.Errorf("write contacts backup: %w", err)
	}
	// Save path even when vcf is empty (no contacts) — the backup file exists and
	// ImportContactsFromBackup will correctly import zero contacts in that case.
	cfg, _ := config.Load()
	cfg.ContactsBackupPath = path
	_ = config.Save(cfg)
	return path, nil
}

// GetContactsBackupInfo returns info about the saved backup, or nil if none exists.
// Note: config.Load() never returns a non-nil error; it always falls back to defaults.
func (a *App) GetContactsBackupInfo() map[string]interface{} {
	cfg, _ := config.Load()
	if cfg.ContactsBackupPath == "" {
		return nil
	}
	if _, err := os.Stat(cfg.ContactsBackupPath); err != nil {
		return nil
	}
	base := filepath.Base(cfg.ContactsBackupPath)
	dateStr := ""
	const prefix = "contacts-backup-"
	const tsLen = len("20060102-150405")
	if len(base) >= len(prefix)+tsLen {
		ts := base[len(prefix) : len(prefix)+tsLen]
		if t, err := time.Parse("20060102-150405", ts); err == nil {
			dateStr = t.Format("January 2, 2006 at 3:04 PM")
		}
	}
	return map[string]interface{}{
		"path": cfg.ContactsBackupPath,
		"date": dateStr,
	}
}

// RunReset shows all hidden apps and removes Device Owner.
// Must be called before the phone is disconnected. Contacts restore is separate.
func (a *App) RunReset() error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	// Step 1: Show all hidden apps (must happen before Device Owner is removed)
	apps, err := a.appManager.ListApps()
	if err != nil {
		return fmt.Errorf("list apps for reset: %w", err)
	}
	for _, app := range apps {
		if app.Hidden {
			if err := a.appManager.ShowApp(app.Package); err != nil {
				return fmt.Errorf("show %s: %w", app.Package, err)
			}
		}
	}
	// Step 2: Remove Device Owner
	if err := a.commands.ClearDeviceOwner(); err != nil {
		return fmt.Errorf("clear device owner: %w", err)
	}
	return nil
}

// ImportContactsFromBackup restores contacts from the saved backup file.
func (a *App) ImportContactsFromBackup() error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	cfg, err := config.Load()
	if err != nil || cfg.ContactsBackupPath == "" {
		return fmt.Errorf("no contacts backup found")
	}
	if _, err := os.Stat(cfg.ContactsBackupPath); err != nil {
		return fmt.Errorf("contacts backup file not found: %s", cfg.ContactsBackupPath)
	}
	return a.commands.ImportContacts(cfg.ContactsBackupPath)
}
```

- [ ] **Step 4: Build to verify no compile errors**

```bash
cd /home/logan/projects/sober/desktop
go build ./... 2>&1
```
Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add desktop/app.go
git commit -m "feat: add wizard and reset Wails methods, remove RunSetup"
```

---

## Chunk 3: Frontend — wails.ts bindings + SetupTab wizard

> **Dependency:** Chunk 2 must be complete before starting Chunk 3. `wails.ts` imports `RunInstall` (replacing `RunSetup`), and `SetupTab.svelte` calls `runInstall`. If the Go rename hasn't happened, the Wails build will fail. Tasks 8 and 9 must be committed together — committing Task 8 alone breaks the app since the old `SetupTab.svelte` still imports `runSetup`.

### Task 8: Update wails.ts bindings

**Files:**
- Modify: `desktop/frontend/src/lib/wails.ts`

- [ ] **Step 1: Add new bindings**

Replace the contents of `desktop/frontend/src/lib/wails.ts` with:

```typescript
// Typed wrappers around Wails-generated Go bindings
// @ts-ignore — generated by wails at build time
import {
  GetConnectionStatus, GetApps, HideApp, ShowApp, InstallAPK,
  IsDeviceOwnerInstalled, OpenFileDialog, UpdateAdmin, UninstallApp,
  GetKnownStores,
  GetGoogleAccountCount, OpenAccountSettings, ExportContactsToDesktop,
  GetContactsBackupInfo, RunInstall, RunReset, ImportContactsFromBackup
} from '../../wailsjs/go/main/App'
// @ts-ignore
import { EventsOn } from '../../wailsjs/runtime/runtime'

export interface App {
  package: string
  label: string
  icon: string
  hidden: boolean
}

export interface ConnectionStatus {
  connected: boolean
  serial: string
}

export interface ContactsBackupInfo {
  path: string
  date: string
}

export const getConnectionStatus = (): Promise<ConnectionStatus> => GetConnectionStatus()
export const getApps = (): Promise<App[]> => GetApps()
export const hideApp = (pkg: string): Promise<void> => HideApp(pkg)
export const showApp = (pkg: string): Promise<void> => ShowApp(pkg)
export const installAPK = (path: string): Promise<void> => InstallAPK(path)
export const openFileDialog = (): Promise<string> => OpenFileDialog()
export const isDeviceOwnerInstalled = (): Promise<boolean> => IsDeviceOwnerInstalled()
export const onConnectionChange = (cb: (status: ConnectionStatus) => void) => {
  EventsOn('connection:change', cb)
}
export const updateAdmin = (): Promise<void> => UpdateAdmin()
export const onAdminVersionMismatch = (cb: (info: { installedVersion: number, bundledVersion: number }) => void) => {
  EventsOn('admin:version-mismatch', cb)
}
export const uninstallApp = (pkg: string): Promise<void> => UninstallApp(pkg)
export const getKnownStores = (): Promise<string[]> => GetKnownStores()

// Wizard
export const getGoogleAccountCount = (): Promise<number> => GetGoogleAccountCount()
export const openAccountSettings = (): Promise<void> => OpenAccountSettings()
export const exportContactsToDesktop = (): Promise<string> => ExportContactsToDesktop()
export const getContactsBackupInfo = (): Promise<ContactsBackupInfo | null> => GetContactsBackupInfo()
export const runInstall = (): Promise<void> => RunInstall()

// Reset
export const runReset = (): Promise<void> => RunReset()
export const importContactsFromBackup = (): Promise<void> => ImportContactsFromBackup()
```

Note: Do NOT commit wails.ts alone — the old SetupTab.svelte still imports `runSetup` which no longer exists. Stage both files together in the Task 9 commit.

---

### Task 9: Rewrite SetupTab.svelte as wizard

**Files:**
- Modify: `desktop/frontend/src/components/SetupTab.svelte`
- Modify: `desktop/frontend/src/App.svelte` (add resetcomplete handler)

- [ ] **Step 1: Add resetcomplete handler to App.svelte**

In `App.svelte`, in the `<script>` block, add after `handleSetupComplete`:
```typescript
function handleResetComplete() {
  deviceOwnerInstalled = false
}
```

In the template, change:
```svelte
<SetupTab {connected} {deviceOwnerInstalled} on:setupcomplete={handleSetupComplete} />
```
to:
```svelte
<SetupTab {connected} {deviceOwnerInstalled} on:setupcomplete={handleSetupComplete} on:resetcomplete={handleResetComplete} />
```

- [ ] **Step 2: Rewrite SetupTab.svelte**

Replace the entire contents of `desktop/frontend/src/components/SetupTab.svelte`:

```svelte
<script lang="ts">
  import { onDestroy, createEventDispatcher } from 'svelte'
  import {
    getGoogleAccountCount, openAccountSettings, exportContactsToDesktop,
    getContactsBackupInfo, runInstall, runReset, importContactsFromBackup
  } from '../lib/wails'
  import type { ContactsBackupInfo } from '../lib/wails'

  export let connected: boolean
  export let deviceOwnerInstalled: boolean

  const dispatch = createEventDispatcher()

  // ── Wizard state (setup mode) ───────────────────────────────────────────
  type WizardStep =
    | 'detect'
    | 'backup-consent'
    | 'backing-up'
    | 'guide-removal'
    | 'installing'
    | 'success'
    | 'error'

  let wizardStep: WizardStep = 'detect'
  let accountCount = 0
  let backupPath = ''
  let errorMessage = ''
  let pollInterval: ReturnType<typeof setInterval> | null = null
  let disconnectedDuringPoll = false
  let isDetecting = false  // guard against re-entrant detectAccounts() calls

  // ── Reset state (post-setup mode) ───────────────────────────────────────
  type ResetState = 'idle' | 'confirm' | 'resetting' | 'restore-prompt' | 'restoring' | 'reset-done'
  let resetState: ResetState = 'idle'
  let resetError = ''
  let backupInfo: ContactsBackupInfo | null = null

  // ── Auto-detect on connect ───────────────────────────────────────────────
  // Guard against re-entrancy: if connected toggles rapidly while detectAccounts()
  // is awaiting, only one invocation should run at a time.
  $: if (connected && !deviceOwnerInstalled && wizardStep === 'detect' && !isDetecting) {
    detectAccounts()
  }

  // Only track disconnect when it can meaningfully pause the wizard.
  $: if (!connected && (wizardStep === 'guide-removal' || wizardStep === 'detect')) {
    disconnectedDuringPoll = true
    stopPoll()
  }

  $: if (connected && disconnectedDuringPoll) {
    disconnectedDuringPoll = false
    if (wizardStep === 'guide-removal') startPoll()
    if (wizardStep === 'detect') detectAccounts()
  }

  async function detectAccounts() {
    isDetecting = true
    try {
      accountCount = await getGoogleAccountCount()
      if (accountCount === 0) {
        // No accounts — skip backup and account-removal steps, go straight to install.
        await doInstall()
      } else {
        wizardStep = 'backup-consent'
      }
    } catch (e: any) {
      // Ignore — phone may have disconnected; reactive block will handle reconnect
    } finally {
      isDetecting = false
    }
  }

  async function skipBackupAndProceed() {
    wizardStep = 'guide-removal'
    startPoll()
    openAccountSettings().catch(() => {})
  }

  async function doBackup() {
    wizardStep = 'backing-up'
    try {
      backupPath = await exportContactsToDesktop()
      wizardStep = 'guide-removal'
      startPoll()
      openAccountSettings().catch(() => {})
    } catch (e: any) {
      errorMessage = e?.message ?? String(e)
      wizardStep = 'error'
    }
  }

  function startPoll() {
    stopPoll()
    pollInterval = setInterval(async () => {
      if (!connected) return
      try {
        accountCount = await getGoogleAccountCount()
        if (accountCount === 0) {
          stopPoll()
          await doInstall()
        }
      } catch {
        // Phone disconnected — reactive block handles resume
      }
    }, 2000)
  }

  function stopPoll() {
    if (pollInterval !== null) {
      clearInterval(pollInterval)
      pollInterval = null
    }
  }

  async function doInstall() {
    wizardStep = 'installing'
    try {
      await runInstall()
      wizardStep = 'success'
      deviceOwnerInstalled = true
      dispatch('setupcomplete')
    } catch (e: any) {
      errorMessage = e?.message ?? String(e)
      wizardStep = 'error'
    }
  }

  function retryFromStart() {
    stopPoll()
    errorMessage = ''
    wizardStep = 'detect'
  }

  // ── Reset flow ────────────────────────────────────────────────────────────
  async function startReset() {
    resetState = 'resetting'
    resetError = ''
    try {
      await runReset()
      backupInfo = await getContactsBackupInfo()
      if (backupInfo) {
        resetState = 'restore-prompt'
      } else {
        resetState = 'reset-done'
        deviceOwnerInstalled = false
        dispatch('resetcomplete')
      }
    } catch (e: any) {
      resetError = e?.message ?? String(e)
      resetState = 'idle'
    }
  }

  async function doRestore() {
    resetState = 'restoring'
    try {
      await importContactsFromBackup()
    } catch {
      // Non-fatal — contacts restore is best-effort
    }
    resetState = 'reset-done'
    deviceOwnerInstalled = false
    dispatch('resetcomplete')
  }

  function skipRestore() {
    resetState = 'reset-done'
    deviceOwnerInstalled = false
    dispatch('resetcomplete')
  }

  onDestroy(stopPoll)
</script>

<div class="setup">
  <h2>Setup</h2>

  {#if deviceOwnerInstalled}
    <!-- ── Post-setup state ── -->
    <div class="banner success">
      SoberAdmin is installed and active as Device Owner. Your phone is locked down.
    </div>

    {#if resetState === 'idle'}
      <div class="reset-section">
        {#if resetError}
          <div class="banner error" style="margin-bottom: 12px">{resetError}</div>
        {/if}
        <button class="danger" on:click={() => resetState = 'confirm'}>
          Reset Phone
        </button>
      </div>

    {:else if resetState === 'confirm'}
      <div class="banner warning">
        <p>This will remove all restrictions, show all hidden apps, and remove SoberAdmin as Device Owner.</p>
        <div class="button-row">
          <button class="danger" on:click={startReset}>Reset Everything</button>
          <button class="secondary" on:click={() => resetState = 'idle'}>Cancel</button>
        </div>
      </div>

    {:else if resetState === 'resetting'}
      <div class="progress">
        <div class="spinner"></div>
        <p>Resetting phone — do not unplug…</p>
      </div>

    {:else if resetState === 'restore-prompt'}
      <div class="banner info">
        <p>Device Owner removed. We have a contacts backup from <strong>{backupInfo?.date}</strong>.</p>
        <p class="hint">Restore it to your phone?</p>
        <div class="button-row">
          <button class="primary" on:click={doRestore}>Restore contacts</button>
          <button class="secondary" on:click={skipRestore}>Skip</button>
        </div>
      </div>

    {:else if resetState === 'restoring'}
      <div class="progress">
        <div class="spinner"></div>
        <p>Restoring contacts…</p>
      </div>

    {:else if resetState === 'reset-done'}
      <div class="banner success">
        Your phone has been fully restored. SoberAdmin is no longer active.
      </div>
    {/if}

  {:else}
    <!-- ── Setup wizard ── -->

    {#if wizardStep === 'detect'}
      <div class="progress">
        <div class="spinner"></div>
        <p>{connected ? 'Checking phone…' : 'Waiting for phone connection…'}</p>
      </div>

    {:else if wizardStep === 'backup-consent'}
      <div class="wizard-step">
        <p class="step-lead">Before you remove your Google accounts, we can save a backup of your contacts to this computer.</p>
        <div class="info-box">
          This file stays on your computer only and is never uploaded anywhere.
        </div>
        <div class="button-col">
          <button class="primary" on:click={doBackup}>Save backup and continue</button>
          <button class="secondary" on:click={skipBackupAndProceed}>Skip — I'll take my chances</button>
        </div>
        <p class="hint">Skipping means if you accidentally tap "Remove data" during account removal, your local contacts won't be recoverable.</p>
      </div>

    {:else if wizardStep === 'backing-up'}
      <div class="progress">
        <div class="spinner"></div>
        <p>Saving contacts backup…</p>
      </div>

    {:else if wizardStep === 'guide-removal'}
      <div class="wizard-step">
        {#if disconnectedDuringPoll}
          <div class="banner warning">Phone disconnected — plug it back in to continue.</div>
        {:else}
          <p class="step-lead">
            {accountCount === 1 ? '1 Google account' : `${accountCount} Google accounts`}
            {accountCount === 1 ? 'needs' : 'need'} to be removed before setup can continue.
          </p>

          <div class="warn-box">
            <strong>Important:</strong> When Android asks what to do with your data — choose <strong>Keep</strong>.
            {backupPath ? ' Your contacts are also backed up to this computer, so they\'re safe either way.' : ''}
          </div>

          <button class="primary" on:click={() => openAccountSettings().catch(() => {})}>
            Open Account Settings on my phone
          </button>

          <div class="poll-status">
            <div class="spinner small"></div>
            Waiting… {accountCount} {accountCount === 1 ? 'account' : 'accounts'} remaining
          </div>

          {#if backupPath}
            <p class="hint">Contacts backup saved to: {backupPath}</p>
          {/if}
        {/if}
      </div>

    {:else if wizardStep === 'installing'}
      <div class="progress">
        <div class="spinner"></div>
        <p>Setting up SoberAdmin — do not unplug your phone…</p>
      </div>

    {:else if wizardStep === 'success'}
      <div class="banner success">
        Setup complete! Your phone is now locked down.<br>
        Switch to the <strong>Apps</strong> tab to manage app visibility.
      </div>
      <div class="readd-section">
        <p>You can now re-add your Google account. Your synced contacts will return automatically.</p>
        <button class="secondary" on:click={() => openAccountSettings().catch(() => {})}>
          Open Account Settings
        </button>
      </div>

    {:else if wizardStep === 'error'}
      <div class="banner error">
        <strong>Setup failed:</strong> {errorMessage}
        <button on:click={retryFromStart}>Try Again</button>
      </div>
    {/if}
  {/if}
</div>

<style>
  .setup { max-width: 600px; color: #e2e2e8; }
  h2 { margin-bottom: 16px; font-size: 20px; color: #e2e2e8; }

  .wizard-step { display: flex; flex-direction: column; gap: 16px; }
  .step-lead { color: #e2e2e8; line-height: 1.6; }
  .hint { color: #6b7280; font-size: 13px; line-height: 1.5; }

  .info-box {
    padding: 12px 16px; background: #1a1a2e; border: 1px solid #312e81;
    border-left: 4px solid #7c6af7; border-radius: 6px; color: #c4b5fd; font-size: 14px;
  }
  .warn-box {
    padding: 12px 16px; background: #1f1a0f; border: 1px solid #92400e;
    border-left: 4px solid #d97706; border-radius: 6px; color: #fcd34d; font-size: 14px; line-height: 1.5;
  }

  .button-col { display: flex; flex-direction: column; gap: 10px; }
  .button-row { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 12px; }

  .poll-status {
    display: flex; align-items: center; gap: 10px;
    color: #9ca3af; font-size: 14px; margin-top: 8px;
  }

  .reset-section { margin-top: 20px; }
  .readd-section { margin-top: 20px; display: flex; flex-direction: column; gap: 12px; color: #9ca3af; font-size: 14px; }

  .primary {
    padding: 11px 28px; background: linear-gradient(135deg, #7c6af7, #a78bfa);
    color: white; border: none; border-radius: 6px; font-size: 15px; cursor: pointer;
    transition: opacity 0.15s; align-self: flex-start;
  }
  .primary:hover { opacity: 0.88; }

  .secondary {
    padding: 10px 20px; background: #1f1f2e; color: #9ca3af;
    border: 1px solid #2a2a38; border-radius: 6px; font-size: 14px; cursor: pointer;
    transition: background 0.15s; align-self: flex-start;
  }
  .secondary:hover { background: #2a2a3a; }

  .danger {
    padding: 10px 20px; background: #1f1515; color: #f87171;
    border: 1px solid #7f1d1d; border-radius: 6px; font-size: 14px; cursor: pointer;
    transition: background 0.15s;
  }
  .danger:hover { background: #2a1515; }

  .progress { display: flex; align-items: center; gap: 16px; margin-top: 8px; color: #9ca3af; }

  .spinner {
    width: 24px; height: 24px; flex-shrink: 0;
    border: 3px solid #2a2a38; border-top-color: #a78bfa;
    border-radius: 50%; animation: spin 0.8s linear infinite;
  }
  .spinner.small { width: 16px; height: 16px; border-width: 2px; }
  @keyframes spin { to { transform: rotate(360deg); } }

  .banner {
    padding: 16px; border-radius: 6px; margin-top: 8px; line-height: 1.6;
  }
  .banner.success { background: #1a2a1f; border: 1px solid #166534; border-left: 4px solid #166534; color: #4ade80; }
  .banner.error {
    background: #1f1515; border: 1px solid #7f1d1d; border-left: 4px solid #7f1d1d; color: #f87171;
    display: flex; align-items: center; gap: 16px; flex-wrap: wrap;
  }
  .banner.error button {
    padding: 4px 12px; cursor: pointer; background: #1f1f2e; color: #f87171;
    border: 1px solid #7f1d1d; border-radius: 4px;
  }
  .banner.warning { background: #1f1a0f; border: 1px solid #92400e; border-left: 4px solid #d97706; color: #fcd34d; }
  .banner.info { background: #1a1a2e; border: 1px solid #312e81; border-left: 4px solid #7c6af7; color: #c4b5fd; }
  .banner p { margin-bottom: 6px; }
  .banner .hint { color: #9ca3af; font-size: 13px; }
</style>
```

- [ ] **Step 3: Verify the app builds (frontend + backend)**

```bash
cd /home/logan/projects/sober/desktop
wails build 2>&1 | tail -20
```
Expected: BUILD SUCCESSFUL. If wails is not available for a full build, verify the frontend alone:
```bash
cd /home/logan/projects/sober/desktop/frontend
npm run build 2>&1 | tail -10
```
Expected: no errors.

- [ ] **Step 4: Commit all frontend files atomically**

Both `wails.ts` and `SetupTab.svelte` must be committed together since the old `runSetup` import is removed in `wails.ts` and the new `runInstall` is used in the new `SetupTab.svelte`.

```bash
git add desktop/frontend/src/lib/wails.ts \
        desktop/frontend/src/components/SetupTab.svelte \
        desktop/frontend/src/App.svelte
git commit -m "feat(frontend): rewrite SetupTab as guided wizard with reset flow"
```

---

## Final verification

- [ ] Run all Go tests:
```bash
cd /home/logan/projects/sober/desktop
go test ./... 2>&1
```
Expected: all pass.

- [ ] Run all Android tests:
```bash
cd /home/logan/projects/sober/android
./gradlew test 2>&1 | tail -10
```
Expected: BUILD SUCCESSFUL.

- [ ] Final commit if any loose files remain:
```bash
cd /home/logan/projects/sober
git status
```

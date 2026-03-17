package adb

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// knownStores lists package names of known app stores.
var knownStores = []string{
	"com.android.vending",             // Google Play Store
	"com.sec.android.app.samsungapps", // Galaxy Store
	"com.amazon.venezia",              // Amazon Appstore
	"com.huawei.appmarket",            // Huawei AppGallery
	"com.xiaomi.market",               // Mi GetApps
	"com.oppo.market",                 // OPPO App Market
}

// GetKnownStoreList returns the list of known app store package names.
func GetKnownStoreList() []string {
	return knownStores
}

// Executor runs ADB commands. Implemented by *Runner and fake runners in tests.
type Executor interface {
	Run(args ...string) (string, error)
}

// App represents an installed app returned by LIST_APPS.
type App struct {
	Package string `json:"package"`
	Label   string `json:"label"`
	Icon    string `json:"icon"` // base64-encoded PNG
	Hidden  bool   `json:"hidden"`
}

// Commands sends ADB broadcast commands to the SoberAdmin APK.
type Commands struct {
	runner Executor
}

// NewCommands creates a Commands using the given Executor.
func NewCommands(r Executor) *Commands {
	return &Commands{runner: r}
}

// HideApp hides the given package via the SoberAdmin APK.
func (c *Commands) HideApp(pkg string) error {
	_, err := c.runner.Run(
		"shell", "am", "broadcast",
		"-a", "com.sober.HIDE_APP",
		"-n", "com.sober.admin/.CommandReceiver",
		"--es", "package", pkg,
	)
	return err
}

// ShowApp makes the given package visible again.
func (c *Commands) ShowApp(pkg string) error {
	_, err := c.runner.Run(
		"shell", "am", "broadcast",
		"-a", "com.sober.SHOW_APP",
		"-n", "com.sober.admin/.CommandReceiver",
		"--es", "package", pkg,
	)
	return err
}

// UninstallApp uninstalls the given package.
func (c *Commands) UninstallApp(pkg string) error {
	_, err := c.runner.Run(
		"shell", "pm", "uninstall",
		"--user", "0",
		pkg,
	)
	if err != nil {
		return fmt.Errorf("uninstall %s: %w", pkg, err)
	}
	return nil
}

// ApplyRestrictions broadcasts APPLY_RESTRICTIONS to enforce DISALLOW_INSTALL_UNKNOWN_SOURCES.
func (c *Commands) ApplyRestrictions() error {
	_, err := c.runner.Run(
		"shell", "am", "broadcast",
		"-a", "com.sober.APPLY_RESTRICTIONS",
		"-n", "com.sober.admin/.CommandReceiver",
	)
	return err
}

// ListApps fetches the current app list from the phone via LIST_APPS.
// Polls for the output file with a 5-second timeout.
func (c *Commands) ListApps() ([]App, error) {
	// Pre-check: give a clear error immediately if Device Owner is not set.
	if !c.IsDeviceOwnerInstalled() {
		return nil, fmt.Errorf("SoberAdmin is not the Device Owner — run Setup first")
	}

	// Delete any stale output file before broadcasting.
	_, _ = c.runner.Run("shell", "run-as", "com.sober.admin", "rm", "-f", "cache/sober_apps.json")

	_, err := c.runner.Run(
		"shell", "am", "broadcast",
		"-a", "com.sober.LIST_APPS",
		"-n", "com.sober.admin/.CommandReceiver",
	)
	if err != nil {
		return nil, fmt.Errorf("LIST_APPS broadcast: %w", err)
	}
	// Note: am broadcast always outputs "result=0" regardless of whether the
	// receiver ran. We detect success/failure by polling for the output file.

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		out, err := c.runner.Run("shell", "run-as", "com.sober.admin", "cat", "cache/sober_apps.json")
		if err != nil {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		out = strings.TrimSpace(out)
		// Receiver wrote an error JSON: {"error":"..."}
		if strings.HasPrefix(out, `{"error"`) {
			return nil, fmt.Errorf("LIST_APPS failed on device: %s", out)
		}
		if strings.HasPrefix(out, "[") {
			_, _ = c.runner.Run("shell", "run-as", "com.sober.admin", "rm", "-f", "cache/sober_apps.json")
			return ParseAppList(out)
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil, fmt.Errorf("LIST_APPS timed out — receiver did not write output within 15 seconds")
}

// InstallAPK installs an APK file onto the connected phone.
func (c *Commands) InstallAPK(path string) error {
	out, err := c.runner.Run("install", "-r", path)
	if err != nil {
		return err
	}
	if !strings.Contains(out, "Success") {
		return fmt.Errorf("install failed: %s", out)
	}
	return nil
}

// InstallSplitAPKs installs a set of APK splits using an adb install-multiple session.
func (c *Commands) InstallSplitAPKs(paths []string) error {
	args := append([]string{"install-multiple", "-r"}, paths...)
	out, err := c.runner.Run(args...)
	if err != nil {
		return err
	}
	if !strings.Contains(out, "Success") {
		return fmt.Errorf("install failed: %s", out)
	}
	return nil
}

// CheckAccounts returns an error if user accounts are present on the device,
// which would prevent set-device-owner from succeeding.
// Fails open (returns nil) if the check itself cannot run.
func (c *Commands) CheckAccounts() error {
	out, err := c.runner.Run("shell", "dumpsys", "account")
	if err != nil {
		return nil // can't check, proceed; SetDeviceOwner will catch it
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Account {") && strings.Contains(line, "type=com.google") {
			return fmt.Errorf("Google accounts are still on this device — " +
				"go to Settings › Accounts and remove them all, then try again")
		}
	}
	return nil
}

// checkExistingDeviceOwner returns a user-friendly error if any device owner is
// already set, nil otherwise. Fails open on runner error.
func (c *Commands) checkExistingDeviceOwner() error {
	out, err := c.runner.Run("shell", "dpm", "list-owners")
	if err != nil {
		return nil // can't check; the actual set-device-owner call will fail if needed
	}
	out = strings.TrimSpace(out)
	if strings.Contains(out, "com.sober.admin") {
		return fmt.Errorf("Accountability Mode is already active on this phone")
	}
	// Only report "another app" if output contains "/" (Android component name format).
	// This avoids false positives from unexpected empty-owner output formats.
	if strings.Contains(out, "/") {
		return fmt.Errorf("Another app is controlling this phone. It must be removed before Sober can be set up")
	}
	return nil
}

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

// IsDeviceOwnerInstalled checks whether SoberAdmin is currently the Device Owner.
func (c *Commands) IsDeviceOwnerInstalled() bool {
	out, err := c.runner.Run("shell", "dpm", "list-owners")
	if err != nil {
		return false
	}
	return strings.Contains(out, "com.sober.admin")
}

// GetInstalledAdminVersionCode returns the versionCode of the installed
// com.sober.admin package, or 0 if it is not installed.
func (c *Commands) GetInstalledAdminVersionCode() (int, error) {
	out, err := c.runner.Run("shell", "dumpsys", "package", "com.sober.admin")
	if err != nil {
		return 0, fmt.Errorf("dumpsys package: %w", err)
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "versionCode=") {
			continue
		}
		// format: "versionCode=7 targetSdk=33"
		parts := strings.SplitN(strings.Fields(line)[0], "=", 2)
		v, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("parse versionCode: %w", err)
		}
		return v, nil
	}
	return 0, nil // not installed
}

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
			_, _ = c.runner.Run("shell", "run-as", "com.sober.admin", "rm", "-f", "cache/sober_import_result.json")
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

// ParseAppList parses the JSON app list written by LIST_APPS.
func ParseAppList(jsonStr string) ([]App, error) {
	var apps []App
	if err := json.Unmarshal([]byte(jsonStr), &apps); err != nil {
		return nil, fmt.Errorf("parse app list: %w", err)
	}
	return apps, nil
}

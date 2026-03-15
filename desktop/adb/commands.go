package adb

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// KnownStores lists package names of known app stores.
var KnownStores = []string{
	"com.android.vending",             // Google Play Store
	"com.sec.android.app.samsungapps", // Galaxy Store
	"com.amazon.venezia",              // Amazon Appstore
	"com.huawei.appmarket",            // Huawei AppGallery
	"com.xiaomi.market",               // Mi GetApps
	"com.oppo.market",                 // OPPO App Market
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
	return err
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

// SetDeviceOwner grants Device Owner to SoberAdmin.
func (c *Commands) SetDeviceOwner() error {
	out, err := c.runner.Run(
		"shell", "dpm", "set-device-owner",
		"com.sober.admin/.AdminReceiver",
	)
	if err != nil {
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

// ParseAppList parses the JSON app list written by LIST_APPS.
func ParseAppList(jsonStr string) ([]App, error) {
	var apps []App
	if err := json.Unmarshal([]byte(jsonStr), &apps); err != nil {
		return nil, fmt.Errorf("parse app list: %w", err)
	}
	return apps, nil
}

package adb

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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
// Polls for the output file with a 10-second timeout.
func (c *Commands) ListApps() ([]App, error) {
	_, err := c.runner.Run(
		"shell", "am", "broadcast",
		"-a", "com.sober.LIST_APPS",
		"-n", "com.sober.admin/.CommandReceiver",
	)
	if err != nil {
		return nil, fmt.Errorf("LIST_APPS broadcast: %w", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		out, err := c.runner.Run("shell", "cat", "/data/local/tmp/sober_apps.json")
		if err == nil && strings.HasPrefix(strings.TrimSpace(out), "[") {
			_, _ = c.runner.Run("shell", "rm", "/data/local/tmp/sober_apps.json")
			return ParseAppList(out)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("LIST_APPS timed out after 10 seconds — check SoberAdmin is installed as Device Owner")
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

// SetDeviceOwner grants Device Owner to SoberAdmin.
func (c *Commands) SetDeviceOwner() error {
	out, err := c.runner.Run(
		"shell", "dpm", "set-device-owner",
		"com.sober.admin/.AdminReceiver",
	)
	if err != nil {
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

// ParseAppList parses the JSON app list written by LIST_APPS.
func ParseAppList(jsonStr string) ([]App, error) {
	var apps []App
	if err := json.Unmarshal([]byte(jsonStr), &apps); err != nil {
		return nil, fmt.Errorf("parse app list: %w", err)
	}
	return apps, nil
}

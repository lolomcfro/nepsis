package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/sober/desktop/adb"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App holds all Wails-bound methods. One instance per application lifetime.
type App struct {
	ctx      context.Context
	runner   *adb.Runner
	commands *adb.Commands
	poller   *adb.Poller

	connected bool
	serial    string
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// startup is called when the Wails app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Inject the platform-appropriate ADB binary
	switch runtime.GOOS {
	case "linux":
		adb.SetBinary(adbLinux)
	case "darwin":
		adb.SetBinary(adbDarwin)
	case "windows":
		adb.SetBinary(adbWindows)
	default:
		// Unsupported OS — app will show an error when commands are attempted
		return
	}

	runner, err := adb.NewRunner()
	if err != nil {
		// Surface error to frontend via connection status
		return
	}
	a.runner = runner
	a.commands = adb.NewCommands(runner)

	a.poller = adb.NewPoller(runner, 2*time.Second, a.onConnectionChange)
	a.poller.Start()
}

func (a *App) onConnectionChange(connected bool, serial string) {
	a.connected = connected
	a.serial = serial
	wailsruntime.EventsEmit(a.ctx, "connection:change", map[string]interface{}{
		"connected": connected,
		"serial":    serial,
	})
}

// --- Wails-bound methods (called from frontend) ---

// GetConnectionStatus returns the current connection state.
func (a *App) GetConnectionStatus() map[string]interface{} {
	return map[string]interface{}{
		"connected": a.connected,
		"serial":    a.serial,
	}
}

// IsDeviceOwnerInstalled checks whether SoberAdmin is the Device Owner.
func (a *App) IsDeviceOwnerInstalled() bool {
	if !a.connected {
		return false
	}
	return a.commands.IsDeviceOwnerInstalled()
}

// GetApps returns the live app list from the phone.
func (a *App) GetApps() ([]adb.App, error) {
	if !a.connected {
		return nil, fmt.Errorf("no phone connected")
	}
	return a.commands.ListApps()
}

// HideApp hides the given package.
func (a *App) HideApp(pkg string) error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	return a.commands.HideApp(pkg)
}

// ShowApp makes the given package visible.
func (a *App) ShowApp(pkg string) error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	return a.commands.ShowApp(pkg)
}

// InstallAPK installs an APK from the given local path.
func (a *App) InstallAPK(path string) error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	return a.commands.InstallAPK(path)
}

// RunSetup executes the full setup flow.
func (a *App) RunSetup() error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}

	// Step 1: Install SoberAdmin APK
	tmpAPK, err := os.CreateTemp("", "sober-admin-*.apk")
	if err != nil {
		return fmt.Errorf("create temp APK: %w", err)
	}
	defer os.Remove(tmpAPK.Name())

	if _, err := tmpAPK.Write(soberAdminAPK); err != nil {
		return fmt.Errorf("write APK: %w", err)
	}
	tmpAPK.Close()

	if err := a.commands.InstallAPK(tmpAPK.Name()); err != nil {
		return fmt.Errorf("install SoberAdmin: %w", err)
	}

	// Step 2: Grant Device Owner
	if err := a.commands.SetDeviceOwner(); err != nil {
		return fmt.Errorf("set device owner: %w", err)
	}

	// Step 3: Apply baseline restrictions
	if err := a.commands.ApplyRestrictions(); err != nil {
		return fmt.Errorf("apply restrictions: %w", err)
	}

	return nil
}

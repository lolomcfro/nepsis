package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sober/desktop/adb"
	"github.com/sober/desktop/config"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App holds all Wails-bound methods. One instance per application lifetime.
type App struct {
	ctx        context.Context
	runner     *adb.Runner
	commands   *adb.Commands  // setup/teardown operations only
	appManager adb.AppManager // hide/show/list/uninstall (mode-agnostic)
	poller     *adb.Poller

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

	runner, err := adb.NewAutoRunner()
	if err != nil {
		// Surface error to frontend via connection status
		return
	}
	a.runner = runner
	a.commands = adb.NewCommands(runner)
	a.appManager = a.commands

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
	if connected {
		go a.maybeUpdateAdmin()
	}
}

func (a *App) maybeUpdateAdmin() {
	installed, err := a.commands.GetInstalledAdminVersionCode()
	if err != nil || installed == 0 || installed >= BundledAdminVersion {
		return // error, not installed, or up to date
	}
	wailsruntime.EventsEmit(a.ctx, "admin:version-mismatch", map[string]interface{}{
		"installedVersion": installed,
		"bundledVersion":   BundledAdminVersion,
	})
}

// UpdateAdmin installs the bundled SoberAdmin APK onto the connected phone.
func (a *App) UpdateAdmin() error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	tmp, err := os.CreateTemp("", "sober-admin-*.apk")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(soberAdminAPK); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	return a.commands.InstallAPK(tmp.Name())
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
	return a.appManager.ListApps()
}

// HideApp hides the given package.
func (a *App) HideApp(pkg string) error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	return a.appManager.HideApp(pkg)
}

// ShowApp makes the given package visible.
func (a *App) ShowApp(pkg string) error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	return a.appManager.ShowApp(pkg)
}

// UninstallApp uninstalls the given package.
func (a *App) UninstallApp(pkg string) error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	return a.appManager.UninstallApp(pkg)
}

// GetKnownStores returns the list of known app store package names.
func (a *App) GetKnownStores() []string {
	return adb.GetKnownStoreList()
}

// OpenFileDialog opens a native file picker and returns the selected path.
func (a *App) OpenFileDialog() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "App Packages", Pattern: "*.apk;*.apkm;*.xapk;*.apks"},
		},
	})
}

// extractAPKsFromZip extracts all *.apk entries from a ZIP archive to a temp
// directory. Returns the extracted paths and a cleanup func. Returns an error
// if the archive contains no APK entries.
func extractAPKsFromZip(zipPath string) (paths []string, cleanup func(), err error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open bundle: %w", err)
	}
	defer r.Close()

	dir, err := os.MkdirTemp("", "sober-apkbundle-*")
	if err != nil {
		return nil, func() {}, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup = func() { os.RemoveAll(dir) }

	for _, f := range r.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".apk") {
			continue
		}
		dest := filepath.Join(dir, filepath.Base(f.Name))
		rc, err := f.Open()
		if err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("open entry %s: %w", f.Name, err)
		}
		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			cleanup()
			return nil, func() {}, fmt.Errorf("create %s: %w", dest, err)
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("extract %s: %w", f.Name, err)
		}
		paths = append(paths, dest)
	}

	if len(paths) == 0 {
		cleanup()
		return nil, func() {}, fmt.Errorf("bundle contains no APK files")
	}
	return paths, cleanup, nil
}

// InstallAPK installs an APK or split bundle from the given local path.
func (a *App) InstallAPK(path string) error {
	if !a.connected {
		return fmt.Errorf("no phone connected")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".apkm", ".xapk", ".apks":
		paths, cleanup, err := extractAPKsFromZip(path)
		defer cleanup()
		if err != nil {
			return err
		}
		return a.commands.InstallSplitAPKs(paths)
	default:
		return a.commands.InstallAPK(path)
	}
}

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
	wailsruntime.EventsEmit(a.ctx, "reset:step", map[string]interface{}{"step": "unhide", "status": "running"})
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
	wailsruntime.EventsEmit(a.ctx, "reset:step", map[string]interface{}{"step": "unhide", "status": "done"})
	// Step 2: Remove Device Owner
	wailsruntime.EventsEmit(a.ctx, "reset:step", map[string]interface{}{"step": "device-owner", "status": "running"})
	if err := a.commands.ClearDeviceOwner(); err != nil {
		return fmt.Errorf("clear device owner: %w", err)
	}
	wailsruntime.EventsEmit(a.ctx, "reset:step", map[string]interface{}{"step": "device-owner", "status": "done"})
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

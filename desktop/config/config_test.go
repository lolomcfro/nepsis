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

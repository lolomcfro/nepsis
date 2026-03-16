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

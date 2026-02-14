package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds persistent application configuration.
type Config struct {
	Devices map[string]string `json:"devices"` // IP → friendly name
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".awair-tui.json"
	}
	return filepath.Join(home, ".awair-tui.json")
}

// LoadConfig reads the config file from ~/.awair-tui.json.
// Returns an empty config on any error.
func LoadConfig() *Config {
	cfg := &Config{Devices: make(map[string]string)}

	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}

	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		return cfg
	}

	if parsed.Devices != nil {
		cfg.Devices = parsed.Devices
	}
	return cfg
}

// SaveConfig writes the config to ~/.awair-tui.json.
// Errors are silently ignored.
func SaveConfig(cfg *Config) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	data = append(data, '\n')
	_ = os.WriteFile(configPath(), data, 0644)
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigDir  = ".remdoc"
	ConfigFile = "config.json"
)

// Config represents the CLI's persistent configuration
type Config struct {
	PortainerURL string `json:"portainer_url"`
	JWT          string `json:"jwt"`
}

// ConfigPath returns the absolute path to the config file
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ConfigDir, ConfigFile), nil
}

// Load reads the config from disk
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found (run 'remdoc login' first)")
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config format: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to disk
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	// Ensure the config directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil { // 0700 = user-only access
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Write with user-only permissions (critical for JWT security)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
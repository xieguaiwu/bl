package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Mode represents the dictionary query mode.
type Mode string

const (
	ModeAuto    Mode = "auto"
	ModeOffline Mode = "offline"
	ModeOnline  Mode = "online"
)

// ValidModes returns all valid mode values.
func ValidModes() []Mode {
	return []Mode{ModeAuto, ModeOffline, ModeOnline}
}

// IsValid checks if a mode string is valid.
func IsValidMode(s string) bool {
	switch Mode(s) {
	case ModeAuto, ModeOffline, ModeOnline:
		return true
	}
	return false
}

// Config holds persistent configuration for bl.
type Config struct {
	// Mode is the default dictionary query mode:
	//   "auto"    — try offline first, fall back to online (default)
	//   "offline" — offline only, error if word not in local dict
	//   "online"  — skip offline dict, always fetch from network
	Mode Mode `json:"mode"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{Mode: ModeAuto}
}

// ConfigPath returns the path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "bl", "config.json"), nil
}

// Load reads the config file from the standard location.
// Returns a default config if the file does not exist.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return DefaultConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if !IsValidMode(string(cfg.Mode)) {
		cfg.Mode = ModeAuto
	}
	return &cfg, nil
}

// Save writes the config to the standard location.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// GenerateConfig creates a default config file if one does not exist.
// Returns true if a new file was created.
func GenerateConfig() (bool, error) {
	path, err := ConfigPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(path); err == nil {
		return false, fmt.Errorf("config already exists at %s", path)
	}
	if err := Save(DefaultConfig()); err != nil {
		return false, err
	}
	return true, nil
}

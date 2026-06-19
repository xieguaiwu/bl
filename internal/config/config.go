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

// LLMProvider defines an OpenAI-compatible API provider for LLM-based translation.
type LLMProvider struct {
	Name    string `json:"name"`     // unique identifier, e.g. "nemotron", "bigpickle", "opencode"
	BaseURL string `json:"base_url"` // API endpoint, e.g. "https://integrate.api.nvidia.com/v1"
	Model   string `json:"model"`    // model ID, e.g. "nvidia/nemotron-3-ultra-550b-a55b"
	APIKey  string `json:"api_key"`  // literal key or "env:VAR_NAME" to reference env var
}

// LLMConfig holds settings for LLM-based translation.
type LLMConfig struct {
	Enabled      bool         `json:"enabled"`                // master switch
	Provider     string       `json:"provider"`               // name of the active provider
	Providers    []LLMProvider `json:"providers"`             // configured providers
	TargetLang   string       `json:"target_lang"`            // target language, e.g. "中文", "English", "日本語"
	SystemPrompt string       `json:"system_prompt,omitempty"` // optional custom system prompt
}

// Config holds persistent configuration for bl.
type Config struct {
	// Mode is the default dictionary query mode:
	//   "auto"    — try offline first, fall back to online (default)
	//   "offline" — offline only, error if word not in local dict
	//   "online"  — skip offline dict, always fetch from network
	Mode Mode      `json:"mode"`
	LLM  LLMConfig `json:"llm,omitempty"`
}

// DefaultLLMConfig returns the default LLM configuration with built-in provider presets.
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		Enabled:  false, // opt-in by default
		Provider: "nemotron",
		Providers: []LLMProvider{
			{
				Name:    "nemotron",
				BaseURL: "https://integrate.api.nvidia.com/v1",
				Model:   "nvidia/nemotron-3-ultra-550b-a55b",
				APIKey:  "env:NVIDIA_API_KEY",
			},
			{
				Name:    "bigpickle",
				BaseURL: "https://api.bigpickle.xyz/v1",
				Model:   "gemini-2.0-flash",
				APIKey:  "env:BIGPICKLE_API_KEY",
			},
			{
				Name:    "opencode",
				BaseURL: "https://api.opencode.com/v1",
				Model:   "opencode-translate",
				APIKey:  "env:OPENCODE_API_KEY",
			},
			{
				Name:    "custom",
				BaseURL: "",
				Model:   "",
				APIKey:  "",
			},
		},
		TargetLang: "中文",
	}
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Mode: ModeAuto,
		LLM:  DefaultLLMConfig(),
	}
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
	// Ensure LLM fields have defaults for older configs without LLM section
	if cfg.LLM.Providers == nil {
		cfg.LLM.Providers = DefaultLLMConfig().Providers
	}
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = DefaultLLMConfig().Provider
	}
	if cfg.LLM.TargetLang == "" {
		cfg.LLM.TargetLang = "中文"
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
// Returns true if a new file was created, false if it already exists.
func GenerateConfig() (bool, error) {
	path, err := ConfigPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(path); err == nil {
		return false, nil // already exists — not an error
	}
	if err := Save(DefaultConfig()); err != nil {
		return false, err
	}
	return true, nil
}

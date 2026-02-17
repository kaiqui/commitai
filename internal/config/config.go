package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigFileName = ".commitai.json"
	EnvAPIKey      = "GEMINI_API_KEY"
)

type Config struct {
	GeminiAPIKey string `json:"gemini_api_key,omitempty"`
	Language     string `json:"language"`
	CommitStyle  string `json:"commit_style"` // conventional, simple
	MaxTokens    int    `json:"max_tokens"`
	Model        string `json:"model"`
}

func DefaultConfig() *Config {
	return &Config{
		Language:    "en",
		CommitStyle: "conventional",
		MaxTokens:   1024,
		Model:       "gemini-2.0-flash",
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try home dir config
	home, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(home, ConfigFileName)
		if data, err := os.ReadFile(path); err == nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("invalid config file: %w", err)
			}
		}
	}

	// Env var overrides config file
	if key := os.Getenv(EnvAPIKey); key != "" {
		cfg.GeminiAPIKey = key
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Never save API key to disk if it came from env
	saveCfg := *cfg
	if os.Getenv(EnvAPIKey) != "" {
		saveCfg.GeminiAPIKey = ""
	}

	data, err := json.MarshalIndent(saveCfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(home, ConfigFileName), data, 0600)
}

func (c *Config) Validate() error {
	if c.GeminiAPIKey == "" {
		return errors.New("Gemini API key not set. Run: commitai config --key YOUR_KEY or set GEMINI_API_KEY env var")
	}
	return nil
}

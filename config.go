package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents user preferences read from TOML
type Config struct {
	MaxItems        int  `toml:"max_items"`
	DefaultMaxItems int  `toml:"default_max"`
	Log             bool `toml:"log"`

	Colors struct {
		Title    string `toml:"title"`
		Prompt   string `toml:"prompt"`
		Item     string `toml:"item"`
		Selected string `toml:"selected"`
		Help     string `toml:"help"`
	} `toml:"colors"`
}

// LoadConfig loads greg configuration from $XDG_CONFIG_HOME/greg/config.toml
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Determine config file path
	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home dir: %w", err)
		}
		xdgHome = filepath.Join(home, ".config")
	}

	configPath := filepath.Join(xdgHome, "greg", "config.toml")

	fmt.Printf("[DEBUG] Loading config from %s\n", configPath)

	// If the file doesn’t exist, return defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("[INFO] Config file not found at %s, using default settings.\n", configPath)
		return defaultConfig(), nil
	}

	// Decode TOML
	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return config, nil
}

// defaultConfig returns fallback settings
func defaultConfig() *Config {
	cfg := &Config{}
	cfg.MaxItems = -1
	cfg.DefaultMaxItems = 10

	// Jade-inspired default colors
	cfg.Colors.Title = "71"     // jade green (border)
	cfg.Colors.Prompt = "79"    // seafoam jade (selected-text accent)
	cfg.Colors.Item = "194"     // soft jade-tinted white (text)
	cfg.Colors.Selected = "235" // deep jade-black (background highlight)
	cfg.Colors.Help = "240"     // muted gray

	cfg.Log = false

	return cfg
}

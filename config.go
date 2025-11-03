package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
)

var CHAIN_LIMIT = 5

// Config represents user preferences read from TOML
type Config struct {
	MaxItems        int  `toml:"max_items"`
	DefaultMaxItems int  `toml:"default_max"`
	Log             bool `toml:"log"`

	File string `toml:"file"`

	Colors struct {
		Title    string `toml:"title"`
		Prompt   string `toml:"prompt"`
		Item     string `toml:"item"`
		Selected string `toml:"selected"`
		Help     string `toml:"help"`
	} `toml:"colors"`
}

// LoadConfig loads configuration from $XDG_CONFIG_HOME/greg/config.toml
// It can recursively load up to 5 config files through "file" references.
func LoadConfig() (*Config, error) {
	base := defaultConfig()

	path, err := resolveBaseConfigPath()
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG] Loading base config from %s\n", path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("[INFO] Config file not found at %s, using defaults.\n", path)
		return base, nil
	}

	if err := decodeConfigFile(path, base); err != nil {
		return nil, err
	}

	// Dynamically load chained configs (max 5)
	current := base
	for i := 0; i < CHAIN_LIMIT && current.File != ""; i++ {
		nextPath, err := resolvePath(current.File)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve file path: %w", err)
		}

		fmt.Printf("[DEBUG] Loading chained config (%d) from %s\n", i+1, nextPath)

		nextConfig := &Config{}
		if err := decodeConfigFile(nextPath, nextConfig); err != nil {
			return nil, fmt.Errorf("error parsing config file %s: %w", nextPath, err)
		}

		mergeConfig(current, nextConfig)
		current.File = nextConfig.File // follow the chain
	}

	return base, nil
}

// defaultConfig returns fallback settings
func defaultConfig() *Config {
	cfg := &Config{}
	cfg.MaxItems = -1
	cfg.DefaultMaxItems = 10

	cfg.Colors.Title = "71"
	cfg.Colors.Prompt = "79"
	cfg.Colors.Item = "194"
	cfg.Colors.Selected = "235"
	cfg.Colors.Help = "240"

	cfg.Log = false
	return cfg
}

// resolveBaseConfigPath returns the main config path.
func resolveBaseConfigPath() (string, error) {
	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home dir: %w", err)
		}
		xdgHome = filepath.Join(home, ".config")
	}
	return filepath.Join(xdgHome, "greg", "config.toml"), nil
}

// resolvePath expands ~ and returns absolute path
func resolvePath(p string) (string, error) {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[1:])
	}
	return filepath.Abs(p)
}

// decodeConfigFile decodes TOML safely into struct
func decodeConfigFile(path string, cfg *Config) error {
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return err
	}
	return nil
}

// mergeConfig merges non-zero values from src into dst recursively
func mergeConfig(dst, src *Config) {
	dstVal := reflect.ValueOf(dst).Elem()
	srcVal := reflect.ValueOf(src).Elem()
	mergeStruct(dstVal, srcVal)
}

func mergeStruct(dst, src reflect.Value) {
	for i := 0; i < dst.NumField(); i++ {
		df := dst.Field(i)
		sf := src.Field(i)

		if !sf.IsValid() || !sf.CanInterface() {
			continue
		}

		switch df.Kind() {
		case reflect.Struct:
			mergeStruct(df, sf)
		default:
			if !isZero(sf) {
				df.Set(sf)
			}
		}
	}
}

// isZero returns true if a reflect.Value is zero
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int64, reflect.Int32:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint64, reflect.Uint32:
		return v.Uint() == 0
	case reflect.Float64, reflect.Float32:
		return v.Float() == 0
	case reflect.Struct:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	default:
		return v.IsZero()
	}
}

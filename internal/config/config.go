package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Preset represents a color preset configuration
type Preset struct {
	Name       string `yaml:"name"`
	Hue        int    `yaml:"hue"`
	Saturation int    `yaml:"saturation"`
	Kelvin     int    `yaml:"kelvin"`
}

// Config represents the application configuration
type Config struct {
	Presets []Preset `yaml:"presets"`
}

// DefaultConfigPath returns the default config file path
// Uses os.UserConfigDir() for cross-platform support:
//   - macOS: ~/Library/Application Support/lifx/config.yaml
//   - Linux: ~/.config/lifx/config.yaml (or $XDG_CONFIG_HOME/lifx/config.yaml)
//   - Windows: %AppData%\lifx\config.yaml
func DefaultConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "lifx", "config.yaml")
}

// Load reads the config from the default path, returning nil if not found
func Load() (*Config, error) {
	return LoadFrom(DefaultConfigPath())
}

// LoadFrom reads the config from the specified path
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	lifxDir := filepath.Join(configDir, "lifx")
	return os.MkdirAll(lifxDir, 0755)
}

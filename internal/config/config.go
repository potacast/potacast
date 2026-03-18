package config

import (
	"os"
	"path/filepath"

	"github.com/potacast/potacast/internal/paths"
	"gopkg.in/yaml.v3"
)

// Config holds potacast configuration.
type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Ctx  int    `yaml:"ctx"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Host: "127.0.0.1",
		Port: 8080,
		Ctx:  4096,
	}
}

// Load reads the config from file, merging with defaults.
// If the file does not exist, returns default config.
func Load() (*Config, error) {
	cfg := Default()
	fpath := paths.ConfigFile()

	data, err := os.ReadFile(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for zero values
	if cfg.Host == "" {
		cfg.Host = Default().Host
	}
	if cfg.Port == 0 {
		cfg.Port = Default().Port
	}
	if cfg.Ctx == 0 {
		cfg.Ctx = Default().Ctx
	}

	return cfg, nil
}

// Save writes the config to file.
func (c *Config) Save() error {
	dir := paths.ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0644)
}

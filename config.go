package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the MCP server configuration.
type Config struct {
	ApiUrl string `json:"api_url"`
	ApiKey string `json:"api_key"`
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mcp-gerenciador", "config.json")
}

// LoadConfig reads the config from disk. Returns defaults if file doesn't exist.
func LoadConfig() *Config {
	cfg := &Config{}
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, cfg)
	return cfg
}

// Save writes the config to disk, creating directories as needed.
func (c *Config) Save() error {
	dir := filepath.Dir(ConfigPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0644)
}

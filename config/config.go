package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

type Config struct {
	Follows []string `json:"follows"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tuiFeed", "follows.json"), nil
}

func Load() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return &Config{}, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func (c *Config) AddFollow(handle string) {
	if slices.Contains(c.Follows, handle) {
		return
	}
	c.Follows = append(c.Follows, handle)
}

func (c *Config) RemoveFollow(index int) {
	if index < 0 || index >= len(c.Follows) {
		return
	}
	c.Follows = append(c.Follows[:index], c.Follows[index+1:]...)
}

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type Config struct {
	Follows    []string          `json:"follows"`
	LastChecks map[string]string `json:"last_checks"`
	SavedPosts []string          `json:"saved_posts"`
}

func (c *Config) SetLastCheck(handle string) {
	if c.LastChecks == nil {
		c.LastChecks = make(map[string]string)
	}
	c.LastChecks[handle] = time.Now().UTC().Format(time.RFC3339Nano)
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
	handle := c.Follows[index]
	c.Follows = append(c.Follows[:index], c.Follows[index+1:]...)
	if c.LastChecks != nil {
		delete(c.LastChecks, handle)
	}
}

func (c *Config) GetLastCheck(handle string) time.Time {
	if c.LastChecks == nil {
		return time.Time{}
	}
	s, ok := c.LastChecks[handle]
	if !ok {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (c *Config) SavePost(post string) {

}

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"
)

var updateMu sync.Mutex

type Config struct {
	Follows       []string          `json:"follows"`
	LastChecks    map[string]string `json:"last_checks"`
	SavedPosts    []string          `json:"saved_posts"`
	RSSFeeds      []string          `json:"rss_feeds"`
	RSSLastChecks map[string]string `json:"rss_last_checks"`
	SavedEntries  []string          `json:"saved_entries"`
}

func (c *Config) SetLastCheck(handle string) {
	if c.LastChecks == nil {
		c.LastChecks = make(map[string]string)
	}
	c.LastChecks[handle] = time.Now().UTC().Format(time.RFC3339)
}

func (c *Config) AddRSSFeed(url string) {
	if slices.Contains(c.RSSFeeds, url) {
		return
	}
	c.RSSFeeds = append(c.RSSFeeds, url)
}

func (c *Config) RemoveRSSFeed(index int) {
	if index < 0 || index >= len(c.RSSFeeds) {
		return
	}
	url := c.RSSFeeds[index]
	c.RSSFeeds = append(c.RSSFeeds[:index], c.RSSFeeds[index+1:]...)
	if c.RSSLastChecks != nil {
		delete(c.RSSLastChecks, url)
	}
}

func (c *Config) SetRSSLastCheck(url string) {
	if c.RSSLastChecks == nil {
		c.RSSLastChecks = make(map[string]string)
	}
	c.RSSLastChecks[url] = time.Now().UTC().Format(time.RFC3339)
}

func (c *Config) SaveEntry(id string) {
	if slices.Contains(c.SavedEntries, id) {
		return
	}
	c.SavedEntries = append(c.SavedEntries, id)
}

func (c *Config) RemoveSavedEntry(id string) {
	for i, e := range c.SavedEntries {
		if e == id {
			c.SavedEntries = append(c.SavedEntries[:i], c.SavedEntries[i+1:]...)
			return
		}
	}
}

func (c *Config) IsEntrySaved(id string) bool {
	return slices.Contains(c.SavedEntries, id)
}

func (c *Config) GetSavedEntryIDs() []string {
	if c.SavedEntries == nil {
		return nil
	}
	result := make([]string, len(c.SavedEntries))
	copy(result, c.SavedEntries)
	return result
}

func configPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
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
	tmp, err := os.CreateTemp(filepath.Dir(p), "tuiFeed-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, p); err != nil {
		if runtime.GOOS == "windows" {
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				return err
			}
			return os.Rename(tmpName, p)
		}
		return err
	}
	return nil
}

func Update(fn func(*Config)) error {
	updateMu.Lock()
	defer updateMu.Unlock()
	cfg, err := Load()
	if err != nil {
		return err
	}
	fn(cfg)
	return cfg.Save()
}

func ApplyUpdateAndReload(fn func(*Config)) (*Config, error) {
	if err := Update(fn); err != nil {
		return nil, err
	}
	return Load()
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

func (c *Config) SavePost(uri string) {
	if slices.Contains(c.SavedPosts, uri) {
		return
	}
	c.SavedPosts = append(c.SavedPosts, uri)
}

func (c *Config) RemoveSavedPostByURI(uri string) {
	for i, u := range c.SavedPosts {
		if u == uri {
			c.SavedPosts = append(c.SavedPosts[:i], c.SavedPosts[i+1:]...)
			return
		}
	}
}

func (c *Config) IsSaved(uri string) bool {
	return slices.Contains(c.SavedPosts, uri)
}

func (c *Config) GetSavedPostURIs() []string {
	if c.SavedPosts == nil {
		return nil
	}
	result := make([]string, len(c.SavedPosts))
	copy(result, c.SavedPosts)
	return result
}

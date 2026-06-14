package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddFollow(t *testing.T) {
	cfg := &Config{}
	cfg.AddFollow("alice.bsky.social")
	cfg.AddFollow("bob.bsky.social")
	cfg.AddFollow("alice.bsky.social")

	if len(cfg.Follows) != 2 {
		t.Fatalf("expected 2 follows, got %d", len(cfg.Follows))
	}
	if cfg.Follows[0] != "alice.bsky.social" {
		t.Errorf("expected alice.bsky.social, got %s", cfg.Follows[0])
	}
	if cfg.Follows[1] != "bob.bsky.social" {
		t.Errorf("expected bob.bsky.social, got %s", cfg.Follows[1])
	}
}

func TestRemoveFollow(t *testing.T) {
	cfg := &Config{Follows: []string{"a", "b", "c"}}
	cfg.RemoveFollow(1)
	if len(cfg.Follows) != 2 {
		t.Fatalf("expected 2 follows, got %d", len(cfg.Follows))
	}
	if cfg.Follows[0] != "a" || cfg.Follows[1] != "c" {
		t.Errorf("unexpected follows: %v", cfg.Follows)
	}
}

func TestRemoveFollowOutOfBounds(t *testing.T) {
	cfg := &Config{Follows: []string{"a"}}
	cfg.RemoveFollow(-1)
	cfg.RemoveFollow(5)
	if len(cfg.Follows) != 1 {
		t.Fatalf("expected 1 follow, got %d", len(cfg.Follows))
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	orig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	cfg := &Config{Follows: []string{"test.bsky.social"}}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	p := filepath.Join(tmp, "tuiFeed", "follows.json")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Follows) != 1 || loaded.Follows[0] != "test.bsky.social" {
		t.Errorf("unexpected loaded config: %+v", loaded)
	}
}

func TestLoadMissing(t *testing.T) {
	tmp := t.TempDir()
	orig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Follows) != 0 {
		t.Errorf("expected empty follows, got %v", cfg.Follows)
	}
}

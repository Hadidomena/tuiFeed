package config

import (
	"os"
	"path/filepath"
	"testing"
)

func withTempConfigHOME(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	orig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	t.Cleanup(func() { os.Setenv("XDG_CONFIG_HOME", orig) })
	return tmp
}

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
	tmp := withTempConfigHOME(t)

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
	withTempConfigHOME(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Follows) != 0 {
		t.Errorf("expected empty follows, got %v", cfg.Follows)
	}
}

func TestSetLastCheck(t *testing.T) {
	cfg := &Config{}
	cfg.SetLastCheck("test.bsky.social")
	if cfg.LastChecks == nil {
		t.Fatal("expected LastChecks map to be initialized")
	}
	if cfg.LastChecks["test.bsky.social"] == "" {
		t.Fatal("expected LastCheck for test.bsky.social to be set")
	}
	cfg.SetLastCheck("another.bsky.social")
	if len(cfg.LastChecks) != 2 {
		t.Errorf("expected 2 last checks, got %d", len(cfg.LastChecks))
	}
}

func TestSetLastCheck_existingMap(t *testing.T) {
	cfg := &Config{LastChecks: map[string]string{"existing": "2024-01-01T00:00:00Z"}}
	cfg.SetLastCheck("test.bsky.social")
	if cfg.LastChecks["existing"] != "2024-01-01T00:00:00Z" {
		t.Errorf("existing entry was overwritten")
	}
}

func TestRemoveFollow_withLastChecks(t *testing.T) {
	cfg := &Config{
		Follows:    []string{"a", "b", "c"},
		LastChecks: map[string]string{"a": "t1", "b": "t2", "c": "t3"},
	}
	cfg.RemoveFollow(1)
	if _, ok := cfg.LastChecks["b"]; ok {
		t.Errorf("expected last check for removed handle 'b' to be deleted")
	}
	if cfg.LastChecks["a"] != "t1" || cfg.LastChecks["c"] != "t3" {
		t.Errorf("remaining last checks are wrong")
	}
}

func TestLoad_invalidJSON(t *testing.T) {
	tmp := withTempConfigHOME(t)

	if err := os.MkdirAll(filepath.Join(tmp, "tuiFeed"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "tuiFeed", "follows.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoad_readError(t *testing.T) {
	tmp := withTempConfigHOME(t)

	if err := os.MkdirAll(filepath.Join(tmp, "tuiFeed"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(tmp, "tuiFeed", "follows.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for unreadable config")
	}
}

func TestSave_mkdirError(t *testing.T) {
	tmp := withTempConfigHOME(t)

	if err := os.WriteFile(filepath.Join(tmp, "tuiFeed"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{Follows: []string{"test.bsky.social"}}
	err := cfg.Save()
	if err == nil {
		t.Fatal("expected mkdir error when tuiFeed is a file")
	}
}

func TestSavePost(t *testing.T) {
	cfg := &Config{}
	cfg.SavePost("at://did:plc:abc/app.bsky.feed.post/1")
	cfg.SavePost("at://did:plc:abc/app.bsky.feed.post/2")

	if len(cfg.SavedPosts) != 2 {
		t.Fatalf("expected 2 saved posts, got %d", len(cfg.SavedPosts))
	}
	if cfg.SavedPosts[0] != "at://did:plc:abc/app.bsky.feed.post/1" {
		t.Errorf("unexpected first URI: %s", cfg.SavedPosts[0])
	}
}

func TestSavePost_duplicate(t *testing.T) {
	cfg := &Config{}
	cfg.SavePost("at://uri/1")
	cfg.SavePost("at://uri/1")

	if len(cfg.SavedPosts) != 1 {
		t.Fatalf("expected 1 saved post after duplicate, got %d", len(cfg.SavedPosts))
	}
}

func TestRemoveSavedPostByURI(t *testing.T) {
	cfg := &Config{SavedPosts: []string{"uri:a", "uri:b", "uri:c"}}
	cfg.RemoveSavedPostByURI("uri:b")

	if len(cfg.SavedPosts) != 2 {
		t.Fatalf("expected 2 saved posts, got %d", len(cfg.SavedPosts))
	}
	if cfg.SavedPosts[0] != "uri:a" || cfg.SavedPosts[1] != "uri:c" {
		t.Errorf("unexpected saved posts: %v", cfg.SavedPosts)
	}

	cfg.RemoveSavedPostByURI("nonexistent")
	if len(cfg.SavedPosts) != 2 {
		t.Fatalf("expected 2 saved posts, got %d", len(cfg.SavedPosts))
	}
}

func TestIsSaved(t *testing.T) {
	cfg := &Config{SavedPosts: []string{"uri:one", "uri:two"}}
	if !cfg.IsSaved("uri:one") {
		t.Error("expected uri:one to be saved")
	}
	if cfg.IsSaved("uri:three") {
		t.Error("expected uri:three not to be saved")
	}
}

func TestGetSavedPostURIs(t *testing.T) {
	cfg := &Config{SavedPosts: []string{"a", "b", "c"}}
	uris := cfg.GetSavedPostURIs()
	if len(uris) != 3 {
		t.Fatalf("expected 3 URIs, got %d", len(uris))
	}
	uris[0] = "modified"
	if cfg.SavedPosts[0] == "modified" {
		t.Error("GetSavedPostURIs should return a copy")
	}
}

func TestAddRSSFeed(t *testing.T) {
	cfg := &Config{}
	cfg.AddRSSFeed("https://example.com/feed.xml")
	cfg.AddRSSFeed("https://example.com/other.xml")
	cfg.AddRSSFeed("https://example.com/feed.xml")

	if len(cfg.RSSFeeds) != 2 {
		t.Fatalf("expected 2 feeds, got %d", len(cfg.RSSFeeds))
	}
	if cfg.RSSFeeds[0] != "https://example.com/feed.xml" {
		t.Errorf("expected feed.xml first, got %s", cfg.RSSFeeds[0])
	}
	if cfg.RSSFeeds[1] != "https://example.com/other.xml" {
		t.Errorf("expected other.xml second, got %s", cfg.RSSFeeds[1])
	}
}

func TestRemoveRSSFeed(t *testing.T) {
	cfg := &Config{RSSFeeds: []string{"a", "b", "c"}}
	cfg.RemoveRSSFeed(1)
	if len(cfg.RSSFeeds) != 2 {
		t.Fatalf("expected 2 feeds, got %d", len(cfg.RSSFeeds))
	}
	if cfg.RSSFeeds[0] != "a" || cfg.RSSFeeds[1] != "c" {
		t.Errorf("unexpected feeds: %v", cfg.RSSFeeds)
	}
}

func TestRemoveRSSFeedOutOfBounds(t *testing.T) {
	cfg := &Config{RSSFeeds: []string{"a"}}
	cfg.RemoveRSSFeed(-1)
	cfg.RemoveRSSFeed(5)
	if len(cfg.RSSFeeds) != 1 {
		t.Fatalf("expected 1 feed, got %d", len(cfg.RSSFeeds))
	}
}

func TestRemoveRSSFeed_withLastChecks(t *testing.T) {
	cfg := &Config{
		RSSFeeds:      []string{"a", "b", "c"},
		RSSLastChecks: map[string]string{"a": "t1", "b": "t2", "c": "t3"},
	}
	cfg.RemoveRSSFeed(1)
	if _, ok := cfg.RSSLastChecks["b"]; ok {
		t.Errorf("expected last check for removed feed 'b' to be deleted")
	}
	if cfg.RSSLastChecks["a"] != "t1" || cfg.RSSLastChecks["c"] != "t3" {
		t.Errorf("remaining last checks are wrong")
	}
}

func TestSetRSSLastCheck(t *testing.T) {
	cfg := &Config{}
	cfg.SetRSSLastCheck("https://example.com/feed.xml")
	if cfg.RSSLastChecks == nil {
		t.Fatal("expected RSSLastChecks map to be initialized")
	}
	if cfg.RSSLastChecks["https://example.com/feed.xml"] == "" {
		t.Fatal("expected RSSLastCheck to be set")
	}
	cfg.SetRSSLastCheck("https://example.com/other.xml")
	if len(cfg.RSSLastChecks) != 2 {
		t.Errorf("expected 2 last checks, got %d", len(cfg.RSSLastChecks))
	}
}

func TestSaveEntry(t *testing.T) {
	cfg := &Config{}
	cfg.SaveEntry("id:one")
	cfg.SaveEntry("id:two")

	if len(cfg.SavedEntries) != 2 {
		t.Fatalf("expected 2 saved entries, got %d", len(cfg.SavedEntries))
	}
	if cfg.SavedEntries[0] != "id:one" {
		t.Errorf("unexpected first entry: %s", cfg.SavedEntries[0])
	}
}

func TestSaveEntry_duplicate(t *testing.T) {
	cfg := &Config{}
	cfg.SaveEntry("id:one")
	cfg.SaveEntry("id:one")

	if len(cfg.SavedEntries) != 1 {
		t.Fatalf("expected 1 saved entry after duplicate, got %d", len(cfg.SavedEntries))
	}
}

func TestRemoveSavedEntry(t *testing.T) {
	cfg := &Config{SavedEntries: []string{"id:a", "id:b", "id:c"}}
	cfg.RemoveSavedEntry("id:b")

	if len(cfg.SavedEntries) != 2 {
		t.Fatalf("expected 2 saved entries, got %d", len(cfg.SavedEntries))
	}
	if cfg.SavedEntries[0] != "id:a" || cfg.SavedEntries[1] != "id:c" {
		t.Errorf("unexpected saved entries: %v", cfg.SavedEntries)
	}

	cfg.RemoveSavedEntry("nonexistent")
	if len(cfg.SavedEntries) != 2 {
		t.Fatalf("expected 2 saved entries, got %d", len(cfg.SavedEntries))
	}
}

func TestIsEntrySaved(t *testing.T) {
	cfg := &Config{SavedEntries: []string{"id:one", "id:two"}}
	if !cfg.IsEntrySaved("id:one") {
		t.Error("expected id:one to be saved")
	}
	if cfg.IsEntrySaved("id:three") {
		t.Error("expected id:three not to be saved")
	}
}

func TestGetSavedEntryIDs(t *testing.T) {
	cfg := &Config{SavedEntries: []string{"a", "b", "c"}}
	ids := cfg.GetSavedEntryIDs()
	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(ids))
	}
	ids[0] = "modified"
	if cfg.SavedEntries[0] == "modified" {
		t.Error("GetSavedEntryIDs should return a copy")
	}
}

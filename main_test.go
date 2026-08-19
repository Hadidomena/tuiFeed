package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/config"
	"github.com/Hadidomena/tuiFeed/feed"
	"github.com/Hadidomena/tuiFeed/internal/testutil"
	"github.com/Hadidomena/tuiFeed/rss"
	"github.com/Hadidomena/tuiFeed/rssfeed"
	"github.com/Hadidomena/tuiFeed/rssfeeds"
	"github.com/Hadidomena/tuiFeed/thread"
)

func newConfig(follows []string, checks map[string]string) *config.Config {
	return &config.Config{Follows: follows, LastChecks: checks}
}
func TestDashboardUpdate_quit(t *testing.T) {
	m := NewDashboardModel()
	_, cmd := m.Update(testutil.KeyRune('q'))
	if cmd == nil {
		t.Fatal("expected Quit command for q")
	}
}

func TestDashboardUpdate_cursorNavigation(t *testing.T) {
	t.Run("up", func(t *testing.T) {
		m := NewDashboardModel()
		m.cursor = 2
		for _, expected := range []int{1, 0, 0} {
			m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyUp))
			if m.cursor != expected {
				t.Errorf("expected cursor %d, got %d", expected, m.cursor)
			}
		}
	})
	t.Run("down", func(t *testing.T) {
		m := NewDashboardModel()
		for _, expected := range []int{1, 2, 3, 4, 5, 6, 7, 7} {
			m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyDown))
			if m.cursor != expected {
				t.Errorf("expected cursor %d, got %d", expected, m.cursor)
			}
		}
	})
}

func TestDashboardUpdate_enter(t *testing.T) {
	type want struct {
		feed, sl, mng, save, rss, rsssl, rssmng, saverss bool
	}
	tests := []struct {
		cursor int
		want   want
	}{
		{0, want{feed: true}},
		{1, want{sl: true}},
		{2, want{mng: true}},
		{3, want{save: true}},
		{4, want{rss: true}},
		{5, want{rsssl: true}},
		{6, want{rssmng: true}},
		{7, want{saverss: true}},
	}
	for _, tt := range tests {
		m := NewDashboardModel()
		m.cursor = tt.cursor
		_, cmd := m.Update(testutil.KeySpecial(tea.KeyEnter))
		if cmd == nil {
			t.Fatalf("cursor %d: expected command", tt.cursor)
		}
		msg := cmd()
		switch {
		case tt.want.feed:
			if _, ok := msg.(OpenFeedMsg); !ok {
				t.Errorf("cursor %d: expected OpenFeedMsg, got %T", tt.cursor, msg)
			}
		case tt.want.sl:
			if _, ok := msg.(OpenAccountSelectMsg); !ok {
				t.Errorf("cursor %d: expected OpenAccountSelectMsg, got %T", tt.cursor, msg)
			}
		case tt.want.mng:
			if _, ok := msg.(OpenFollowsMsg); !ok {
				t.Errorf("cursor %d: expected OpenFollowsMsg, got %T", tt.cursor, msg)
			}
		case tt.want.save:
			if _, ok := msg.(OpenSavedPostsMsg); !ok {
				t.Errorf("cursor %d: expected OpenSavedPostsMsg, got %T", tt.cursor, msg)
			}
		case tt.want.rss:
			if _, ok := msg.(OpenRSSMsg); !ok {
				t.Errorf("cursor %d: expected OpenRSSMsg, got %T", tt.cursor, msg)
			}
		case tt.want.rsssl:
			if _, ok := msg.(OpenRSSSelectMsg); !ok {
				t.Errorf("cursor %d: expected OpenRSSSelectMsg, got %T", tt.cursor, msg)
			}
		case tt.want.rssmng:
			if _, ok := msg.(OpenRSSManageMsg); !ok {
				t.Errorf("cursor %d: expected OpenRSSManageMsg, got %T", tt.cursor, msg)
			}
		case tt.want.saverss:
			if _, ok := msg.(OpenSavedRSSMsg); !ok {
				t.Errorf("cursor %d: expected OpenSavedRSSMsg, got %T", tt.cursor, msg)
			}
		}
	}
}

func TestView_nonEmpty(t *testing.T) {
	tests := []struct {
		name string
		m    tea.Model
	}{
		{"dashboard", NewDashboardModel()},
		{"accountSelect", NewAccountSelectModel(newConfig([]string{"a.bsky.social"}, map[string]string{"a.bsky.social": "2024-01-15T10:00:00Z"}))},
		{"accountSelectEmpty", NewAccountSelectModel(&config.Config{})},
		{"accountSelectNoDate", NewAccountSelectModel(newConfig([]string{"a.bsky.social"}, map[string]string{}))},
		{"loading", LoadingModel{message: "Fetching posts..."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.m.View()
			if v.Content == "" {
				t.Error("expected non-empty view")
			}
		})
	}
}

func TestAccountSelectUpdate_esc(t *testing.T) {
	m := NewAccountSelectModel(&config.Config{})
	_, cmd := m.Update(testutil.KeySpecial(tea.KeyEscape))
	if cmd == nil {
		t.Fatal("expected BackToDashboardMsg for esc")
	}
	msg := cmd()
	if _, ok := msg.(BackToDashboardMsg); !ok {
		t.Errorf("expected BackToDashboardMsg, got %T", msg)
	}
}

func TestAccountSelectUpdate_upDown(t *testing.T) {
	cfg := newConfig([]string{"a.bsky.social", "b.bsky.social", "c.bsky.social"}, nil)
	m := NewAccountSelectModel(cfg)
	m.cursor = 2
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyUp))
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("expected cursor still 2, got %d", m.cursor)
	}
}

func TestAccountSelectUpdate_enter(t *testing.T) {
	cfg := newConfig([]string{"a.bsky.social"}, nil)
	m := NewAccountSelectModel(cfg)
	_, cmd := m.Update(testutil.KeySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("expected command for enter")
	}
	msg := cmd()
	sel, ok := msg.(SelectAccountForSinceLastCheck)
	if !ok {
		t.Fatalf("expected SelectAccountForSinceLastCheck, got %T", msg)
	}
	if sel.handle != "a.bsky.social" {
		t.Errorf("expected handle 'a.bsky.social', got %q", sel.handle)
	}
}

func TestAccountSelectUpdate_enterEmpty(t *testing.T) {
	m := NewAccountSelectModel(&config.Config{})
	_, cmd := m.Update(testutil.KeySpecial(tea.KeyEnter))
	if cmd != nil {
		t.Error("expected nil command for enter on empty list")
	}
}

func TestRSSFeedSelectUpdate_esc(t *testing.T) {
	m := NewRSSFeedSelectModel(&config.Config{})
	_, cmd := m.Update(testutil.KeySpecial(tea.KeyEscape))
	if cmd == nil {
		t.Fatal("expected BackToDashboardMsg for esc")
	}
	msg := cmd()
	if _, ok := msg.(BackToDashboardMsg); !ok {
		t.Errorf("expected BackToDashboardMsg, got %T", msg)
	}
}

func TestRSSFeedSelectUpdate_upDown(t *testing.T) {
	cfg := &config.Config{RSSFeeds: []string{"https://a.com/f", "https://b.com/f", "https://c.com/f"}}
	m := NewRSSFeedSelectModel(cfg)
	m.cursor = 2
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyUp))
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("expected cursor still 2, got %d", m.cursor)
	}
}

func TestRSSFeedSelectUpdate_enter(t *testing.T) {
	cfg := &config.Config{RSSFeeds: []string{"https://a.com/f"}}
	m := NewRSSFeedSelectModel(cfg)
	_, cmd := m.Update(testutil.KeySpecial(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("expected command for enter")
	}
	msg := cmd()
	sel, ok := msg.(SelectRSSFeedForSinceLastCheck)
	if !ok {
		t.Fatalf("expected SelectRSSFeedForSinceLastCheck, got %T", msg)
	}
	if sel.url != "https://a.com/f" {
		t.Errorf("expected url 'https://a.com/f', got %q", sel.url)
	}
}

func TestRSSFeedSelectUpdate_enterEmpty(t *testing.T) {
	m := NewRSSFeedSelectModel(&config.Config{})
	_, cmd := m.Update(testutil.KeySpecial(tea.KeyEnter))
	if cmd != nil {
		t.Error("expected nil command for enter on empty list")
	}
}

func TestRSSFeedSelectView_nonEmpty(t *testing.T) {
	cfg := &config.Config{RSSFeeds: []string{"https://a.com/f"}}
	m := NewRSSFeedSelectModel(cfg)
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestLoadingUpdate_returnsSelf(t *testing.T) {
	m := LoadingModel{}
	newModel, cmd := m.Update(testutil.KeyRune('q'))
	if cmd != nil {
		t.Error("expected nil command")
	}
	if _, ok := newModel.(LoadingModel); !ok {
		t.Error("expected LoadingModel")
	}
}
func TestMainModel_OpenThreadMsg(t *testing.T) {
	m := NewMainModel()
	result, cmd := m.Update(feed.OpenThreadMsg{URI: "at://test/uri"})
	mm := result.(MainModel)

	if mm.state != showThreadView {
		t.Errorf("expected showThreadView, got %d", mm.state)
	}
	if cmd == nil {
		t.Error("expected non-nil command from thread.Init()")
	}

	v := mm.View()
	if !strings.Contains(v.Content, "Loading comments") {
		t.Errorf("expected thread loading view, got: %s", v.Content)
	}
}
func TestMainModel_OpenThreadMsg_fromDifferentStates(t *testing.T) {
	m := NewMainModel()
	mm, _ := testutil.UpdateModel(m, feed.OpenThreadMsg{URI: "at://test/uri"})
	if mm.state != showThreadView {
		t.Errorf("expected showThreadView from dashboard, got %d", mm.state)
	}

	mm, _ = testutil.UpdateModel(mm, thread.BackMsg{})
	mm, _ = testutil.UpdateModel(mm, feed.OpenThreadMsg{URI: "at://test/uri2"})
	if mm.state != showThreadView {
		t.Errorf("expected showThreadView from feed after back, got %d", mm.state)
	}
}

func TestMainModel_OpenRSSMsg(t *testing.T) {
	m := NewMainModel()
	result, cmd := m.Update(OpenRSSMsg{})
	mm := result.(MainModel)

	if mm.state != showRSSView {
		t.Errorf("expected showRSSView, got %d", mm.state)
	}
	if cmd == nil {
		t.Error("expected non-nil command from rssfeed.Init()")
	}

	v := mm.View()
	if !strings.Contains(v.Content, "Loading RSS feeds") {
		t.Errorf("expected RSS loading view, got: %s", v.Content)
	}
}

func TestMainModel_OpenRSSManageMsg(t *testing.T) {
	m := NewMainModel()
	result, cmd := m.Update(OpenRSSManageMsg{})
	mm := result.(MainModel)

	if mm.state != showRSSManageView {
		t.Errorf("expected showRSSManageView, got %d", mm.state)
	}
	if cmd != nil {
		t.Error("expected nil command from rssfeeds.Init()")
	}
}

func TestMainModel_OpenRSSSelectMsg(t *testing.T) {
	m := NewMainModel()
	result, cmd := m.Update(OpenRSSSelectMsg{})
	mm := result.(MainModel)

	if mm.state != showRSSFeedSelectView {
		t.Errorf("expected showRSSFeedSelectView, got %d", mm.state)
	}
	if cmd != nil {
		t.Error("expected nil command from rssFeedSelect.Init()")
	}
}

func TestMainModel_OpenSavedRSSMsg(t *testing.T) {
	m := NewMainModel()
	result, cmd := m.Update(OpenSavedRSSMsg{})
	mm := result.(MainModel)

	if mm.state != showSavedRSSView {
		t.Errorf("expected showSavedRSSView, got %d", mm.state)
	}
	if cmd == nil {
		t.Error("expected non-nil command from saved rssfeed.Init()")
	}
}

func TestMainModel_rssfeedBackMsg(t *testing.T) {
	m := NewMainModel()
	m.state = showRSSView
	mm, _ := testutil.UpdateModel(m, rssfeed.BackMsg{})
	if mm.state != showDashboardView {
		t.Errorf("expected showDashboardView, got %d", mm.state)
	}
}

func TestMainModel_rssfeedsBackMsg(t *testing.T) {
	m := NewMainModel()
	m.state = showRSSManageView
	mm, _ := testutil.UpdateModel(m, rssfeeds.BackMsg{})
	if mm.state != showDashboardView {
		t.Errorf("expected showDashboardView, got %d", mm.state)
	}
}

func TestMainModel_SelectRSSFeedForSinceLastCheck(t *testing.T) {
	m := NewMainModel()
	mm, cmd := testutil.UpdateModel(m, SelectRSSFeedForSinceLastCheck{url: "https://a.com/f"})
	if mm.state != showLoadingView {
		t.Errorf("expected showLoadingView, got %d", mm.state)
	}
	if cmd == nil {
		t.Error("expected non-nil fetch command")
	}
}

func TestMainModel_RSSEntriesFetchedMsg(t *testing.T) {
	m := NewMainModel()
	m.state = showLoadingView
	msg := RSSEntriesFetchedMsg{
		url:     "https://a.com/f",
		entries: []rss.Entry{{Title: "Entry", ID: "id:1"}},
	}
	mm, cmd := testutil.UpdateModel(m, msg)
	if mm.state != showRSSSinceLastCheckView {
		t.Errorf("expected showRSSSinceLastCheckView, got %d", mm.state)
	}
	if cmd == nil {
		t.Error("expected non-nil command from rssfeed.Init()")
	}
}

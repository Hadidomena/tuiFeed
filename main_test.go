package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/config"
	"github.com/Hadidomena/tuiFeed/feed"
	"github.com/Hadidomena/tuiFeed/thread"
)

func updateDashboard(m DashboardModel, msg tea.Msg) DashboardModel {
	newModel, _ := m.Update(msg)
	return newModel.(DashboardModel)
}

func TestNewDashboardModel(t *testing.T) {
	m := NewDashboardModel()
	if len(m.choices) != 4 {
		t.Errorf("expected 4 choices, got %d", len(m.choices))
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
}

func TestDashboardInit(t *testing.T) {
	m := NewDashboardModel()
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected nil command")
	}
}

func TestDashboardUpdate_quit(t *testing.T) {
	m := NewDashboardModel()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("expected Quit command for q")
	}
}

func TestDashboardUpdate_upK(t *testing.T) {
	m := NewDashboardModel()
	m.cursor = 2
	m = updateDashboard(m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m = updateDashboard(m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	m = updateDashboard(m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("expected cursor still 0, got %d", m.cursor)
	}
}

func TestDashboardUpdate_downJ(t *testing.T) {
	m := NewDashboardModel()
	m = updateDashboard(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m = updateDashboard(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", m.cursor)
	}
	m = updateDashboard(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 3 {
		t.Errorf("expected cursor 3, got %d", m.cursor)
	}
	m = updateDashboard(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 3 {
		t.Errorf("expected cursor still 3, got %d", m.cursor)
	}
}

func TestDashboardUpdate_enter(t *testing.T) {
	tests := []struct {
		cursor   int
		wantFeed bool
		wantSL   bool
		wantMng  bool
		wantSave bool
	}{
		{0, true, false, false, false},
		{1, false, true, false, false},
		{2, false, false, true, false},
		{3, false, false, false, true},
	}
	for _, tt := range tests {
		m := NewDashboardModel()
		m.cursor = tt.cursor
		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Fatalf("cursor %d: expected command", tt.cursor)
		}
		msg := cmd()
		switch {
		case tt.wantFeed:
			if _, ok := msg.(OpenFeedMsg); !ok {
				t.Errorf("cursor %d: expected OpenFeedMsg, got %T", tt.cursor, msg)
			}
		case tt.wantSL:
			if _, ok := msg.(OpenAccountSelectMsg); !ok {
				t.Errorf("cursor %d: expected OpenAccountSelectMsg, got %T", tt.cursor, msg)
			}
		case tt.wantMng:
			if _, ok := msg.(OpenFollowsMsg); !ok {
				t.Errorf("cursor %d: expected OpenFollowsMsg, got %T", tt.cursor, msg)
			}
		case tt.wantSave:
			if _, ok := msg.(OpenSavedPostsMsg); !ok {
				t.Errorf("cursor %d: expected OpenSavedPostsMsg, got %T", tt.cursor, msg)
			}
		}
	}
}

func TestDashboardView(t *testing.T) {
	m := NewDashboardModel()
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestNewAccountSelectModel(t *testing.T) {
	cfg := &config.Config{
		Follows:    []string{"a.bsky.social", "b.bsky.social"},
		LastChecks: map[string]string{"a.bsky.social": "2024-01-15T10:00:00Z"},
	}
	m := NewAccountSelectModel(cfg)
	if len(m.accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(m.accounts))
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
}

func TestAccountSelectInit(t *testing.T) {
	cfg := &config.Config{}
	m := NewAccountSelectModel(cfg)
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected nil command")
	}
}

func updateAccountSelect(m AccountSelectModel, msg tea.Msg) AccountSelectModel {
	newModel, _ := m.Update(msg)
	return newModel.(AccountSelectModel)
}

func TestAccountSelectUpdate_esc(t *testing.T) {
	cfg := &config.Config{}
	m := NewAccountSelectModel(cfg)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected BackToDashboardMsg for esc")
	}
	msg := cmd()
	if _, ok := msg.(BackToDashboardMsg); !ok {
		t.Errorf("expected BackToDashboardMsg, got %T", msg)
	}
}

func TestAccountSelectUpdate_upDown(t *testing.T) {
	cfg := &config.Config{Follows: []string{"a.bsky.social", "b.bsky.social", "c.bsky.social"}}
	m := NewAccountSelectModel(cfg)
	m.cursor = 2
	m = updateAccountSelect(m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m = updateAccountSelect(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", m.cursor)
	}
	m = updateAccountSelect(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor still 2, got %d", m.cursor)
	}
}

func TestAccountSelectUpdate_enter(t *testing.T) {
	cfg := &config.Config{Follows: []string{"a.bsky.social"}}
	m := NewAccountSelectModel(cfg)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
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
	cfg := &config.Config{}
	m := NewAccountSelectModel(cfg)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil command for enter on empty list")
	}
}

func TestAccountSelectView(t *testing.T) {
	cfg := &config.Config{
		Follows:    []string{"a.bsky.social"},
		LastChecks: map[string]string{"a.bsky.social": "2024-01-15T10:00:00Z"},
	}
	m := NewAccountSelectModel(cfg)
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestAccountSelectView_empty(t *testing.T) {
	cfg := &config.Config{}
	m := NewAccountSelectModel(cfg)
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestAccountSelectView_noDate(t *testing.T) {
	cfg := &config.Config{
		Follows:    []string{"a.bsky.social"},
		LastChecks: map[string]string{},
	}
	m := NewAccountSelectModel(cfg)
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestLoadingInit(t *testing.T) {
	m := LoadingModel{}
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected nil command")
	}
}

func TestLoadingUpdate(t *testing.T) {
	m := LoadingModel{}
	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd != nil {
		t.Error("expected nil command")
	}
	if _, ok := newModel.(LoadingModel); !ok {
		t.Error("expected LoadingModel")
	}
}

func TestLoadingUpdate_nonKey(t *testing.T) {
	m := LoadingModel{}
	newModel, cmd := m.Update(struct{ tea.Msg }{})
	if cmd != nil {
		t.Error("expected nil command for non-key message")
	}
	if _, ok := newModel.(LoadingModel); !ok {
		t.Error("expected LoadingModel")
	}
}

func TestLoadingView(t *testing.T) {
	m := LoadingModel{message: "Fetching posts..."}
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestUpdateSubModel(t *testing.T) {
	d := NewDashboardModel()
	_, cmd := updateSubModel(d, tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("expected command from updateSubModel")
	}
}

func updateMainModel(m MainModel, msg tea.Msg) MainModel {
	r, _ := m.Update(msg)
	return r.(MainModel)
}

func TestMainModel_OpenThreadMsg_transitions(t *testing.T) {
	m := NewMainModel()
	result, cmd := m.Update(feed.OpenThreadMsg{URI: "at://test/uri"})
	mm := result.(MainModel)

	if mm.state != showThreadView {
		t.Errorf("expected showThreadView, got %d", mm.state)
	}
	if cmd == nil {
		t.Error("expected non-nil command from thread.Init()")
	}
}

func TestMainModel_OpenThreadMsg_showsLoadingView(t *testing.T) {
	m := NewMainModel()
	result, _ := m.Update(feed.OpenThreadMsg{URI: "at://test/uri"})
	mm := result.(MainModel)

	v := mm.View()
	if !strings.Contains(v.Content, "Loading comments") {
		t.Errorf("expected thread loading view, got: %s", v.Content)
	}
}

func TestMainModel_ThreadBackMsg_returnsToFeed(t *testing.T) {
	m := NewMainModel()
	mm := updateMainModel(m, feed.OpenThreadMsg{URI: "at://test/uri"})

	mm = updateMainModel(mm, thread.BackMsg{})
	if mm.state != showFeedView {
		t.Errorf("expected showFeedView after back, got %d", mm.state)
	}
}

func TestMainModel_OpenThreadMsg_viewDispatch(t *testing.T) {
	m := NewMainModel()
	mm := updateMainModel(m, feed.OpenThreadMsg{URI: "at://test/uri"})

	v := mm.View()
	if !strings.Contains(v.Content, "Comments") && !strings.Contains(v.Content, "Loading comments") {
		t.Errorf("expected thread-related view content, got: %s", v.Content)
	}
}

func TestMainModel_OpenThreadMsg_fromDifferentStates(t *testing.T) {
	m := NewMainModel()
	// Try from dashboard (initial state)
	mm := updateMainModel(m, feed.OpenThreadMsg{URI: "at://test/uri"})
	if mm.state != showThreadView {
		t.Errorf("expected showThreadView from dashboard, got %d", mm.state)
	}

	// Go back
	mm = updateMainModel(mm, thread.BackMsg{})

	// Try from feed state
	mm = updateMainModel(mm, feed.OpenThreadMsg{URI: "at://test/uri2"})
	if mm.state != showThreadView {
		t.Errorf("expected showThreadView from feed, got %d", mm.state)
	}
}

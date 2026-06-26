package follows

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/config"
)

func setupTestConfig(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Follows = []string{"alice.bsky.social", "bob.bsky.social"}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
}

func updateFollows(m Model, msg tea.Msg) (Model, *tea.Cmd) {
	newModel, cmd := m.Update(msg)
	result := newModel.(Model)
	if cmd != nil {
		return result, &cmd
	}
	return result, nil
}

func TestInit(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestUpdate_nonKeyMsg(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	_, cmd := m.Update(struct{ tea.Msg }{})
	if cmd != nil {
		t.Error("expected nil command for non-key message")
	}
}

func TestUpdateList_esc(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected BackMsg command for esc")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("expected BackMsg, got %T", msg)
	}
}

func TestUpdateList_upK(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.cursor = 1
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("expected cursor still 0 (at start), got %d", m.cursor)
	}
}

func TestUpdateList_downJ(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: 'j'})
	if m.cursor != 1 {
		t.Errorf("expected cursor still 1 (at end), got %d", m.cursor)
	}
}

func TestUpdateList_a(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: 'a'})
	if m.mode != modeInput {
		t.Errorf("expected modeInput, got %v", m.mode)
	}
	if m.input != "" {
		t.Errorf("expected empty input, got %q", m.input)
	}
}

func TestUpdateList_d(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: 'd'})
	if len(m.cfg.Follows) != 1 {
		t.Errorf("expected 1 follow after delete, got %d", len(m.cfg.Follows))
	}
	if m.statusMsg == "" {
		t.Error("expected status message after delete")
	}
}

func TestUpdateList_dAtEnd(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.cursor = 1
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: 'd'})
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 after deleting last, got %d", m.cursor)
	}
	if len(m.cfg.Follows) != 1 {
		t.Errorf("expected 1 follow, got %d", len(m.cfg.Follows))
	}
}

func TestUpdateList_dEmpty(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)
	cfg := &config.Config{}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	m := Model{cfg: cfg, cursor: 0}
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: 'd'})
	if m.statusMsg != "" {
		t.Errorf("expected no status message for empty list, got %q", m.statusMsg)
	}
}

func TestUpdateInput_esc(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.mode = modeInput
	m.input = "hello"
	m.statusMsg = "old"
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeList {
		t.Errorf("expected modeList, got %v", m.mode)
	}
	if m.input != "" {
		t.Errorf("expected empty input, got %q", m.input)
	}
	if m.statusMsg != "" {
		t.Errorf("expected empty status, got %q", m.statusMsg)
	}
}

func TestUpdateInput_enterEmpty(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.mode = modeInput
	m.input = "   "
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeList {
		t.Errorf("expected modeList after empty enter, got %v", m.mode)
	}
	if m.statusMsg != "Handle cannot be empty" {
		t.Errorf("expected empty handle warning, got %q", m.statusMsg)
	}
}

func TestUpdateInput_enterValid(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.mode = modeInput
	m.input = "@testuser.bsky.social"
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.mode != modeList {
		t.Errorf("expected modeList, got %v", m.mode)
	}
	if m.input != "" {
		t.Errorf("expected empty input, got %q", m.input)
	}
	if m.cursor != len(m.cfg.Follows)-1 {
		t.Errorf("expected cursor at last position, got %d", m.cursor)
	}
}

func TestUpdateInput_backspace(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.mode = modeInput
	m.input = "hello"
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.input != "hell" {
		t.Errorf("expected 'hell' after backspace, got %q", m.input)
	}
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.input != "" {
		t.Errorf("expected empty input, got %q", m.input)
	}
	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.input != "" {
		t.Errorf("expected still empty input, got %q", m.input)
	}
}

func TestUpdateInput_text(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.mode = modeInput
	m, _ = updateFollows(m, tea.KeyPressMsg{Text: "h"})
	if m.input != "h" {
		t.Errorf("expected 'h', got %q", m.input)
	}
	m, _ = updateFollows(m, tea.KeyPressMsg{Text: "i"})
	if m.input != "hi" {
		t.Errorf("expected 'hi', got %q", m.input)
	}
}

func TestView_listMode(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_listModeWithCursor(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.cursor = 1
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_inputMode(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.mode = modeInput
	m.input = "test"
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_emptyList(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)
	cfg := &config.Config{}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	m := Model{cfg: cfg}
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_withStatus(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.statusMsg = "Something happened"
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestNewModel_loadError(t *testing.T) {
	orig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", "/dev/null/nonexistent")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	_, err := NewModel()
	if err != nil {
		return
	}
	t.Fatal("expected error, got nil")
}

func TestUpdateList_dSaveError(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)
	if err := os.MkdirAll(tmp+"/tuiFeed", 0o000); err != nil {
		t.Fatal(err)
	}

	m, _ = updateFollows(m, tea.KeyPressMsg{Code: 'd'})
	if len(m.cfg.Follows) != 1 {
		t.Fatalf("expected 1 follow, got %d", len(m.cfg.Follows))
	}
}

func TestUpdateInput_enterSaveError(t *testing.T) {
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	m.mode = modeInput
	m.input = "newuser.bsky.social"

	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)
	if err := os.MkdirAll(tmp+"/tuiFeed", 0o000); err != nil {
		t.Fatal(err)
	}

	m, _ = updateFollows(m, tea.KeyPressMsg{Code: tea.KeyEnter})
}

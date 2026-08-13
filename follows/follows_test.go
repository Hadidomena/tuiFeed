package follows

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/config"
	"github.com/Hadidomena/tuiFeed/internal/testutil"
)

func setupTestConfig(t *testing.T) {
	t.Helper()
	cfg := testutil.SetupTestConfig(t)
	cfg.Follows = []string{"alice.bsky.social", "bob.bsky.social"}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
}

func setupModel(t *testing.T) Model {
	t.Helper()
	setupTestConfig(t)
	m, err := NewModel()
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	return m
}
func TestUpdate_nonKeyMsg(t *testing.T) {
	m := setupModel(t)
	_, cmd := m.Update(struct{ tea.Msg }{})
	if cmd != nil {
		t.Error("expected nil command for non-key message")
	}
}

func TestUpdateList_esc(t *testing.T) {
	m := setupModel(t)
	_, cmd := m.Update(testutil.KeySpecial(tea.KeyEscape))
	if cmd == nil {
		t.Fatal("expected BackMsg command for esc")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("expected BackMsg, got %T", msg)
	}
}

func TestUpdateList_upK(t *testing.T) {
	m := setupModel(t)
	m.cursor = 1
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyUp))
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyUp))
	if m.cursor != 0 {
		t.Errorf("expected cursor still 0 (at start), got %d", m.cursor)
	}
}

func TestUpdateList_downJ(t *testing.T) {
	m := setupModel(t)
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyDown))
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('j'))
	if m.cursor != 1 {
		t.Errorf("expected cursor still 1 (at end), got %d", m.cursor)
	}
}

func TestUpdateList_a(t *testing.T) {
	m := setupModel(t)
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('a'))
	if m.mode != modeInput {
		t.Errorf("expected modeInput, got %v", m.mode)
	}
	if m.input != "" {
		t.Errorf("expected empty input, got %q", m.input)
	}
}

func TestUpdateList_d(t *testing.T) {
	m := setupModel(t)
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('d'))
	if len(m.cfg.Follows) != 1 {
		t.Errorf("expected 1 follow after delete, got %d", len(m.cfg.Follows))
	}
	if m.statusMsg == "" {
		t.Error("expected status message after delete")
	}
}

func TestUpdateList_dAtEnd(t *testing.T) {
	m := setupModel(t)
	m.cursor = 1
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('d'))
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
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('d'))
	if m.statusMsg != "" {
		t.Errorf("expected no status message for empty list, got %q", m.statusMsg)
	}
}

func TestUpdateInput_esc(t *testing.T) {
	m := setupModel(t)
	m.mode = modeInput
	m.input = "hello"
	m.statusMsg = "old"
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyEscape))
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
	m := setupModel(t)
	m.mode = modeInput
	m.input = "   "
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyEnter))
	if m.mode != modeList {
		t.Errorf("expected modeList after empty enter, got %v", m.mode)
	}
	if m.statusMsg != "Handle cannot be empty" {
		t.Errorf("expected empty handle warning, got %q", m.statusMsg)
	}
}

func TestUpdateInput_enterValid(t *testing.T) {
	m := setupModel(t)
	m.mode = modeInput
	m.input = "@testuser.bsky.social"
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyEnter))
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
	m := setupModel(t)
	m.mode = modeInput
	m.input = "hello"
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyBackspace))
	if m.input != "hell" {
		t.Errorf("expected 'hell' after backspace, got %q", m.input)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyBackspace))
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyBackspace))
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyBackspace))
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyBackspace))
	if m.input != "" {
		t.Errorf("expected empty input, got %q", m.input)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyBackspace))
	if m.input != "" {
		t.Errorf("expected still empty input, got %q", m.input)
	}
}

func TestUpdateInput_text(t *testing.T) {
	m := setupModel(t)
	m.mode = modeInput
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('h'))
	if m.input != "h" {
		t.Errorf("expected 'h', got %q", m.input)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('i'))
	if m.input != "hi" {
		t.Errorf("expected 'hi', got %q", m.input)
	}
}

func TestView_listMode(t *testing.T) {
	m := setupModel(t)
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_listModeWithCursor(t *testing.T) {
	m := setupModel(t)
	m.cursor = 1
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_inputMode(t *testing.T) {
	m := setupModel(t)
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
	m := setupModel(t)
	m.statusMsg = "Something happened"
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestNewModel_loadError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	if err := os.MkdirAll(filepath.Join(tmp, "tuiFeed", "follows.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := NewModel()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateList_dSaveError(t *testing.T) {
	m := setupModel(t)
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := os.WriteFile(filepath.Join(tmp, "tuiFeed"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	m, _ = testutil.UpdateModel(m, testutil.KeyRune('d'))
	if len(m.cfg.Follows) != 2 {
		t.Fatalf("expected 2 follows (save failed, state unchanged), got %d", len(m.cfg.Follows))
	}
}

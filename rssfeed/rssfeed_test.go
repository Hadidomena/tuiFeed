package rssfeed

import (
	"net/http"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/attach"
	"github.com/Hadidomena/tuiFeed/config"
	"github.com/Hadidomena/tuiFeed/internal/testutil"
	"github.com/Hadidomena/tuiFeed/rss"
)

func bareEntry() []rss.Entry {
	return []rss.Entry{{Title: "Entry One", ID: "id:1"}}
}

func entryWithID() []rss.Entry {
	return []rss.Entry{{
		Title: "Entry",
		Link:  "https://example.com/e",
		ID:    "id:1",
		Text:  "Hello",
	}}
}

func entryWithTwoImages() []rss.Entry {
	return []rss.Entry{{ID: "id:1", Images: []string{"a.jpg", "b.jpg"}}}
}

func entryWithOneImage() []rss.Entry {
	return []rss.Entry{{ID: "id:1", Images: []string{"a.jpg"}}}
}

func nEntries(n int) []rss.Entry {
	entries := make([]rss.Entry, n)
	for i := range entries {
		entries[i] = rss.Entry{Title: "Entry", ID: "id", Text: "Hello"}
	}
	return entries
}

func TestNewStaticModel(t *testing.T) {
	entries := bareEntry()
	m := NewStaticModel(entries, "My Title")
	if m.title != "My Title" {
		t.Errorf("expected 'My Title', got %q", m.title)
	}
	if len(m.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m.entries))
	}
	if m.pageSize != 10 {
		t.Errorf("expected pageSize 10, got %d", m.pageSize)
	}
}

func TestInit_savedView(t *testing.T) {
	m := NewSavedModel(&config.Config{})
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected loadSavedEntries command for saved view")
	}
}

func TestUpdate_quit(t *testing.T) {
	m := NewStaticModel(bareEntry(), "Test")
	_, cmd := m.Update(testutil.KeyRune('q'))
	if cmd == nil {
		t.Fatal("expected Quit command for 'q'")
	}
}

func TestUpdate_ctrlC(t *testing.T) {
	m := NewStaticModel(bareEntry(), "Test")
	msg := testutil.KeyPress("", 'c', tea.ModCtrl)
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected Quit command for ctrl+c")
	}
}

func TestUpdate_esc(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	_, cmd := m.Update(testutil.KeySpecial(tea.KeyEscape))
	if cmd == nil {
		t.Fatal("expected BackMsg command for esc")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("expected BackMsg, got %T", msg)
	}
}

func TestUpdate_downJ(t *testing.T) {
	m := NewStaticModel(nEntries(3), "Test")
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyDown))
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('j'))
	if m.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("expected cursor still 2 (at end), got %d", m.cursor)
	}
}

func TestUpdate_upK(t *testing.T) {
	m := NewStaticModel(nEntries(3), "Test")
	m.cursor = 2
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyUp))
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('k'))
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyUp))
	if m.cursor != 0 {
		t.Errorf("expected cursor still 0 (at start), got %d", m.cursor)
	}
}

func TestUpdate_scrollWindow(t *testing.T) {
	m := NewStaticModel(nEntries(25), "Test")
	m.cursor = 14
	m.scrollPos = 4
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyDown))
	if m.cursor != 15 {
		t.Errorf("expected cursor 15, got %d", m.cursor)
	}
	if m.scrollPos != 5 {
		t.Errorf("expected scrollPos 5 (advanced), got %d", m.scrollPos)
	}
	m.cursor = 5
	m.scrollPos = 5
	m, _ = testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyUp))
	if m.scrollPos != 4 {
		t.Errorf("expected scrollPos 4 (retreated), got %d", m.scrollPos)
	}
}

func TestUpdate_left(t *testing.T) {
	m := NewStaticModel(entryWithTwoImages(), "Test")
	m.hasRendered = true
	m.imgCursor = 1
	m, cmd := testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyLeft))
	if m.imgCursor != 0 {
		t.Errorf("expected imgCursor 0, got %d", m.imgCursor)
	}
	if cmd == nil {
		t.Error("expected render attachment command")
	}
}

func TestUpdate_leftAtZero(t *testing.T) {
	m := NewStaticModel(entryWithTwoImages(), "Test")
	m.hasRendered = true
	m.imgCursor = 0
	_, cmd := testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyLeft))
	if cmd != nil {
		t.Error("expected no command when imgCursor at 0")
	}
}

func TestUpdate_right(t *testing.T) {
	m := NewStaticModel(entryWithTwoImages(), "Test")
	m.hasRendered = true
	m.imgCursor = 0
	m, cmd := testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyRight))
	if m.imgCursor != 1 {
		t.Errorf("expected imgCursor 1, got %d", m.imgCursor)
	}
	if cmd == nil {
		t.Error("expected render attachment command")
	}
}

func TestUpdate_rightAtEnd(t *testing.T) {
	m := NewStaticModel(entryWithTwoImages(), "Test")
	m.hasRendered = true
	m.imgCursor = 1
	_, cmd := testutil.UpdateModel(m, testutil.KeySpecial(tea.KeyRight))
	if cmd != nil {
		t.Error("expected no command when imgCursor at end")
	}
}

func TestUpdate_oKey(t *testing.T) {
	m := NewStaticModel(entryWithOneImage(), "Test")
	m, cmd := testutil.UpdateModel(m, testutil.KeyRune('o'))
	if cmd == nil {
		t.Error("expected open attachment command")
	}
	if m.statusMsg != "Opening image externally..." {
		t.Errorf("expected status, got %q", m.statusMsg)
	}
}

func TestUpdate_oKeyNoImages(t *testing.T) {
	m := NewStaticModel(bareEntry(), "Test")
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('o'))
	if m.statusMsg == "Opening image externally..." {
		t.Error("expected no status when no images")
	}
}

func TestUpdate_aKey(t *testing.T) {
	m := NewStaticModel(entryWithOneImage(), "Test")
	m.height = 40
	m, cmd := testutil.UpdateModel(m, testutil.KeyRune('a'))
	if !m.hasRendered {
		t.Error("expected hasRendered to be true")
	}
	if cmd == nil {
		t.Error("expected render attachment command")
	}
}

func TestUpdate_rKey(t *testing.T) {
	m := NewStaticModel(nEntries(1), "Test")
	m.hasRendered = true
	m.imageRows = 10
	m.scrollPos = 5
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('r'))
	if !m.loading {
		t.Error("expected loading to be true")
	}
	if m.hasRendered {
		t.Error("expected hasRendered to be false")
	}
	if m.imageRows != 0 {
		t.Errorf("expected imageRows 0, got %d", m.imageRows)
	}
	if m.scrollPos != 0 {
		t.Errorf("expected scrollPos 0, got %d", m.scrollPos)
	}
}

func TestUpdate_rKey_savedView(t *testing.T) {
	m := NewStaticModel(bareEntry(), "Test")
	m.isSavedView = true
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('r'))
	if m.loading {
		t.Error("expected loading to stay false in saved view")
	}
}

func TestUpdate_entriesLoadedMsg(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m.loading = true
	m.cursor = 5
	msg := entriesLoadedMsg{entries: []rss.Entry{
		{Title: "a"},
		{Title: "b"},
	}}
	m, cmd := testutil.UpdateModel(m, msg)
	if cmd != nil {
		t.Error("expected no command")
	}
	if m.loading {
		t.Error("expected loading false")
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	if len(m.entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(m.entries))
	}
}

func TestUpdate_entriesLoadedMsg_empty(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m.loading = true
	m, _ = testutil.UpdateModel(m, entriesLoadedMsg{})
	if m.statusMsg != "No entries found." {
		t.Errorf("expected 'No entries found.', got %q", m.statusMsg)
	}
}

func TestUpdate_loadErrorMsg(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m.loading = true
	m, _ = testutil.UpdateModel(m, loadErrorMsg("something went wrong"))
	if m.loading {
		t.Error("expected loading false")
	}
	if m.statusMsg != "something went wrong" {
		t.Errorf("expected error message, got %q", m.statusMsg)
	}
}

func TestUpdate_imageRenderedMsg(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m, _ = testutil.UpdateModel(m, attach.RenderedMsg{ImageRows: 15})
	if m.imageRows != 15 {
		t.Errorf("expected imageRows 15, got %d", m.imageRows)
	}
}

func TestView_loading(t *testing.T) {
	m := NewStaticModel(nil, "Test Feed")
	m.loading = true
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_noEntries(t *testing.T) {
	m := NewStaticModel(nil, "Test Feed")
	m.loading = false
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_withEntries(t *testing.T) {
	m := NewStaticModel(entryWithID(), "My Feed")
	m.loading = false
	m.cursor = 0
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_cursorPastEnd(t *testing.T) {
	m := NewStaticModel(bareEntry(), "Feed")
	m.loading = false
	m.cursor = 5
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_paginationAbove(t *testing.T) {
	m := NewStaticModel(nEntries(20), "Feed")
	m.loading = false
	m.cursor = 12
	m.scrollPos = 5
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_paginationBelow(t *testing.T) {
	m := NewStaticModel(nEntries(20), "Feed")
	m.loading = false
	m.cursor = 0
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_renderedWithImages(t *testing.T) {
	m := NewStaticModel(entryWithTwoImages(), "Feed")
	m.loading = false
	m.cursor = 0
	m.hasRendered = true
	m.imageRows = 5
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_withStatusMessage(t *testing.T) {
	m := NewStaticModel(bareEntry(), "Feed")
	m.loading = false
	m.statusMsg = "Some status"
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_savedView(t *testing.T) {
	m := NewStaticModel(bareEntry(), "Saved RSS entries")
	m.loading = false
	m.cursor = 0
	m.isSavedView = true
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(v.Content, "[s] remove") {
		t.Errorf("expected '[s] remove' in saved view help bar, got: %s", v.Content)
	}
}

func TestView_helpBarShowsOpenLink(t *testing.T) {
	m := NewStaticModel(bareEntry(), "Feed")
	m.loading = false
	m.cursor = 0
	v := m.View()
	if !strings.Contains(v.Content, "[w] open link") {
		t.Errorf("expected '[w] open link' in help bar, got: %s", v.Content)
	}
}

func TestRenderAttachment_oob(t *testing.T) {
	m := NewStaticModel(entryWithOneImage(), "Test")
	m.imgCursor = 5
	msg := m.renderAttachment()
	errMsg, ok := msg.(attach.ErrorMsg)
	if !ok {
		t.Fatalf("expected attach.ErrorMsg, got %T", msg)
	}
	if string(errMsg) != "No image to render" {
		t.Errorf("expected 'No image to render', got %q", string(errMsg))
	}
}

func TestOpenAttachment_oob(t *testing.T) {
	m := NewStaticModel(entryWithOneImage(), "Test")
	m.imgCursor = -1
	msg := m.openAttachment()
	errMsg, ok := msg.(attach.ErrorMsg)
	if !ok {
		t.Fatalf("expected attach.ErrorMsg, got %T", msg)
	}
	if string(errMsg) != "No image to open" {
		t.Errorf("expected 'No image to open', got %q", string(errMsg))
	}
}

func TestUpdate_wKey_openLink(t *testing.T) {
	m := NewStaticModel(entryWithID(), "Test")
	m.cursor = 0
	_, cmd := m.Update(testutil.KeyRune('w'))
	if cmd == nil {
		t.Fatal("expected command for w key")
	}
}

func TestUpdate_wKey_noLink(t *testing.T) {
	m := NewStaticModel(bareEntry(), "Test")
	m.cursor = 0
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('w'))
	if m.statusMsg != "Entry has no link" {
		t.Errorf("expected 'Entry has no link', got %q", m.statusMsg)
	}
}

func TestUpdate_sKey_save(t *testing.T) {
	cfg := testutil.SetupTestConfig(t)
	m := NewStaticModel(entryWithID(), "Test")
	m.cfg = cfg
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('s'))

	if m.statusMsg != "Saved!" {
		t.Errorf("expected status 'Saved!', got %q", m.statusMsg)
	}
	saved, _ := config.Load()
	if !saved.IsEntrySaved("id:1") {
		t.Error("expected entry to be saved")
	}
}

func TestUpdate_sKey_unsave(t *testing.T) {
	cfg := testutil.SetupTestConfig(t)
	cfg.SaveEntry("id:1")
	_ = cfg.Save()
	m := NewStaticModel(entryWithID(), "Test")
	m.cfg = cfg
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('s'))

	if m.statusMsg != "Unsaved" {
		t.Errorf("expected status 'Unsaved', got %q", m.statusMsg)
	}
	saved, _ := config.Load()
	if saved.IsEntrySaved("id:1") {
		t.Error("expected entry to be unsaved")
	}
}

func TestUpdate_sKey_noConfig(t *testing.T) {
	m := NewStaticModel(entryWithID(), "Test")
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('s'))
	if m.statusMsg != "" {
		t.Errorf("expected no status without config, got %q", m.statusMsg)
	}
}

func TestUpdate_sKey_noID(t *testing.T) {
	m := NewStaticModel([]rss.Entry{{Title: "No id"}}, "Test")
	m.cfg = &config.Config{}
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('s'))
	if m.statusMsg != "Cannot save entry (no ID)" {
		t.Errorf("expected 'Cannot save entry (no ID)', got %q", m.statusMsg)
	}
}

func TestUpdate_sKey_removeFromSaved(t *testing.T) {
	cfg := testutil.SetupTestConfig(t)
	cfg.SaveEntry("id:1")
	_ = cfg.Save()
	m := NewStaticModel(entryWithID(), "Test")
	m.cfg = cfg
	m.isSavedView = true
	m, _ = testutil.UpdateModel(m, testutil.KeyRune('s'))

	if m.statusMsg != "No saved entries" {
		t.Errorf("expected removal status, got %q", m.statusMsg)
	}
	if len(m.entries) != 0 {
		t.Errorf("expected 0 entries after removal, got %d", len(m.entries))
	}
	saved, _ := config.Load()
	if saved.IsEntrySaved("id:1") {
		t.Error("expected entry to be removed from saved")
	}
}

func TestLoadEntries_noConfig(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	msg := m.loadEntries()
	errMsg, ok := msg.(loadErrorMsg)
	if !ok {
		t.Fatalf("expected loadErrorMsg, got %T", msg)
	}
	if string(errMsg) != "Not available in this view." {
		t.Errorf("expected 'Not available in this view.', got %q", string(errMsg))
	}
}

func TestLoadEntries_noFeeds(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m.cfg = &config.Config{}
	msg := m.loadEntries()
	errMsg, ok := msg.(loadErrorMsg)
	if !ok {
		t.Fatalf("expected loadErrorMsg, got %T", msg)
	}
	if string(errMsg) != "No RSS feeds subscribed. Add some in Manage RSS feeds." {
		t.Errorf("expected no feeds message, got %q", string(errMsg))
	}
}

func TestLoadEntries_fetch(t *testing.T) {
	const body = `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Hello</title>
      <link>https://example.com/1</link>
      <guid>https://example.com/1</guid>
    </item>
  </channel>
</rss>`
	server := testutil.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	cfg := &config.Config{}
	cfg.AddRSSFeed(server.URL)
	m := NewStaticModel(nil, "Test")
	m.cfg = cfg
	msg := m.loadEntries()
	pl, ok := msg.(entriesLoadedMsg)
	if !ok {
		t.Fatalf("expected entriesLoadedMsg, got %T", msg)
	}
	if len(pl.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(pl.entries))
	}
	if pl.entries[0].Title != "Hello" {
		t.Errorf("expected 'Hello', got %q", pl.entries[0].Title)
	}
}

func TestLoadSavedEntries_empty(t *testing.T) {
	cfg := &config.Config{}
	m := NewSavedModel(cfg)
	msg := m.loadSavedEntries()
	pl, ok := msg.(entriesLoadedMsg)
	if !ok {
		t.Fatalf("expected entriesLoadedMsg, got %T", msg)
	}
	if len(pl.entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(pl.entries))
	}
}

func TestLoadSavedEntries_noConfig(t *testing.T) {
	m := NewSavedModel(nil)
	msg := m.loadSavedEntries()
	errMsg, ok := msg.(loadErrorMsg)
	if !ok {
		t.Fatalf("expected loadErrorMsg, got %T", msg)
	}
	if string(errMsg) != "Not available in this view." {
		t.Errorf("expected 'Not available in this view.', got %q", string(errMsg))
	}
}

func TestLoadSavedEntries_partialFailureWarns(t *testing.T) {
	const body = `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Hello</title>
      <link>https://example.com/1</link>
      <guid>https://example.com/1</guid>
    </item>
  </channel>
</rss>`
	good := testutil.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer good.Close()
	bad := testutil.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "oops", http.StatusInternalServerError)
	}))
	defer bad.Close()

	cfg := &config.Config{}
	cfg.AddRSSFeed(good.URL)
	cfg.AddRSSFeed(bad.URL)
	cfg.SaveEntry("https://example.com/1")

	m := NewSavedModel(cfg)
	msg := m.loadSavedEntries()
	pl, ok := msg.(entriesLoadedMsg)
	if !ok {
		t.Fatalf("expected entriesLoadedMsg, got %T", msg)
	}
	if len(pl.entries) != 1 {
		t.Errorf("expected 1 saved entry, got %d", len(pl.entries))
	}
	if pl.warn == "" {
		t.Error("expected warning on partial failure")
	}
}

package feed

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bluesky-social/indigo/xrpc"

	"github.com/Hadidomena/tuiFeed/bsk"
	"github.com/Hadidomena/tuiFeed/config"
)

func TestNewStaticModel(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{AuthorHandle: "test.bsky.social"}},
	}
	m := NewStaticModel(posts, "My Title")
	if m.title != "My Title" {
		t.Errorf("expected 'My Title', got %q", m.title)
	}
	if len(m.posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(m.posts))
	}
	if m.pageSize != 10 {
		t.Errorf("expected pageSize 10, got %d", m.pageSize)
	}
}

func TestInit_noClient(t *testing.T) {
	m := NewStaticModel(nil, "Static Feed")
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected nil command when client is nil")
	}
}

func TestInit_withClient(t *testing.T) {
	m := Model{client: &xrpc.Client{}}
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected loadPosts command when client is set")
	}
}

func update(m Model, msg tea.Msg) (Model, *tea.Cmd) {
	newModel, cmd := m.Update(msg)
	result := newModel.(Model)
	if cmd != nil {
		return result, &cmd
	}
	return result, nil
}

func TestUpdate_quit(t *testing.T) {
	m := NewStaticModel([]bsk.FeedItem{
		{PostInfo: bsk.PostInfo{AuthorHandle: "test.bsky.social"}},
	}, "Test")
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("expected Quit command for 'q'")
	}
}

func TestUpdate_ctrlC(t *testing.T) {
	m := NewStaticModel([]bsk.FeedItem{
		{PostInfo: bsk.PostInfo{AuthorHandle: "test.bsky.social"}},
	}, "Test")

	msg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected Quit command for ctrl+c")
	}
}

func TestUpdate_esc(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected BackMsg command for esc")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("expected BackMsg, got %T", msg)
	}
}

func TestUpdate_downJ(t *testing.T) {
	posts := make([]bsk.FeedItem, 3)
	m := NewStaticModel(posts, "Test")
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = update(m, tea.KeyPressMsg{Code: 'j'})
	if m.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", m.cursor)
	}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor still 2 (at end), got %d", m.cursor)
	}
}

func TestUpdate_upK(t *testing.T) {
	posts := make([]bsk.FeedItem, 3)
	m := NewStaticModel(posts, "Test")
	m.cursor = 2
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = update(m, tea.KeyPressMsg{Code: 'k'})
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("expected cursor still 0 (at start), got %d", m.cursor)
	}
}

func TestUpdate_scrollWindow(t *testing.T) {
	posts := make([]bsk.FeedItem, 25)
	m := NewStaticModel(posts, "Test")
	m.cursor = 14
	m.scrollPos = 4
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 15 {
		t.Errorf("expected cursor 15, got %d", m.cursor)
	}
	if m.scrollPos != 5 {
		t.Errorf("expected scrollPos 5 (advanced), got %d", m.scrollPos)
	}
	m.cursor = 5
	m.scrollPos = 5
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.scrollPos != 4 {
		t.Errorf("expected scrollPos 4 (retreated), got %d", m.scrollPos)
	}
}

func TestUpdate_left(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg", "b.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m.hasRendered = true
	m.imgCursor = 1
	m, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.imgCursor != 0 {
		t.Errorf("expected imgCursor 0, got %d", m.imgCursor)
	}
	if cmd == nil {
		t.Error("expected render attachment command")
	}
}

func TestUpdate_leftAtZero(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg", "b.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m.hasRendered = true
	m.imgCursor = 0
	_, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	if cmd != nil {
		t.Error("expected no command when imgCursor at 0")
	}
}

func TestUpdate_right(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg", "b.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m.hasRendered = true
	m.imgCursor = 0
	m, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.imgCursor != 1 {
		t.Errorf("expected imgCursor 1, got %d", m.imgCursor)
	}
	if cmd == nil {
		t.Error("expected render attachment command")
	}
}

func TestUpdate_rightAtEnd(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg", "b.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m.hasRendered = true
	m.imgCursor = 1
	_, cmd := update(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd != nil {
		t.Error("expected no command when imgCursor at end")
	}
}

func TestUpdate_oKey(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m, cmd := update(m, tea.KeyPressMsg{Code: 'o'})
	if cmd == nil {
		t.Error("expected open attachment command")
	}
	if m.statusMsg != "Opening image externally..." {
		t.Errorf("expected status, got %q", m.statusMsg)
	}
}

func TestUpdate_oKeyNoEmbeds(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{AuthorHandle: "test.bsky.social"}},
	}
	m := NewStaticModel(posts, "Test")
	m, _ = update(m, tea.KeyPressMsg{Code: 'o'})
	if m.statusMsg == "Opening image externally..." {
		t.Error("expected no status when no embeds")
	}
}

func TestUpdate_leftH(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg", "b.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m.hasRendered = true
	m.imgCursor = 1
	m, cmd := update(m, tea.KeyPressMsg{Code: 'h'})
	if m.imgCursor != 0 {
		t.Errorf("expected imgCursor 0, got %d", m.imgCursor)
	}
	if cmd == nil {
		t.Error("expected render command")
	}
}

func TestUpdate_rightL(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg", "b.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m.hasRendered = true
	m.imgCursor = 0
	m, cmd := update(m, tea.KeyPressMsg{Code: 'l'})
	if m.imgCursor != 1 {
		t.Errorf("expected imgCursor 1, got %d", m.imgCursor)
	}
	if cmd == nil {
		t.Error("expected render command")
	}
}

func TestUpdate_aKey(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m, cmd := update(m, tea.KeyPressMsg{Code: 'a'})
	if !m.hasRendered {
		t.Error("expected hasRendered to be true")
	}
	if cmd == nil {
		t.Error("expected render attachment command")
	}
}

func TestUpdate_rKey(t *testing.T) {
	posts := make([]bsk.FeedItem, 1)
	m := NewStaticModel(posts, "Test")
	m.hasRendered = true
	m.imageRows = 10
	m.scrollPos = 5
	m, _ = update(m, tea.KeyPressMsg{Code: 'r'})
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

func TestUpdate_postsLoadedMsg(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m.loading = true
	m.cursor = 5
	newPosts := postsLoadedMsg([]bsk.FeedItem{
		{PostInfo: bsk.PostInfo{AuthorHandle: "a.bsky.social"}},
		{PostInfo: bsk.PostInfo{AuthorHandle: "b.bsky.social"}},
	})
	m, cmd := update(m, newPosts)
	if cmd != nil {
		t.Error("expected no command")
	}
	if m.loading {
		t.Error("expected loading false")
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	if len(m.posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(m.posts))
	}
}

func TestUpdate_postsLoadedMsg_empty(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m.loading = true
	m, _ = update(m, postsLoadedMsg([]bsk.FeedItem{}))
	if m.statusMsg != "No posts found." {
		t.Errorf("expected 'No posts found.', got %q", m.statusMsg)
	}
}

func TestUpdate_loadErrorMsg(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m.loading = true
	m, _ = update(m, loadErrorMsg("something went wrong"))
	if m.loading {
		t.Error("expected loading false")
	}
	if m.statusMsg != "something went wrong" {
		t.Errorf("expected error message, got %q", m.statusMsg)
	}
}

func TestUpdate_imageRenderedMsg(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m, _ = update(m, imageRenderedMsg{imageRows: 15})
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

func TestView_noPosts(t *testing.T) {
	m := NewStaticModel(nil, "Test Feed")
	m.loading = false
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_withPostsAndCursor(t *testing.T) {
	posts := []bsk.FeedItem{
		{
			PostInfo: bsk.PostInfo{
				AuthorDisplayName: "Test",
				AuthorHandle:      "test.bsky.social",
				Text:              "Hello world",
				LikeCount:         5,
				ReplyCount:        2,
				IndexedAt:         "2024-01-15T10:00:00Z",
			},
		},
	}
	m := NewStaticModel(posts, "My Feed")
	m.loading = false
	m.cursor = 0
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_renderedWithEmbeds(t *testing.T) {
	posts := []bsk.FeedItem{
		{
			PostInfo: bsk.PostInfo{
				AuthorHandle: "test.bsky.social",
				Text:         "Hello",
				IndexedAt:    "2024-01-15T10:00:00Z",
				Embeds:       []string{"a.jpg", "b.jpg"},
			},
		},
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 0
	m.hasRendered = true
	m.imageRows = 5
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_hasRenderedNoEmbeds(t *testing.T) {
	posts := []bsk.FeedItem{
		{
			PostInfo: bsk.PostInfo{
				AuthorHandle: "test.bsky.social",
				Text:         "Hello",
				IndexedAt:    "2024-01-15T10:00:00Z",
			},
		},
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 0
	m.hasRendered = true
	m.imageRows = 0
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_withStatusMessage(t *testing.T) {
	posts := []bsk.FeedItem{
		{
			PostInfo: bsk.PostInfo{
				AuthorHandle: "test.bsky.social",
				Text:         "Hello",
				IndexedAt:    "2024-01-15T10:00:00Z",
			},
		},
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 0
	m.statusMsg = "Some status"
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_cursorPastEnd(t *testing.T) {
	posts := []bsk.FeedItem{
		{
			PostInfo: bsk.PostInfo{
				AuthorHandle: "test.bsky.social",
				Text:         "Hello",
				IndexedAt:    "2024-01-15T10:00:00Z",
			},
		},
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 5
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_paginationAbove(t *testing.T) {
	posts := make([]bsk.FeedItem, 20)
	for i := range posts {
		posts[i] = bsk.FeedItem{
			PostInfo: bsk.PostInfo{
				AuthorHandle: "test.bsky.social",
				Text:         "Hello",
				IndexedAt:    "2024-01-15T10:00:00Z",
			},
		}
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 12
	m.scrollPos = 5
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_paginationBelow(t *testing.T) {
	posts := make([]bsk.FeedItem, 20)
	for i := range posts {
		posts[i] = bsk.FeedItem{
			PostInfo: bsk.PostInfo{
				AuthorHandle: "test.bsky.social",
				Text:         "Hello",
				IndexedAt:    "2024-01-15T10:00:00Z",
			},
		}
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 0
	m.scrollPos = 0
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_noCreatedAt(t *testing.T) {
	posts := []bsk.FeedItem{
		{
			PostInfo: bsk.PostInfo{
				AuthorHandle: "test.bsky.social",
				Text:         "Hello",
				CreatedAt:    "2024-06-01T12:00:00Z",
			},
		},
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 0
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_shortDateFormat(t *testing.T) {
	posts := []bsk.FeedItem{
		{
			PostInfo: bsk.PostInfo{
				AuthorHandle: "test.bsky.social",
				Text:         "Hello",
				IndexedAt:    "2024-01",
			},
		},
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 0
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_emptyAuthorFallback(t *testing.T) {
	posts := []bsk.FeedItem{
		{
			PostInfo: bsk.PostInfo{
				AuthorDisplayName: "",
				AuthorHandle:      "test.bsky.social",
				Text:              "Hello",
				IndexedAt:         "2024-01-15T10:00:00Z",
			},
		},
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 0
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_longTextTruncation(t *testing.T) {
	longText := ""
	for i := 0; i < 200; i++ {
		longText += "a"
	}
	posts := []bsk.FeedItem{
		{
			PostInfo: bsk.PostInfo{
				AuthorHandle: "test.bsky.social",
				Text:         longText,
				IndexedAt:    "2024-01-15T10:00:00Z",
			},
		},
	}
	m := NewStaticModel(posts, "Feed")
	m.loading = false
	m.cursor = 0
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestLoadPosts_noCfg(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	msg := m.loadPosts()
	errMsg, ok := msg.(loadErrorMsg)
	if !ok {
		t.Fatalf("expected loadErrorMsg, got %T", msg)
	}
	if string(errMsg) != "Not available in this view." {
		t.Errorf("expected 'Not available in this view.', got %q", string(errMsg))
	}
}

func TestLoadPosts_noClient(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m.cfg = &config.Config{}
	msg := m.loadPosts()
	errMsg, ok := msg.(loadErrorMsg)
	if !ok {
		t.Fatalf("expected loadErrorMsg, got %T", msg)
	}
	if string(errMsg) != "Not available in this view." {
		t.Errorf("expected error for no client, got %q", string(errMsg))
	}
}

func TestLoadPosts_noFollows(t *testing.T) {
	m := NewStaticModel(nil, "Test")
	m.cfg = &config.Config{}
	m.client = &xrpc.Client{}
	msg := m.loadPosts()
	errMsg, ok := msg.(loadErrorMsg)
	if !ok {
		t.Fatalf("expected loadErrorMsg, got %T", msg)
	}
	if string(errMsg) != "No followed accounts. Add some in Manage follows." {
		t.Errorf("expected no follows message, got %q", string(errMsg))
	}
}

func TestRenderAttachment_oob(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m.imgCursor = 5
	msg := m.renderAttachment()
	errMsg, ok := msg.(loadErrorMsg)
	if !ok {
		t.Fatalf("expected loadErrorMsg, got %T", msg)
	}
	if string(errMsg) != "No image to render" {
		t.Errorf("expected 'No image to render', got %q", string(errMsg))
	}
}

func TestOpenAttachment_oob(t *testing.T) {
	posts := []bsk.FeedItem{
		{PostInfo: bsk.PostInfo{Embeds: []string{"a.jpg"}}},
	}
	m := NewStaticModel(posts, "Test")
	m.imgCursor = -1
	msg := m.openAttachment()
	errMsg, ok := msg.(loadErrorMsg)
	if !ok {
		t.Fatalf("expected loadErrorMsg, got %T", msg)
	}
	if string(errMsg) != "No image to open" {
		t.Errorf("expected 'No image to open', got %q", string(errMsg))
	}
}

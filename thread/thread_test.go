package thread

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/attach"
	"github.com/Hadidomena/tuiFeed/bsk"
	"github.com/Hadidomena/tuiFeed/internal/testutil"
	"github.com/Hadidomena/tuiFeed/utils"
)

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func makeNode(handle, text, uri string, replies ...*bsk.ThreadNode) *bsk.ThreadNode {
	node := &bsk.ThreadNode{
		Post: bsk.PostInfo{
			AuthorHandle:      handle,
			AuthorDisplayName: "User " + handle,
			Text:              text,
			LikeCount:         5,
			ReplyCount:        int64(len(replies)),
			IndexedAt:         "2024-06-01T12:00:00Z",
		},
		URI: uri,
	}
	for _, r := range replies {
		r.Parent = node
		node.Replies = append(node.Replies, r)
	}
	return node
}

type modelOption func(*Model)

func defaultModel(root *bsk.ThreadNode, opts ...modelOption) Model {
	m := Model{
		root:       root,
		current:    root,
		replies:    root.Replies,
		breadcrumb: []string{"@" + root.Post.AuthorHandle},
		pageSize:   10,
		width:      utils.DefaultWidth,
		height:     40,
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func withCurrent(n *bsk.ThreadNode) modelOption {
	return func(m *Model) { m.current = n; m.replies = n.Replies }
}

func withCursor(c int) modelOption {
	return func(m *Model) { m.cursor = c }
}

func withBreadcrumb(bc []string) modelOption {
	return func(m *Model) { m.breadcrumb = bc }
}

func TestInit_withURI(t *testing.T) {
	m := NewModel("at://uri/test")
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected non-nil command when uri is set")
	}
}

func TestInit_emptyURI(t *testing.T) {
	m := NewModel("")
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected nil command when uri is empty")
	}
}

func TestView_loading(t *testing.T) {
	m := NewModel("at://uri/test")
	v := m.View()
	if !strings.Contains(v.Content, "Loading comments") {
		t.Errorf("expected loading message, got: %s", v.Content)
	}
}

func TestView_error(t *testing.T) {
	m := Model{statusMsg: "Error loading"}
	v := m.View()
	if !strings.Contains(v.Content, "Error loading") {
		t.Errorf("expected error message in view, got: %s", v.Content)
	}
}

func TestView_noThreadData(t *testing.T) {
	m := Model{loading: false}
	v := m.View()
	if !strings.Contains(v.Content, "No thread data") {
		t.Errorf("expected 'No thread data', got: %s", v.Content)
	}
}

func TestView_noReplies(t *testing.T) {
	root := makeNode("author", "Root post", "at://uri/root")
	m := defaultModel(root)
	v := m.View()
	if !strings.Contains(v.Content, "No replies yet") {
		t.Errorf("expected 'No replies yet', got: %s", v.Content)
	}
}

func TestView_withReplies(t *testing.T) {
	reply := makeNode("replier", "A reply", "at://uri/reply")
	root := makeNode("author", "Root post", "at://uri/root", reply)
	m := defaultModel(root)
	v := m.View()
	if !strings.Contains(v.Content, "@replier") {
		t.Errorf("expected reply handle in view, got: %s", v.Content)
	}
	if !strings.Contains(v.Content, "1 reply") {
		t.Errorf("expected '1 reply', got: %s", v.Content)
	}
}

func TestView_breadcrumb(t *testing.T) {
	reply := makeNode("replier", "A reply", "at://uri/reply")
	root := makeNode("author", "Root post", "at://uri/root", reply)
	m := defaultModel(root, withCurrent(reply), withBreadcrumb([]string{"@author", "@replier"}))
	v := m.View()
	if !strings.Contains(v.Content, "@author > @replier") {
		t.Errorf("expected breadcrumb '@author > @replier', got: %s", v.Content)
	}
}

func TestView_helpBar_atRoot(t *testing.T) {
	root := makeNode("author", "Root post", "at://uri/root")
	m := defaultModel(root)
	v := m.View()
	if strings.Contains(v.Content, "[h] parent") {
		t.Errorf("expected no 'h' at root level, got: %s", v.Content)
	}
	if !strings.Contains(v.Content, "[j/k]") {
		t.Errorf("expected 'j/k' in help bar, got: %s", v.Content)
	}
	if !strings.Contains(v.Content, "[esc] to feed") {
		t.Errorf("expected 'esc' in help bar, got: %s", v.Content)
	}
}

func TestView_helpBar_withParent(t *testing.T) {
	reply := makeNode("replier", "A reply", "at://uri/reply")
	root := makeNode("author", "Root post", "at://uri/root", reply)
	m := defaultModel(root, withCurrent(reply), withBreadcrumb([]string{"@author", "@replier"}))
	v := m.View()
	if !strings.Contains(v.Content, "[h] parent") {
		t.Errorf("expected 'h' when not at root, got: %s", v.Content)
	}
}

func TestUpdate_threadLoadedSuccess(t *testing.T) {
	reply := makeNode("replier", "A reply", "at://uri/reply")
	root := makeNode("author", "Root post", "at://uri/root", reply)
	msg := threadLoadedMsg{root: root}

	result, cmd := Model{loading: true}.Update(msg)
	m := result.(Model)

	if m.loading {
		t.Error("expected loading to be false after successful load")
	}
	if m.root != root {
		t.Error("expected root to be set")
	}
	if m.current != root {
		t.Error("expected current to be root")
	}
	if len(m.replies) != 1 {
		t.Errorf("expected 1 reply, got %d", len(m.replies))
	}
	if len(m.breadcrumb) != 1 || m.breadcrumb[0] != "@author" {
		t.Errorf("expected breadcrumb [@author], got %v", m.breadcrumb)
	}
	if cmd != nil {
		t.Error("expected nil command from threadLoadedMsg")
	}
}

func TestUpdate_threadLoadedError(t *testing.T) {
	errMsg := "fetch failed"
	msg := threadLoadedMsg{err: &testError{msg: errMsg}}

	result, cmd := Model{loading: true}.Update(msg)
	m := result.(Model)

	if m.loading {
		t.Error("expected loading to be false after error")
	}
	if m.root != nil {
		t.Error("expected root to be nil after error")
	}
	if !strings.Contains(m.statusMsg, errMsg) {
		t.Errorf("expected statusMsg to contain %q, got %q", errMsg, m.statusMsg)
	}
	if cmd != nil {
		t.Error("expected nil command from error")
	}
}

func TestUpdate_threadLoadedNilRoot(t *testing.T) {
	result, cmd := Model{loading: true}.Update(threadLoadedMsg{root: nil})
	m := result.(Model)

	if m.loading {
		t.Error("expected loading to be false")
	}
	if m.root != nil {
		t.Error("expected root to stay nil")
	}
	if !strings.Contains(m.statusMsg, "empty thread data") {
		t.Errorf("expected status about empty data, got %q", m.statusMsg)
	}
	if cmd != nil {
		t.Error("expected nil command")
	}
}

func TestUpdate_quitKeys(t *testing.T) {
	m := Model{loading: true}
	for _, key := range []tea.KeyPressMsg{
		testutil.KeyRune('q'),
		testutil.KeyPress("", 'c', tea.ModCtrl),
	} {
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Error("expected Quit command for quit key")
		}
	}
}

func TestUpdate_escWhileLoading(t *testing.T) {
	m := Model{loading: true}
	_, cmd := m.Update(testutil.KeySpecial(tea.KeyEsc))
	if cmd != nil {
		t.Error("expected nil command for esc while loading")
	}
}

func TestUpdate_escAfterLoading(t *testing.T) {
	root := makeNode("author", "Root post", "at://uri/root")
	m := Model{
		root:    root,
		current: root,
		replies: root.Replies,
	}

	result, cmd := m.Update(testutil.KeySpecial(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("expected non-nil command for esc after loading")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("expected BackMsg, got %T", msg)
	}
	_ = result
}

func TestUpdate_jkNavigation(t *testing.T) {
	reply1 := makeNode("r1", "Reply 1", "at://uri/r1")
	reply2 := makeNode("r2", "Reply 2", "at://uri/r2")
	root := makeNode("author", "Root", "at://uri/root", reply1, reply2)
	m := defaultModel(root)

	r, _ := m.Update(testutil.KeyRune('j'))
	if r.(Model).cursor != 1 {
		t.Errorf("expected cursor 1 after j, got %d", m.cursor)
	}

	r, _ = m.Update(testutil.KeyRune('k'))
	m = r.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 after k, got %d", m.cursor)
	}
}

func TestUpdate_jAtBottom(t *testing.T) {
	reply := makeNode("r1", "Reply 1", "at://uri/r1")
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := Model{
		root:     root,
		current:  root,
		replies:  root.Replies,
		cursor:   0,
		pageSize: 10,
	}

	r, _ := m.Update(testutil.KeyRune('j'))
	m = r.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 (no more items), got %d", m.cursor)
	}
}

func TestUpdate_kAtTop(t *testing.T) {
	reply1 := makeNode("r1", "Reply 1", "at://uri/r1")
	reply2 := makeNode("r2", "Reply 2", "at://uri/r2")
	root := makeNode("author", "Root", "at://uri/root", reply1, reply2)
	m := Model{
		root:     root,
		current:  root,
		replies:  root.Replies,
		cursor:   0,
		pageSize: 10,
	}

	r, _ := m.Update(testutil.KeyRune('k'))
	m = r.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 (already at top), got %d", m.cursor)
	}
}

func TestUpdate_enterDrillsIntoReply(t *testing.T) {
	nested := makeNode("nested", "Nested reply", "at://uri/nested")
	reply := makeNode("replier", "A reply", "at://uri/reply", nested)
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)

	r, _ := m.Update(testutil.KeySpecial(tea.KeyEnter))
	m = r.(Model)

	if m.current != reply {
		t.Error("expected current to be the drilled-into reply")
	}
	if len(m.breadcrumb) != 2 || m.breadcrumb[1] != "@replier" {
		t.Errorf("expected breadcrumb [@author, @replier], got %v", m.breadcrumb)
	}
	if len(m.replies) != 1 {
		t.Errorf("expected 1 reply at new level, got %d", len(m.replies))
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor reset to 0, got %d", m.cursor)
	}
}

func TestUpdate_enterNoOpOnReplyWithoutReplies(t *testing.T) {
	reply := makeNode("replier", "A reply", "at://uri/reply")
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root, withCursor(0))

	r, _ := m.Update(testutil.KeySpecial(tea.KeyEnter))
	m = r.(Model)

	if m.current != root {
		t.Error("expected current to remain root when reply has no nested replies")
	}
}

func TestUpdate_enterNoOpWhenEmptyReplies(t *testing.T) {
	root := makeNode("author", "Root", "at://uri/root")
	m := defaultModel(root)

	r, _ := m.Update(testutil.KeySpecial(tea.KeyEnter))
	m = r.(Model)

	if m.current != root {
		t.Error("expected current to remain root when no replies exist")
	}
}

func TestUpdate_hGoesUpToParent(t *testing.T) {
	nested := makeNode("nested", "Nested", "at://uri/nested")
	reply := makeNode("replier", "A reply", "at://uri/reply", nested)
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root, withCurrent(reply), withBreadcrumb([]string{"@author", "@replier"}))

	r, _ := m.Update(testutil.KeyRune('h'))
	m = r.(Model)

	if m.current != root {
		t.Error("expected current to go back to root after h")
	}
	if len(m.breadcrumb) != 1 || m.breadcrumb[0] != "@author" {
		t.Errorf("expected breadcrumb [@author], got %v", m.breadcrumb)
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor reset to 0, got %d", m.cursor)
	}
}

func TestUpdate_hNoOpAtRoot(t *testing.T) {
	reply := makeNode("replier", "A reply", "at://uri/reply")
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)

	r, _ := m.Update(testutil.KeyRune('h'))
	m = r.(Model)

	if m.current != root {
		t.Error("expected current to stay at root when pressing h at root")
	}
}

func TestUpdate_backspaceSameAsH(t *testing.T) {
	nested := makeNode("nested", "Nested", "at://uri/nested")
	reply := makeNode("replier", "A reply", "at://uri/reply", nested)
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root, withCurrent(reply), withBreadcrumb([]string{"@author", "@replier"}))

	r, _ := m.Update(testutil.KeySpecial(tea.KeyBackspace))
	m = r.(Model)

	if m.current != root {
		t.Error("expected backspace to go up to root like h")
	}
}

func TestUpdate_jkWhileLoading(t *testing.T) {
	m := Model{loading: true}
	r, _ := m.Update(testutil.KeyRune('j'))
	m = r.(Model)
	if m.cursor != 0 {
		t.Errorf("expected no cursor movement while loading, got %d", m.cursor)
	}
	r, _ = m.Update(testutil.KeyRune('k'))
	m = r.(Model)
	if m.cursor != 0 {
		t.Errorf("expected no cursor movement while loading, got %d", m.cursor)
	}
}

func TestUpdate_enterWhileLoading(t *testing.T) {
	m := Model{loading: true}
	r, _ := m.Update(testutil.KeySpecial(tea.KeyEnter))
	m = r.(Model)
	if m.current != nil {
		t.Error("expected no drill-in while loading")
	}
}

func TestView_showsSelectedReplyDetail(t *testing.T) {
	reply := makeNode("replier", "A reply with some text", "at://uri/reply")
	root := makeNode("author", "Root post", "at://uri/root", reply)
	m := defaultModel(root)
	v := m.View()
	if !strings.Contains(v.Content, "A reply with some text") {
		t.Errorf("expected reply text in view, got: %s", v.Content)
	}
	if !strings.Contains(v.Content, "\u2764\ufe0f 5") {
		t.Errorf("expected like count in view, got: %s", v.Content)
	}
}

func TestView_replyCountLabel(t *testing.T) {
	reply1 := makeNode("r1", "R1", "at://uri/r1")
	reply2 := makeNode("r2", "R2", "at://uri/r2")
	root := makeNode("author", "Root", "at://uri/root", reply1, reply2)
	m := defaultModel(root)
	v := m.View()
	if !strings.Contains(v.Content, "2 replies") {
		t.Errorf("expected '2 replies', got: %s", v.Content)
	}
}

func TestView_showsAttachmentCount(t *testing.T) {
	reply := makeNode("replier", "A reply", "at://uri/reply")
	reply.Post.Embeds = []string{"https://example.com/img.jpg"}
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)
	v := m.View()
	if !strings.Contains(v.Content, "1 attachment") {
		t.Errorf("expected attachment count in view, got: %s", v.Content)
	}
}

func TestView_showsNestedReplyCount(t *testing.T) {
	nested := makeNode("nested", "Nested", "at://uri/nested")
	reply := makeNode("replier", "A reply", "at://uri/reply", nested)
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)
	v := m.View()
	if !strings.Contains(v.Content, "[1 reply]") {
		t.Errorf("expected nested reply indicator, got: %s", v.Content)
	}
}

func TestUpdate_scrollAdvances(t *testing.T) {
	replies := make([]*bsk.ThreadNode, 15)
	for i := range replies {
		replies[i] = makeNode(
			fmt.Sprintf("user%d", i),
			fmt.Sprintf("Reply %d", i),
			fmt.Sprintf("at://uri/r%d", i),
		)
	}
	root := makeNode("author", "Root", "at://uri/root", replies...)
	m := defaultModel(root)

	for i := 1; i <= 10; i++ {
		r, _ := m.Update(testutil.KeyRune('j'))
		m = r.(Model)
	}

	if m.scrollPos != 3 {
		t.Errorf("expected scrollPos 3 after 10 j presses, got %d", m.scrollPos)
	}
	if m.cursor != 10 {
		t.Errorf("expected cursor 10 after 10 j presses, got %d", m.cursor)
	}
}

func TestUpdate_scrollRetreats(t *testing.T) {
	replies := make([]*bsk.ThreadNode, 15)
	for i := range replies {
		replies[i] = makeNode(
			fmt.Sprintf("user%d", i),
			fmt.Sprintf("Reply %d", i),
			fmt.Sprintf("at://uri/r%d", i),
		)
	}
	root := makeNode("author", "Root", "at://uri/root", replies...)
	m := Model{
		root:      root,
		current:   root,
		replies:   root.Replies,
		pageSize:  10,
		cursor:    1,
		scrollPos: 1,
	}

	// Press k: cursor goes from 1 to 0, which is < scrollPos (1), so scrollPos retreats to 0
	r, _ := m.Update(testutil.KeyRune('k'))
	m = r.(Model)

	if m.scrollPos != 0 {
		t.Errorf("expected scrollPos 0 after retreat, got %d", m.scrollPos)
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
}

func TestView_moreAboveBelow(t *testing.T) {
	replies := make([]*bsk.ThreadNode, 15)
	for i := range replies {
		replies[i] = makeNode(
			fmt.Sprintf("user%d", i),
			fmt.Sprintf("Reply %d", i),
			fmt.Sprintf("at://uri/r%d", i),
		)
	}
	root := makeNode("author", "Root", "at://uri/root", replies...)
	m := Model{
		root:      root,
		current:   root,
		replies:   root.Replies,
		pageSize:  10,
		cursor:    5,
		scrollPos: 3,
	}
	v := m.View()
	if !strings.Contains(v.Content, "more above") {
		t.Errorf("expected 'more above' indicator, got: %s", v.Content)
	}
	if !strings.Contains(v.Content, "more below") {
		t.Errorf("expected 'more below' indicator, got: %s", v.Content)
	}
}

func TestView_breadcrumbTruncated(t *testing.T) {
	longBc := []string{"@a", "@b", "@c", "@d", "@e", "@f", "@g", "@h"}
	root := makeNode("a", "Root", "at://uri/a")
	m := defaultModel(root, withBreadcrumb(longBc))
	v := m.View()
	if !strings.Contains(v.Content, "... > @d > @e > @f > @g > @h") {
		t.Errorf("expected truncated breadcrumb, got: %s", v.Content)
	}
}

func TestView_breadcrumbNotTruncated(t *testing.T) {
	shortBc := []string{"@a", "@b", "@c"}
	root := makeNode("a", "Root", "at://uri/a")
	m := defaultModel(root, withBreadcrumb(shortBc))
	v := m.View()
	if strings.Contains(v.Content, "... >") {
		t.Errorf("expected no truncation for short breadcrumb, got: %s", v.Content)
	}
}

func TestView_breadcrumbAtLimit(t *testing.T) {
	limitBc := []string{"@a", "@b", "@c", "@d", "@e"}
	root := makeNode("a", "Root", "at://uri/a")
	m := defaultModel(root, withBreadcrumb(limitBc))
	v := m.View()
	if strings.Contains(v.Content, "... >") {
		t.Errorf("expected no truncation at exactly maxBreadcrumb, got: %s", v.Content)
	}
}

func TestUpdate_aKey_rendersAttachment(t *testing.T) {
	reply := makeNode("replier", "Reply", "at://uri/reply")
	reply.Post.Embeds = []string{"https://example.com/img.jpg"}
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)
	_, cmd := m.Update(testutil.KeyRune('a'))
	if cmd == nil {
		t.Fatal("expected command for a key with embeds")
	}
}

func TestUpdate_aKey_noEmbeds(t *testing.T) {
	reply := makeNode("replier", "Reply", "at://uri/reply")
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)
	_, cmd := m.Update(testutil.KeyRune('a'))
	if cmd != nil {
		t.Error("expected no command for a key without embeds")
	}
}

func TestUpdate_aKey_noReplies(t *testing.T) {
	root := makeNode("author", "Root", "at://uri/root")
	m := defaultModel(root)
	_, cmd := m.Update(testutil.KeyRune('a'))
	if cmd != nil {
		t.Error("expected no command for a key with no replies")
	}
}

func TestUpdate_oKey_opensAttachment(t *testing.T) {
	reply := makeNode("replier", "Reply", "at://uri/reply")
	reply.Post.Embeds = []string{"https://example.com/img.jpg"}
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)
	r, cmd := m.Update(testutil.KeyRune('o'))
	m = r.(Model)
	if cmd == nil {
		t.Fatal("expected command for o key with embeds")
	}
	if m.statusMsg != "Opening image externally..." {
		t.Errorf("expected status message, got %q", m.statusMsg)
	}
}

func TestUpdate_oKey_noEmbeds(t *testing.T) {
	reply := makeNode("replier", "Reply", "at://uri/reply")
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)
	_, cmd := m.Update(testutil.KeyRune('o'))
	if cmd != nil {
		t.Error("expected no command for o key without embeds")
	}
}

func TestUpdate_leftRight_imageNavigation(t *testing.T) {
	reply := makeNode("replier", "Reply", "at://uri/reply")
	reply.Post.Embeds = []string{"a.jpg", "b.jpg"}
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := Model{
		root:        root,
		current:     root,
		replies:     root.Replies,
		pageSize:    10,
		hasRendered: true,
		imgCursor:   1,
	}
	r, cmd := m.Update(testutil.KeySpecial(tea.KeyLeft))
	m = r.(Model)
	if cmd == nil {
		t.Fatal("expected command for left arrow")
	}
	if m.imgCursor != 0 {
		t.Errorf("expected imgCursor 0, got %d", m.imgCursor)
	}

	r, cmd = m.Update(testutil.KeySpecial(tea.KeyRight))
	m = r.(Model)
	if cmd == nil {
		t.Fatal("expected command for right arrow")
	}
	if m.imgCursor != 1 {
		t.Errorf("expected imgCursor 1, got %d", m.imgCursor)
	}
}

func TestUpdate_leftNoOpAtFirstImage(t *testing.T) {
	reply := makeNode("replier", "Reply", "at://uri/reply")
	reply.Post.Embeds = []string{"a.jpg", "b.jpg"}
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := Model{
		root:        root,
		current:     root,
		replies:     root.Replies,
		pageSize:    10,
		hasRendered: true,
		imgCursor:   0,
	}
	r, _ := m.Update(testutil.KeySpecial(tea.KeyLeft))
	m = r.(Model)
	if m.imgCursor != 0 {
		t.Errorf("expected imgCursor to stay 0, got %d", m.imgCursor)
	}
}

func TestUpdate_imageRenderedMsg(t *testing.T) {
	m := Model{}
	r, _ := m.Update(attach.RenderedMsg{ImageRows: 15, Status: "rendered"})
	m = r.(Model)
	if m.imageRows != 15 {
		t.Errorf("expected imageRows 15, got %d", m.imageRows)
	}
	if m.statusMsg != "rendered" {
		t.Errorf("expected status 'rendered', got %q", m.statusMsg)
	}
}

func TestUpdate_renderErrorMsg(t *testing.T) {
	m := Model{}
	r, _ := m.Update(attach.ErrorMsg("something went wrong"))
	m = r.(Model)
	if m.statusMsg != "something went wrong" {
		t.Errorf("expected status, got %q", m.statusMsg)
	}
}

func TestUpdate_jkResetsImageState(t *testing.T) {
	reply1 := makeNode("r1", "R1", "at://uri/r1")
	reply2 := makeNode("r2", "R2", "at://uri/r2")
	root := makeNode("author", "Root", "at://uri/root", reply1, reply2)
	m := Model{
		root:        root,
		current:     root,
		replies:     root.Replies,
		pageSize:    10,
		hasRendered: true,
		imageRows:   10,
		imgCursor:   2,
		statusMsg:   "some status",
	}
	r, _ := m.Update(testutil.KeyRune('j'))
	m = r.(Model)
	if m.hasRendered {
		t.Error("expected hasRendered to be reset after j")
	}
	if m.imageRows != 0 {
		t.Errorf("expected imageRows 0, got %d", m.imageRows)
	}
	if m.statusMsg != "" {
		t.Errorf("expected empty statusMsg, got %q", m.statusMsg)
	}
}

func TestView_helpBarShowsAttachmentKeys(t *testing.T) {
	reply := makeNode("replier", "Reply", "at://uri/reply")
	reply.Post.Embeds = []string{"https://example.com/img.jpg"}
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)
	v := m.View()
	if !strings.Contains(v.Content, "[a] attachments") {
		t.Errorf("expected '[a] attachments' in help, got: %s", v.Content)
	}
}

func TestView_helpBarNoAttachmentKeysWithoutEmbeds(t *testing.T) {
	reply := makeNode("replier", "Reply", "at://uri/reply")
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := defaultModel(root)
	v := m.View()
	if strings.Contains(v.Content, "[a] attachments") {
		t.Errorf("expected no attachment keys without embeds, got: %s", v.Content)
	}
}

func TestView_downKeyNoOpAtBottom(t *testing.T) {
	reply := makeNode("r1", "Reply", "at://uri/r1")
	root := makeNode("author", "Root", "at://uri/root", reply)
	m := Model{
		root:     root,
		current:  root,
		replies:  root.Replies,
		cursor:   0,
		pageSize: 10,
	}
	r, _ := m.Update(testutil.KeyRune('j'))
	m = r.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0 (only 1 item), got %d", m.cursor)
	}
}

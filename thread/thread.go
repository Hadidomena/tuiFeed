package thread

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/bsk"
)

type BackMsg struct{}

type threadLoadedMsg struct {
	root *bsk.ThreadNode
	err  error
}

type Model struct {
	root       *bsk.ThreadNode
	current    *bsk.ThreadNode
	replies    []*bsk.ThreadNode
	cursor     int
	scrollPos  int
	pageSize   int
	breadcrumb []string
	loading    bool
	statusMsg  string
	uri        string
}

func NewModel(uri string) Model {
	return Model{
		uri:      uri,
		loading:  true,
		pageSize: 10,
	}
}

func (m Model) Init() tea.Cmd {
	if m.uri == "" {
		return nil
	}
	return m.loadThread
}

func (m Model) loadThread() tea.Msg {
	ctx := context.Background()
	client := bsk.NewClient()
	root, err := bsk.GetThreadByURI(ctx, client, m.uri)
	if err != nil {
		return threadLoadedMsg{err: err}
	}
	return threadLoadedMsg{root: root}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.loading {
				return m, nil
			}
			return m, func() tea.Msg { return BackMsg{} }
		case "down", "j":
			if !m.loading && m.cursor < len(m.replies)-1 {
				m.cursor++
				if m.cursor >= m.scrollPos+m.pageSize {
					m.scrollPos++
				}
			}
		case "up", "k":
			if !m.loading && m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scrollPos {
					m.scrollPos--
				}
			}
		case "h", "backspace":
			if !m.loading && m.current != m.root {
				m.current = m.current.Parent
				m.replies = m.current.Replies
				m.breadcrumb = m.breadcrumb[:len(m.breadcrumb)-1]
				m.cursor = 0
				m.scrollPos = 0
			}
		case "enter":
			if !m.loading && len(m.replies) > 0 && m.cursor < len(m.replies) {
				reply := m.replies[m.cursor]
				if len(reply.Replies) > 0 {
					m.current = reply
					m.replies = reply.Replies
					m.breadcrumb = append(m.breadcrumb, "@"+reply.Post.AuthorHandle)
					m.cursor = 0
					m.scrollPos = 0
				}
			}
		}
	case threadLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error loading thread: %v", msg.err)
		} else if msg.root == nil {
			m.statusMsg = "Error: empty thread data received"
		} else {
			m.root = msg.root
			m.current = msg.root
			m.replies = msg.root.Replies
			m.breadcrumb = []string{"@" + msg.root.Post.AuthorHandle}
		}
	}

	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder

	if m.statusMsg != "" && m.root == nil {
		b.WriteString("Comments\n")
		b.WriteString(strings.Repeat("─", 30))
		b.WriteString("\n\n")
		b.WriteString(m.statusMsg + "\n")
		return tea.NewView(b.String())
	}

	if m.loading {
		b.WriteString("Loading comments...\n")
		return tea.NewView(b.String())
	}

	if m.root == nil {
		b.WriteString("No thread data.\n")
		return tea.NewView(b.String())
	}

	b.WriteString("Comments\n")
	b.WriteString(strings.Repeat("─", 30))
	b.WriteString("\n\n")

	b.WriteString(strings.Join(m.breadcrumb, " > "))
	b.WriteString("\n\n")

	if len(m.replies) == 0 {
		b.WriteString("  No replies yet.\n")
	} else {
		replyLabel := "ies"
		if len(m.replies) == 1 {
			replyLabel = "y"
		}
		b.WriteString(fmt.Sprintf("%d repl%s\n\n", len(m.replies), replyLabel))

		end := m.scrollPos + m.pageSize
		if end > len(m.replies) {
			end = len(m.replies)
		}

		for i := m.scrollPos; i < end; i++ {
			reply := m.replies[i]
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}

			createdAt := reply.Post.IndexedAt
			if createdAt == "" {
				createdAt = reply.Post.CreatedAt
			}
			if len(createdAt) > 10 {
				createdAt = createdAt[:10]
			}

			author := reply.Post.AuthorDisplayName
			if author == "" {
				author = reply.Post.AuthorHandle
			}

			b.WriteString(fmt.Sprintf("%s@%s (%s)  \u2764\ufe0f %d  \U0001f4ac %d  \U0001f4c5 %s\n",
				cursor, reply.Post.AuthorHandle, author, reply.Post.LikeCount, reply.Post.ReplyCount, createdAt))

			text := strings.TrimSpace(reply.Post.Text)
			if len(text) > 120 {
				text = text[:120] + "..."
			}
			text = strings.ReplaceAll(text, "\n", " ")
			b.WriteString(fmt.Sprintf("    %s\n", text))

			if len(reply.Post.Embeds) > 0 {
				b.WriteString(fmt.Sprintf("    [%d attachment(s)]\n", len(reply.Post.Embeds)))
			}
			if len(reply.Replies) > 0 {
				nestedLabel := "ies"
				if len(reply.Replies) == 1 {
					nestedLabel = "y"
				}
				b.WriteString(fmt.Sprintf("    [%d repl%s]\n", len(reply.Replies), nestedLabel))
			}
			b.WriteString("\n")
		}

		if m.scrollPos > 0 {
			b.WriteString(fmt.Sprintf("  ... %d more above\n", m.scrollPos))
		}
		if end < len(m.replies) {
			b.WriteString(fmt.Sprintf("  ... %d more below\n", len(m.replies)-end))
		}
		idx := m.cursor
		if idx >= len(m.replies) {
			idx = 0
		}
		b.WriteString(fmt.Sprintf("Reply %d/%d\n\n", idx+1, len(m.replies)))

		selected := m.replies[idx]
		b.WriteString(bsk.FormatPost(bsk.FeedItem{PostInfo: selected.Post, URI: selected.URI}, -1))
	}

	if m.statusMsg != "" {
		b.WriteString(fmt.Sprintf("%s\n", m.statusMsg))
	}

	help := "\n[j/k] navigate  [enter] drill"
	if m.current != m.root {
		help += "  [h] parent"
	}
	help += "  [esc] back  [q] quit\n"
	b.WriteString(help)

	return tea.NewView(b.String())
}

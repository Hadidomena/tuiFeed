package thread

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/attach"
	"github.com/Hadidomena/tuiFeed/bsk"
	"github.com/Hadidomena/tuiFeed/utils"
)

type BackMsg struct{}

type threadLoadedMsg struct {
	root *bsk.ThreadNode
	err  error
}

type Model struct {
	root        *bsk.ThreadNode
	current     *bsk.ThreadNode
	replies     []*bsk.ThreadNode
	cursor      int
	scrollPos   int
	pageSize    int
	breadcrumb  []string
	loading     bool
	statusMsg   string
	uri         string
	imgCursor   int
	hasRendered bool
	imageRows   int
	width       int
	height      int
}

func (m Model) WithSize(w, h int) Model {
	m.width = w
	m.height = h
	m.pageSize = utils.PageSize(h)
	return m
}

func NewModel(uri string) Model {
	return Model{
		uri:      uri,
		loading:  true,
		pageSize: utils.DefaultPageSize,
		width:    utils.DefaultWidth,
		height:   24,
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			bsk.ClearImages()
			return m, tea.Quit
		case "esc":
			m.hasRendered = false
			m.imageRows = 0
			bsk.ClearImages()
			if m.loading {
				return m, nil
			}
			return m, func() tea.Msg { return BackMsg{} }
		case "down", "j":
			if !m.loading {
				old := m.cursor
				oldScroll := m.scrollPos
				m.cursor, m.scrollPos = utils.ScrollDown(m.cursor, m.scrollPos, m.pageSize, len(m.replies))
				if m.cursor != old || m.scrollPos != oldScroll {
					m.imgCursor = 0
					m.hasRendered = false
					m.imageRows = 0
					m.statusMsg = ""
					bsk.ClearImages()
				}
			}
		case "up", "k":
			if !m.loading {
				old := m.cursor
				oldScroll := m.scrollPos
				m.cursor, m.scrollPos = utils.ScrollUp(m.cursor, m.scrollPos)
				if m.cursor != old || m.scrollPos != oldScroll {
					m.imgCursor = 0
					m.hasRendered = false
					m.imageRows = 0
					m.statusMsg = ""
					bsk.ClearImages()
				}
			}
		case "h", "backspace":
			if !m.loading && m.current != m.root {
				m.current = m.current.Parent
				m.replies = m.current.Replies
				m.breadcrumb = m.breadcrumb[:len(m.breadcrumb)-1]
				m.cursor = 0
				m.scrollPos = 0
				m.imgCursor = 0
				m.hasRendered = false
				m.imageRows = 0
				m.statusMsg = ""
				bsk.ClearImages()
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
					m.imgCursor = 0
					m.hasRendered = false
					m.imageRows = 0
					m.statusMsg = ""
					bsk.ClearImages()
				}
			}
		case "left":
			if !m.loading && m.hasRendered && m.imgCursor > 0 && len(m.replies) > 0 && m.cursor < len(m.replies) {
				m.imgCursor--
				return m, m.renderAttachment
			}
		case "right", "l":
			if !m.loading && m.hasRendered && len(m.replies) > 0 && m.cursor < len(m.replies) && m.imgCursor < len(m.replies[m.cursor].Post.Embeds)-1 {
				m.imgCursor++
				return m, m.renderAttachment
			}
		case "a":
			if !m.loading && len(m.replies) > 0 && m.cursor < len(m.replies) && len(m.replies[m.cursor].Post.Embeds) > 0 {
				yOff := m.computeYOffset()
				if attach.ComputeMaxRows(m.height, yOff) <= 0 {
					m.statusMsg = "Not enough room. Resize terminal taller or scroll up."
					return m, nil
				}
				m.imgCursor = 0
				m.hasRendered = true
				return m, m.renderAttachment
			}
		case "o":
			if !m.loading && len(m.replies) > 0 && m.cursor < len(m.replies) && len(m.replies[m.cursor].Post.Embeds) > 0 {
				if !m.hasRendered {
					m.hasRendered = true
					m.imgCursor = 0
				}
				m.statusMsg = "Opening image externally..."
				return m, m.openAttachment
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
	case attach.RenderedMsg:
		m.imageRows = msg.ImageRows
		m.statusMsg = msg.Status
	case attach.ErrorMsg:
		m.statusMsg = string(msg)
	}

	m = m.recalcPageSize()
	return m, nil
}

func (m Model) recalcPageSize() Model {
	if m.height <= 0 || m.root == nil || len(m.replies) == 0 {
		return m
	}

	cw := utils.ContentWidth(m.width)
	budget := utils.TextBudget(m.height)

	idx := m.cursor
	if idx >= len(m.replies) {
		idx = 0
	}
	postLines := strings.Count(bsk.FormatPost(bsk.FeedItem{PostInfo: m.replies[idx].Post, URI: m.replies[idx].URI}, -1, cw, budget), "\n")

	itemLines := strings.Count(bsk.FormatPostListItem(m.replies[idx].Post, false, cw), "\n") + 1
	if len(m.replies[idx].Replies) > 0 {
		itemLines++
	}
	if itemLines < 1 {
		itemLines = 1
	}

	reserve := 10
	available := m.height - reserve - postLines
	m.pageSize = available / itemLines
	if m.pageSize < 0 {
		m.pageSize = 0
	}
	return m
}

func (m Model) renderAttachment() tea.Msg {
	if len(m.replies) == 0 || m.cursor >= len(m.replies) {
		return attach.ErrorMsg("No reply selected")
	}
	yOffset := m.computeYOffset()
	maxCols := attach.ComputeMaxCols(m.width)
	maxRows := attach.ComputeMaxRows(m.height, yOffset)
	return attach.Render(m.replies[m.cursor].Post.Embeds, m.imgCursor, yOffset, maxCols, maxRows)
}

func (m Model) computeYOffset() int {
	if m.root == nil {
		return 4
	}

	lines := 0

	lines += 3

	const maxBreadcrumb = 5
	breadcrumbLine := strings.Join(m.breadcrumb, " > ")
	if len(m.breadcrumb) > maxBreadcrumb {
		breadcrumbLine = "... > " + strings.Join(m.breadcrumb[len(m.breadcrumb)-maxBreadcrumb:], " > ")
	}
	lines += 1 + strings.Count(breadcrumbLine, "\n")
	lines += 1

	if len(m.replies) == 0 {
		lines += 1
	} else {
		cw := utils.ContentWidth(m.width)
		lines += 2

		end := utils.ScrollWindowEnd(m.scrollPos, m.pageSize, len(m.replies))
		for i := m.scrollPos; i < end; i++ {
			reply := m.replies[i]
			itemStr := bsk.FormatPostListItem(reply.Post, m.cursor == i, cw)
			lines += strings.Count(itemStr, "\n")
			if len(reply.Replies) > 0 {
				lines += 1
			}
			lines += 1
		}

		if m.scrollPos > 0 {
			lines += 1
		}
		if end < len(m.replies) {
			lines += 1
		}

		lines += 2

		idx := m.cursor
		if idx >= len(m.replies) {
			idx = 0
		}
		postText := bsk.FormatPost(bsk.FeedItem{PostInfo: m.replies[idx].Post, URI: m.replies[idx].URI}, -1, cw, utils.TextBudget(m.height))
		lines += strings.Count(postText, "\n")
	}

	return lines
}

func (m Model) openAttachment() tea.Msg {
	reply := m.replies[m.cursor]
	return attach.Open(reply.Post.Embeds, m.imgCursor)
}

func (m Model) View() tea.View {
	var b strings.Builder

	if m.statusMsg != "" && m.root == nil {
		utils.WriteHeader(&b, "Comments", m.width)
		b.WriteString(m.statusMsg + "\n")
		return tea.NewView(utils.CenterBlock(b.String(), m.width))
	}

	if m.loading {
		b.WriteString("Loading comments...\n")
		return tea.NewView(utils.CenterBlock(b.String(), m.width))
	}

	if m.root == nil {
		b.WriteString("No thread data.\n")
		return tea.NewView(utils.CenterBlock(b.String(), m.width))
	}

	utils.WriteHeader(&b, "Comments", m.width)

	const maxBreadcrumb = 5
	if len(m.breadcrumb) > maxBreadcrumb {
		b.WriteString("... > ")
		b.WriteString(strings.Join(m.breadcrumb[len(m.breadcrumb)-maxBreadcrumb:], " > "))
	} else {
		b.WriteString(strings.Join(m.breadcrumb, " > "))
	}
	b.WriteString("\n\n")

	if len(m.replies) == 0 {
		b.WriteString("  No replies yet.\n")
	} else {
		cw := utils.ContentWidth(m.width)
		b.WriteString(fmt.Sprintf("%d repl%s\n\n", len(m.replies), utils.Pluralize(len(m.replies), "y", "ies")))

		end := utils.ScrollWindowEnd(m.scrollPos, m.pageSize, len(m.replies))

		for i := m.scrollPos; i < end; i++ {
			reply := m.replies[i]
			b.WriteString(bsk.FormatPostListItem(reply.Post, m.cursor == i, cw))
			if len(reply.Replies) > 0 {
				b.WriteString(fmt.Sprintf("    [%d repl%s]\n", len(reply.Replies), utils.Pluralize(len(reply.Replies), "y", "ies")))
			}
			b.WriteString("\n")
		}

		bsk.WriteMoreIndicators(&b, m.scrollPos, end, len(m.replies))
		idx := m.cursor
		if idx >= len(m.replies) {
			idx = 0
		}
		b.WriteString(fmt.Sprintf("Reply %d/%d\n\n", idx+1, len(m.replies)))

		selected := m.replies[idx]
		cursor := -1
		if m.hasRendered {
			cursor = m.imgCursor
		}
		b.WriteString(bsk.FormatPost(bsk.FeedItem{PostInfo: selected.Post, URI: selected.URI}, cursor, cw, utils.TextBudget(m.height)))
	}

	if m.imageRows > 0 {
		b.WriteString(strings.Repeat("\n", m.imageRows))
	}

	if m.statusMsg != "" {
		b.WriteString(fmt.Sprintf("%s\n", m.statusMsg))
	}

	hasEmbeds := len(m.replies) > 0 && m.cursor < len(m.replies) && len(m.replies[m.cursor].Post.Embeds) > 0
	help := "\n[j/k] navigate  [enter] drill"
	if m.current != m.root {
		help += "  [h] parent"
	}
	if hasEmbeds {
		if m.hasRendered && len(m.replies[m.cursor].Post.Embeds) > 1 {
			help += "  [←/→] images"
		}
		help += "  [a] attachments  [o] open"
	}
	help += "  [esc] to feed  [q] quit\n"
	b.WriteString(help)

	return tea.NewView(utils.CenterBlock(b.String(), m.width))
}

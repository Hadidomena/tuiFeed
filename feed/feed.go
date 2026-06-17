package feed

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bluesky-social/indigo/xrpc"

	"github.com/Hadidomena/tuiFeed/bsk"
	"github.com/Hadidomena/tuiFeed/config"
)

type BackMsg struct{}

type postsLoadedMsg []bsk.FeedItem

type loadErrorMsg string

type imageRenderedMsg struct {
	imageRows int
	status    string
}

type Model struct {
	cfg         *config.Config
	client      *xrpc.Client
	posts       []bsk.FeedItem
	cursor      int
	imgCursor   int
	loading     bool
	statusMsg   string
	hasRendered bool
	imageRows   int
}

func NewModel() (Model, error) {
	cfg, err := config.Load()
	if err != nil {
		return Model{}, err
	}
	return Model{
		cfg:     cfg,
		client:  bsk.NewClient(),
		loading: true,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return m.loadPosts
}

func (m Model) loadPosts() tea.Msg {
	ctx := context.Background()

	if len(m.cfg.Follows) == 0 {
		return loadErrorMsg("No followed accounts. Add some in Manage follows.")
	}

	var allPosts []bsk.FeedItem
	for _, handle := range m.cfg.Follows {
		items, err := bsk.GetAuthorFeed(ctx, m.client, handle, 10)
		if err != nil {
			return loadErrorMsg(fmt.Sprintf("Error fetching @%s: %v", handle, err))
		}
		allPosts = append(allPosts, items...)
	}

	sort.Slice(allPosts, func(i, j int) bool {
		return allPosts[i].CreatedAt > allPosts[j].CreatedAt
	})

	return postsLoadedMsg(allPosts)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			bsk.ClearImages()
			return m, tea.Quit
		case "esc":
			m.hasRendered = false
			m.imageRows = 0
			bsk.ClearImages()
			return m, func() tea.Msg { return BackMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.imgCursor = 0
				m.hasRendered = false
				m.imageRows = 0
				m.statusMsg = ""
				bsk.ClearImages()
			}
		case "down", "j":
			if m.cursor < len(m.posts)-1 {
				m.cursor++
				m.imgCursor = 0
				m.hasRendered = false
				m.imageRows = 0
				m.statusMsg = ""
				bsk.ClearImages()
			}
		case "left", "h":
			if m.hasRendered && m.imgCursor > 0 {
				m.imgCursor--
				m.statusMsg = fmt.Sprintf("Rendering image %d/%d...", m.imgCursor+1, len(m.posts[m.cursor].Embeds))
				return m, m.renderAttachment
			}
		case "right", "l":
			if m.hasRendered && m.imgCursor < len(m.posts[m.cursor].Embeds)-1 {
				m.imgCursor++
				m.statusMsg = fmt.Sprintf("Rendering image %d/%d...", m.imgCursor+1, len(m.posts[m.cursor].Embeds))
				return m, m.renderAttachment
			}
		case "a":
			if m.cursor < len(m.posts) && len(m.posts[m.cursor].Embeds) > 0 {
				m.imgCursor = 0
				m.hasRendered = true
				m.statusMsg = fmt.Sprintf("Rendering image 1/%d...", len(m.posts[m.cursor].Embeds))
				return m, m.renderAttachment
			}
		case "o":
			if m.cursor < len(m.posts) && len(m.posts[m.cursor].Embeds) > 0 {
				m.statusMsg = "Opening image externally..."
				return m, m.openAttachment
			}
		case "r":
			m.loading = true
			m.statusMsg = ""
			m.hasRendered = false
			m.imageRows = 0
			bsk.ClearImages()
			return m, m.loadPosts
		}
	case postsLoadedMsg:
		m.posts = []bsk.FeedItem(msg)
		m.loading = false
		m.cursor = 0
		m.imgCursor = 0
		m.hasRendered = false
		m.imageRows = 0
		if len(msg) == 0 {
			m.statusMsg = "No posts found."
		}
	case loadErrorMsg:
		m.loading = false
		m.statusMsg = string(msg)
	case imageRenderedMsg:
		m.imageRows = msg.imageRows
		m.statusMsg = msg.status
	}

	return m, nil
}

func (m Model) renderAttachment() tea.Msg {
	post := m.posts[m.cursor]
	if m.imgCursor >= len(post.Embeds) || m.imgCursor < 0 {
		return loadErrorMsg("No image to render")
	}

	postText := bsk.FormatPost(post)
	postLines := strings.Count(postText, "\n")
	yOffset := 5 + postLines

	bsk.ClearImages()
	rows, err := bsk.RenderImage(post.Embeds[m.imgCursor], yOffset)
	if err != nil {
		return loadErrorMsg(fmt.Sprintf("Render failed: %v", err))
	}
	return imageRenderedMsg{
		imageRows: rows,
		status:    fmt.Sprintf("Image %d/%d  [←/→] navigate  [o] open externally", m.imgCursor+1, len(post.Embeds)),
	}
}

func (m Model) openAttachment() tea.Msg {
	post := m.posts[m.cursor]
	if m.imgCursor >= len(post.Embeds) || m.imgCursor < 0 {
		return loadErrorMsg("No image to open")
	}

	resp, err := http.Get(post.Embeds[m.imgCursor])
	if err != nil {
		return loadErrorMsg(fmt.Sprintf("Download failed: %v", err))
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return loadErrorMsg(fmt.Sprintf("Read failed: %v", err))
	}

	if err := bsk.RenderImageExternal(data); err != nil {
		return loadErrorMsg(fmt.Sprintf("Open failed: %v", err))
	}
	return loadErrorMsg("Opened externally")
}

func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString("Feed\n")
	b.WriteString(strings.Repeat("─", 30))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("Loading posts...\n")
		return tea.NewView(b.String())
	}

	if len(m.posts) == 0 {
		b.WriteString("No posts to display.\n")
	} else {
		idx := m.cursor
		if idx >= len(m.posts) {
			idx = 0
		}
		b.WriteString(fmt.Sprintf("Post %d/%d\n\n", idx+1, len(m.posts)))
		b.WriteString(bsk.FormatPost(m.posts[idx]))
	}

	if m.hasRendered && m.imageRows > 0 {
		b.WriteString(strings.Repeat("\n", m.imageRows))
	}

	if m.statusMsg != "" {
		b.WriteString(fmt.Sprintf("%s\n", m.statusMsg))
	}

	if m.hasRendered && len(m.posts) > 0 && m.cursor < len(m.posts) && len(m.posts[m.cursor].Embeds) > 1 {
		b.WriteString("\n[←/→] prev/next image  [o] open externally  [esc] back  [q] quit\n")
	} else {
		b.WriteString("\n[a] attachments  [o] open externally  [r] refresh  [esc] back  [q] quit\n")
	}
	return tea.NewView(b.String())
}

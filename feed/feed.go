package feed

import (
	"context"
	"fmt"
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

type Model struct {
	cfg       *config.Config
	client    *xrpc.Client
	posts     []bsk.FeedItem
	cursor    int
	loading   bool
	statusMsg string
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
			return m, tea.Quit
		case "esc":
			return m, func() tea.Msg { return BackMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.posts)-1 {
				m.cursor++
			}
		case "a":
			if m.cursor < len(m.posts) && len(m.posts[m.cursor].Embeds) > 0 {
				m.statusMsg = "Rendering images..."
				return m, m.renderAttachments
			}
		case "r":
			m.loading = true
			m.statusMsg = ""
			return m, m.loadPosts
		}
	case postsLoadedMsg:
		m.posts = []bsk.FeedItem(msg)
		m.loading = false
		m.cursor = 0
		if len(msg) == 0 {
			m.statusMsg = "No posts found."
		}
	case loadErrorMsg:
		m.loading = false
		m.statusMsg = string(msg)
	}

	return m, nil
}

func (m Model) renderAttachments() tea.Msg {
	post := m.posts[m.cursor]
	failed := 0
	for _, embed := range post.Embeds {
		if err := bsk.RenderImage(embed); err != nil {
			failed++
		}
	}
	if failed > 0 {
		return loadErrorMsg(fmt.Sprintf("%d/%d images failed to render", failed, len(post.Embeds)))
	}
	return loadErrorMsg("Images rendered")
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

	if m.statusMsg != "" {
		b.WriteString(fmt.Sprintf("%s\n", m.statusMsg))
	}

	b.WriteString("\n[a] attachments  [r] refresh  [esc] back  [q] quit\n")
	return tea.NewView(b.String())
}

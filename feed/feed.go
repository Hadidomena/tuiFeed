package feed

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bluesky-social/indigo/xrpc"

	"github.com/Hadidomena/tuiFeed/attach"
	"github.com/Hadidomena/tuiFeed/bsk"
	"github.com/Hadidomena/tuiFeed/config"
	"github.com/Hadidomena/tuiFeed/utils"
)

type BackMsg struct{}

type OpenThreadMsg struct {
	URI string
}

type postsLoadedMsg []bsk.FeedItem

type loadErrorMsg string

type Model struct {
	cfg         *config.Config
	client      *xrpc.Client
	posts       []bsk.FeedItem
	cursor      int
	scrollPos   int
	pageSize    int
	imgCursor   int
	loading     bool
	statusMsg   string
	hasRendered bool
	imageRows   int
	title       string
	isSavedView bool
	width       int
	height      int
}

func NewModel() (Model, error) {
	cfg, err := config.Load()
	if err != nil {
		return Model{}, err
	}
	return Model{
		cfg:      cfg,
		client:   bsk.NewClient(),
		loading:  true,
		pageSize: utils.DefaultPageSize,
		title:    "Feed",
		width:    utils.DefaultWidth,
		height:   24,
	}, nil
}

func NewStaticModel(posts []bsk.FeedItem, title string) Model {
	return Model{
		posts:    posts,
		pageSize: utils.DefaultPageSize,
		title:    title,
		width:    utils.DefaultWidth,
		height:   24,
	}
}

func (m Model) WithConfig(cfg *config.Config) Model {
	m.cfg = cfg
	return m
}

func (m Model) WithSize(w, h int) Model {
	m.width = w
	m.height = h
	m.pageSize = utils.PageSize(h)
	return m
}

func NewSavedModel(cfg *config.Config) Model {
	return Model{
		cfg:         cfg,
		client:      bsk.NewClient(),
		loading:     true,
		pageSize:    utils.DefaultPageSize,
		title:       "Saved posts",
		isSavedView: true,
		width:       utils.DefaultWidth,
		height:      24,
	}
}

func (m Model) Init() tea.Cmd {
	if m.client == nil {
		return nil
	}
	if m.isSavedView {
		return m.loadSavedPosts
	}
	return m.loadPosts
}

func (m Model) loadPosts() tea.Msg {
	if m.cfg == nil || m.client == nil {
		return loadErrorMsg("Not available in this view.")
	}
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

func (m Model) loadSavedPosts() tea.Msg {
	if m.cfg == nil || m.client == nil {
		return loadErrorMsg("Not available in this view.")
	}
	uris := m.cfg.GetSavedPostURIs()
	if len(uris) == 0 {
		return postsLoadedMsg(nil)
	}
	ctx := context.Background()
	posts, err := bsk.GetPosts(ctx, m.client, uris)
	if err != nil {
		return loadErrorMsg(fmt.Sprintf("Error loading saved posts: %v", err))
	}
	return postsLoadedMsg(posts)
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
			return m, func() tea.Msg { return BackMsg{} }
		case "down", "j":
			old := m.cursor
			oldScroll := m.scrollPos
			m.cursor, m.scrollPos = utils.ScrollDown(m.cursor, m.scrollPos, m.pageSize, len(m.posts))
			if m.cursor != old || m.scrollPos != oldScroll {
				m.imgCursor = 0
				m.hasRendered = false
				m.imageRows = 0
				m.statusMsg = ""
				bsk.ClearImages()
			}
		case "up", "k":
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
		case "left", "h":
			if m.hasRendered && m.imgCursor > 0 {
				m.imgCursor--
				return m, m.renderAttachment
			}
		case "right", "l":
			if m.hasRendered && m.imgCursor < len(m.posts[m.cursor].Embeds)-1 {
				m.imgCursor++
				return m, m.renderAttachment
			}
		case "a":
			if m.cursor < len(m.posts) && len(m.posts[m.cursor].Embeds) > 0 {
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
			if m.cursor < len(m.posts) && len(m.posts[m.cursor].Embeds) > 0 {
				if !m.hasRendered {
					m.hasRendered = true
					m.imgCursor = 0
				}
				m.statusMsg = "Opening image externally..."
				return m, m.openAttachment
			}
		case "s":
			if m.cfg == nil || len(m.posts) == 0 || m.cursor >= len(m.posts) {
				return m, nil
			}
			uri := m.posts[m.cursor].URI
			if uri == "" {
				m.statusMsg = "Cannot save post (no URI)"
				return m, nil
			}
			if m.isSavedView {
				fresh, err := config.ApplyUpdateAndReload(func(cfg *config.Config) {
					cfg.RemoveSavedPostByURI(uri)
				})
				if err != nil {
					m.statusMsg = fmt.Sprintf("Error removing saved post: %v", err)
					return m, nil
				}
				m.posts = append(m.posts[:m.cursor], m.posts[m.cursor+1:]...)
				if m.cursor >= len(m.posts) && m.cursor > 0 {
					m.cursor--
				}
				if m.scrollPos >= len(m.posts) {
					m.scrollPos = len(m.posts) - m.pageSize
					if m.scrollPos < 0 {
						m.scrollPos = 0
					}
				}
				m.hasRendered = false
				m.imageRows = 0
				bsk.ClearImages()
				m.statusMsg = "Removed from saved"
				if len(m.posts) == 0 {
					m.statusMsg = "No saved posts"
				}
				m.cfg = fresh
				m = m.recalcPageSize()
				return m, nil
			}
			if m.cfg.IsSaved(uri) {
				if err := config.Update(func(cfg *config.Config) {
					cfg.RemoveSavedPostByURI(uri)
				}); err != nil {
					m.statusMsg = fmt.Sprintf("Error unsaving post: %v", err)
					return m, nil
				}
				m.statusMsg = "Unsaved"
			} else {
				if err := config.Update(func(cfg *config.Config) {
					cfg.SavePost(uri)
				}); err != nil {
					m.statusMsg = fmt.Sprintf("Error saving post: %v", err)
					return m, nil
				}
				m.statusMsg = "Saved!"
			}
			fresh, err := config.Load()
			if err != nil {
				m.statusMsg = fmt.Sprintf("Saved, but config reload failed: %v", err)
				return m, nil
			}
			m.cfg = fresh
		case "c":
			if len(m.posts) > 0 && m.cursor < len(m.posts) {
				uri := m.posts[m.cursor].URI
				if uri != "" {
					return m, func() tea.Msg {
						return OpenThreadMsg{URI: uri}
					}
				}
			}
		case "r":
			if m.isSavedView {
				return m, nil
			}
			m.loading = true
			m.statusMsg = ""
			m.hasRendered = false
			m.imageRows = 0
			m.scrollPos = 0
			bsk.ClearImages()
			return m, m.loadPosts
		}
	case postsLoadedMsg:
		m.posts = []bsk.FeedItem(msg)
		m.loading = false
		m.cursor = 0
		m.imgCursor = 0
		m.scrollPos = 0
		m.hasRendered = false
		m.imageRows = 0
		if len(msg) == 0 {
			m.statusMsg = "No posts found."
		}
	case loadErrorMsg:
		m.loading = false
		m.statusMsg = string(msg)
	case attach.ErrorMsg:
		m.statusMsg = string(msg)
	case attach.RenderedMsg:
		m.imageRows = msg.ImageRows
		m.statusMsg = msg.Status
	}

	m = m.recalcPageSize()
	return m, nil
}

func (m Model) recalcPageSize() Model {
	if m.height <= 0 || len(m.posts) == 0 {
		m.pageSize = utils.DefaultPageSize
		return m
	}

	fixedBefore := 5

	idx := m.cursor
	if idx >= len(m.posts) {
		idx = 0
	}
	cw := utils.ContentWidth(m.width)
	postText := bsk.FormatPost(m.posts[idx], -1, cw)
	postLines := strings.Count(postText, "\n")

	afterList := 2 + postLines + m.imageRows + 2
	if m.statusMsg != "" {
		afterList += 1
	}

	available := m.height - fixedBefore - afterList
	if available < 4 {
		m.pageSize = 1
	} else {
		m.pageSize = available / 4
	}
	if m.pageSize < 1 {
		m.pageSize = 1
	}
	return m
}

func (m Model) renderAttachment() tea.Msg {
	if len(m.posts) == 0 || m.cursor >= len(m.posts) {
		return attach.ErrorMsg("No post selected")
	}
	yOffset := m.computeYOffset()
	maxCols := attach.ComputeMaxCols(m.width)
	maxRows := attach.ComputeMaxRows(m.height, yOffset)
	return attach.Render(m.posts[m.cursor].Embeds, m.imgCursor, yOffset, maxCols, maxRows)
}

func (m Model) computeYOffset() int {
	if len(m.posts) == 0 {
		return 4
	}

	cw := utils.ContentWidth(m.width)

	lines := 0

	lines += 3

	lines += 2

	end := utils.ScrollWindowEnd(m.scrollPos, m.pageSize, len(m.posts))
	for i := m.scrollPos; i < end; i++ {
		lines += strings.Count(bsk.FormatPostListItem(m.posts[i].PostInfo, m.cursor == i, cw), "\n")
		lines += 1
	}

	if m.scrollPos > 0 {
		lines += 1
	}
	if end < len(m.posts) {
		lines += 1
	}

	lines += 2

	idx := m.cursor
	if idx >= len(m.posts) {
		idx = 0
	}
	postText := bsk.FormatPost(m.posts[idx], -1, cw)
	lines += strings.Count(postText, "\n")

	return lines
}

func (m Model) openAttachment() tea.Msg {
	post := m.posts[m.cursor]
	return attach.Open(post.Embeds, m.imgCursor)
}

func (m Model) View() tea.View {
	var b strings.Builder

	utils.WriteHeader(&b, m.title, m.width)

	if m.loading {
		if m.isSavedView {
			b.WriteString("Loading saved posts...\n")
		} else {
			b.WriteString("Loading posts...\n")
		}
		return tea.NewView(b.String())
	}

	if len(m.posts) == 0 {
		b.WriteString("No posts to display.\n")
	} else {
		cw := utils.ContentWidth(m.width)
		b.WriteString(fmt.Sprintf("%d posts\n\n", len(m.posts)))

		end := utils.ScrollWindowEnd(m.scrollPos, m.pageSize, len(m.posts))

		for i := m.scrollPos; i < end; i++ {
			b.WriteString(bsk.FormatPostListItem(m.posts[i].PostInfo, m.cursor == i, cw))
			b.WriteString("\n")
		}

		bsk.WriteMoreIndicators(&b, m.scrollPos, end, len(m.posts))
		idx := m.cursor
		if idx >= len(m.posts) {
			idx = 0
		}
		b.WriteString(fmt.Sprintf("Post %d/%d\n\n", idx+1, len(m.posts)))
		cursor := -1
		if m.hasRendered {
			cursor = m.imgCursor
		}
		b.WriteString(bsk.FormatPost(m.posts[idx], cursor, cw))
	}

	if m.hasRendered && m.imageRows > 0 {
		b.WriteString(strings.Repeat("\n", m.imageRows))
	}

	if m.statusMsg != "" {
		b.WriteString(fmt.Sprintf("%s\n", m.statusMsg))
	}

	if m.isSavedView {
		if m.hasRendered && len(m.posts) > 0 && m.cursor < len(m.posts) && len(m.posts[m.cursor].Embeds) > 1 {
			b.WriteString("\n[←/→] prev/next image  [s] remove  [a] attachments  [o] open externally  [esc] back  [q] quit\n")
		} else {
			b.WriteString("\n[s] remove  [a] attachments  [o] open externally  [esc] back  [q] quit\n")
		}
	} else if m.hasRendered && len(m.posts) > 0 && m.cursor < len(m.posts) && len(m.posts[m.cursor].Embeds) > 1 {
		b.WriteString("\n[←/→] prev/next image  [c] comments  [o] open externally  [s] save  [r] refresh  [esc] back  [q] quit\n")
	} else {
		b.WriteString("\n[c] comments  [s] save  [a] attachments  [o] open externally  [r] refresh  [esc] back  [q] quit\n")
	}
	return tea.NewView(utils.CenterBlock(b.String(), m.width))
}

package main

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Hadidomena/tuiFeed/bsk"
	"github.com/Hadidomena/tuiFeed/config"
	"github.com/Hadidomena/tuiFeed/feed"
	"github.com/Hadidomena/tuiFeed/follows"
	"github.com/Hadidomena/tuiFeed/rss"
	"github.com/Hadidomena/tuiFeed/rssfeed"
	"github.com/Hadidomena/tuiFeed/rssfeeds"
	"github.com/Hadidomena/tuiFeed/thread"
	"github.com/Hadidomena/tuiFeed/utils"
)

type sessionState int

const (
	showDashboardView sessionState = iota
	showFollowsView
	showFeedView
	showAccountSelectView
	showSinceLastCheckView
	showLoadingView
	showSavedPostsView
	showThreadView
	showRSSView
	showRSSFeedSelectView
	showRSSSinceLastCheckView
	showRSSManageView
	showSavedRSSView
)

type OpenFollowsMsg struct{}
type OpenFeedMsg struct{}
type OpenAccountSelectMsg struct{}
type OpenSavedPostsMsg struct{}
type OpenRSSMsg struct{}
type OpenRSSManageMsg struct{}
type OpenRSSSelectMsg struct{}
type OpenSavedRSSMsg struct{}
type BackToDashboardMsg struct{}

type SelectAccountForSinceLastCheck struct {
	handle string
}

type SelectRSSFeedForSinceLastCheck struct {
	url string
}

type PostsFetchedMsg struct {
	handle string
	posts  []bsk.FeedItem
	err    error
}

type RSSEntriesFetchedMsg struct {
	url     string
	entries []rss.Entry
	err     error
}

type DashboardModel struct {
	cursor  int
	choices []string
	width   int
	height  int
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{
		choices: []string{"View Feed", "Posts since last check", "Manage follows", "Saved posts", "View RSS feeds", "RSS since last check", "Manage RSS feeds", "Saved RSS entries"},
		width:   utils.DefaultWidth,
		height:  24,
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return nil
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.cursor = utils.CursorUp(m.cursor)
		case "down", "j":
			m.cursor = utils.CursorDown(m.cursor, len(m.choices))
		case "enter", " ", "space":
			if m.cursor == 0 {
				return m, func() tea.Msg { return OpenFeedMsg{} }
			}
			if m.cursor == 1 {
				return m, func() tea.Msg { return OpenAccountSelectMsg{} }
			}
			if m.cursor == 2 {
				return m, func() tea.Msg { return OpenFollowsMsg{} }
			}
			if m.cursor == 3 {
				return m, func() tea.Msg { return OpenSavedPostsMsg{} }
			}
			if m.cursor == 4 {
				return m, func() tea.Msg { return OpenRSSMsg{} }
			}
			if m.cursor == 5 {
				return m, func() tea.Msg { return OpenRSSSelectMsg{} }
			}
			if m.cursor == 6 {
				return m, func() tea.Msg { return OpenRSSManageMsg{} }
			}
			if m.cursor == 7 {
				return m, func() tea.Msg { return OpenSavedRSSMsg{} }
			}
		}
	}
	return m, nil
}

func (m DashboardModel) View() tea.View {
	var b strings.Builder
	utils.WriteHeader(&b, "Bluesky TUI Feed", m.width)
	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("%s  %s\n", cursor, choice))
	}
	b.WriteString("\nPress q to quit.\n")
	return tea.NewView(utils.CenterBlock(b.String(), m.width))
}

type AccountSelectModel struct {
	accounts   []string
	lastChecks map[string]string
	cursor     int
	width      int
	height     int
}

func NewAccountSelectModel(cfg *config.Config) AccountSelectModel {
	return AccountSelectModel{
		accounts:   cfg.Follows,
		lastChecks: cfg.LastChecks,
		width:      utils.DefaultWidth,
		height:     24,
	}
}

func (m AccountSelectModel) Init() tea.Cmd {
	return nil
}

func (m AccountSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackToDashboardMsg{} }
		case "up", "k":
			m.cursor = utils.CursorUp(m.cursor)
		case "down", "j":
			m.cursor = utils.CursorDown(m.cursor, len(m.accounts))
		case "enter":
			if len(m.accounts) > 0 {
				return m, func() tea.Msg {
					return SelectAccountForSinceLastCheck{handle: m.accounts[m.cursor]}
				}
			}
		}
	}
	return m, nil
}

func (m AccountSelectModel) View() tea.View {
	var b strings.Builder

	utils.WriteHeader(&b, "Posts since last check", m.width)

	if len(m.accounts) == 0 {
		b.WriteString("  No accounts followed.\n")
		b.WriteString("  Add accounts in 'Manage follows' first.\n\n")
	} else {
		for i, handle := range m.accounts {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}
			lastStr := "never"
			if t, ok := m.lastChecks[handle]; ok && t != "" {
				if len(t) >= 10 {
					lastStr = t[:10]
				}
			}
			b.WriteString(fmt.Sprintf("%s@%s  (last: %s)\n", cursor, handle, lastStr))
		}
	}

	b.WriteString("\n[\u2191/\u2193] navigate  [enter] select  [esc] back\n")
	return tea.NewView(utils.CenterBlock(b.String(), m.width))
}

type RSSFeedSelectModel struct {
	feeds      []string
	lastChecks map[string]string
	cursor     int
	width      int
	height     int
}

func NewRSSFeedSelectModel(cfg *config.Config) RSSFeedSelectModel {
	return RSSFeedSelectModel{
		feeds:      cfg.RSSFeeds,
		lastChecks: cfg.RSSLastChecks,
		width:      utils.DefaultWidth,
		height:     24,
	}
}

func (m RSSFeedSelectModel) Init() tea.Cmd {
	return nil
}

func (m RSSFeedSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackToDashboardMsg{} }
		case "up", "k":
			m.cursor = utils.CursorUp(m.cursor)
		case "down", "j":
			m.cursor = utils.CursorDown(m.cursor, len(m.feeds))
		case "enter":
			if len(m.feeds) > 0 {
				return m, func() tea.Msg {
					return SelectRSSFeedForSinceLastCheck{url: m.feeds[m.cursor]}
				}
			}
		}
	}
	return m, nil
}

func (m RSSFeedSelectModel) View() tea.View {
	var b strings.Builder

	utils.WriteHeader(&b, "RSS since last check", m.width)

	if len(m.feeds) == 0 {
		b.WriteString("  No RSS feeds subscribed.\n")
		b.WriteString("  Add feeds in 'Manage RSS feeds' first.\n\n")
	} else {
		for i, url := range m.feeds {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}
			lastStr := "never"
			if t, ok := m.lastChecks[url]; ok && t != "" {
				if len(t) >= 10 {
					lastStr = t[:10]
				}
			}
			fmt.Fprintf(&b, "%s%s  (last: %s)\n", cursor, url, lastStr)
		}
	}

	b.WriteString("\n[\u2191/\u2193] navigate  [enter] select  [esc] back\n")
	return tea.NewView(utils.CenterBlock(b.String(), m.width))
}

type LoadingModel struct {
	message string
	width   int
	height  int
}

func (m LoadingModel) Init() tea.Cmd {
	return nil
}

func (m LoadingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		return m, nil
	}
	return m, nil
}

func (m LoadingModel) View() tea.View {
	s := fmt.Sprintf("\n  %s\n\n", m.message)
	return tea.NewView(utils.CenterBlock(s, m.width))
}

type MainModel struct {
	state         sessionState
	cfg           *config.Config
	dashboard     DashboardModel
	follows       follows.Model
	accountSelect AccountSelectModel
	rssFeedSelect RSSFeedSelectModel
	feed          feed.Model
	thread        thread.Model
	rssfeed       rssfeed.Model
	rssfeeds      rssfeeds.Model
	loading       LoadingModel
	width         int
	height        int
}

func NewMainModel() MainModel {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	width := utils.DefaultWidth
	height := 24
	fm, err := follows.NewModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing follows: %v\n", err)
		os.Exit(1)
	}
	fm = fm.WithSize(width, height)
	fd, err := feed.NewModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing feed: %v\n", err)
		os.Exit(1)
	}
	fd = fd.WithSize(width, height)
	return MainModel{
		state:     showDashboardView,
		cfg:       cfg,
		dashboard: NewDashboardModel(),
		follows:   fm,
		feed:      fd,
		loading:   LoadingModel{message: "Fetching posts...", width: width, height: height},
		width:     width,
		height:    height,
	}
}

func (m MainModel) Init() tea.Cmd {
	return m.dashboard.Init()
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case OpenFollowsMsg:
		m.state = showFollowsView
		fm, err := follows.NewModel()
		if err == nil {
			m.follows = fm.WithSize(m.width, m.height)
		}
		return m, m.follows.Init()
	case follows.BackMsg:
		m.state = showDashboardView
		m.cfg, _ = config.Load()
		m.dashboard.width = m.width
		m.dashboard.height = m.height
		return m, nil
	case OpenFeedMsg:
		m.state = showFeedView
		fd, err := feed.NewModel()
		if err == nil {
			m.feed = fd.WithSize(m.width, m.height)
		}
		return m, m.feed.Init()
	case OpenSavedPostsMsg:
		m.state = showSavedPostsView
		m.cfg, _ = config.Load()
		m.feed = feed.NewSavedModel(m.cfg).WithSize(m.width, m.height)
		return m, m.feed.Init()
	case feed.BackMsg:
		m.state = showDashboardView
		m.dashboard.width = m.width
		m.dashboard.height = m.height
		return m, nil
	case feed.OpenThreadMsg:
		m.state = showThreadView
		m.thread = thread.NewModel(msg.URI).WithSize(m.width, m.height)
		return m, m.thread.Init()
	case thread.BackMsg:
		m.state = showFeedView
		m.feed = m.feed.WithSize(m.width, m.height)
		return m, nil
	case OpenAccountSelectMsg:
		m.state = showAccountSelectView
		m.cfg, _ = config.Load()
		m.accountSelect = NewAccountSelectModel(m.cfg)
		m.accountSelect.width = m.width
		m.accountSelect.height = m.height
		return m, m.accountSelect.Init()
	case BackToDashboardMsg:
		m.state = showDashboardView
		m.dashboard.width = m.width
		m.dashboard.height = m.height
		return m, nil
	case SelectAccountForSinceLastCheck:
		m.state = showLoadingView
		m.loading = LoadingModel{message: fmt.Sprintf("Fetching posts for @%s...", msg.handle), width: m.width, height: m.height}
		m.cfg, _ = config.Load()
		lastCheck := ""
		if m.cfg.LastChecks != nil {
			lastCheck = m.cfg.LastChecks[msg.handle]
		}
		return m, fetchSinceLastCheckCmd(msg.handle, lastCheck)
	case PostsFetchedMsg:
		m.state = showSinceLastCheckView
		m.cfg, _ = config.Load()
		if msg.err != nil {
			m.feed = feed.NewStaticModel(nil, fmt.Sprintf("Error: %v", msg.err)).WithSize(m.width, m.height)
		} else {
			m.feed = feed.NewStaticModel(msg.posts, fmt.Sprintf("Posts since last check \u2014 @%s", msg.handle)).WithConfig(m.cfg).WithSize(m.width, m.height)
		}
		return m, m.feed.Init()
	case OpenRSSMsg:
		m.state = showRSSView
		rf, err := rssfeed.NewModel()
		if err == nil {
			m.rssfeed = rf.WithSize(m.width, m.height)
		}
		return m, m.rssfeed.Init()
	case OpenRSSManageMsg:
		m.state = showRSSManageView
		rf, err := rssfeeds.NewModel()
		if err == nil {
			m.rssfeeds = rf.WithSize(m.width, m.height)
		}
		return m, m.rssfeeds.Init()
	case OpenRSSSelectMsg:
		m.state = showRSSFeedSelectView
		m.cfg, _ = config.Load()
		m.rssFeedSelect = NewRSSFeedSelectModel(m.cfg)
		m.rssFeedSelect.width = m.width
		m.rssFeedSelect.height = m.height
		return m, m.rssFeedSelect.Init()
	case OpenSavedRSSMsg:
		m.state = showSavedRSSView
		m.cfg, _ = config.Load()
		m.rssfeed = rssfeed.NewSavedModel(m.cfg).WithSize(m.width, m.height)
		return m, m.rssfeed.Init()
	case rssfeed.BackMsg, rssfeeds.BackMsg:
		m.state = showDashboardView
		m.dashboard.width = m.width
		m.dashboard.height = m.height
		return m, nil
	case SelectRSSFeedForSinceLastCheck:
		m.state = showLoadingView
		m.loading = LoadingModel{message: fmt.Sprintf("Fetching entries from %s...", msg.url), width: m.width, height: m.height}
		m.cfg, _ = config.Load()
		lastCheck := ""
		if m.cfg.RSSLastChecks != nil {
			lastCheck = m.cfg.RSSLastChecks[msg.url]
		}
		return m, fetchRSSSinceLastCheckCmd(msg.url, lastCheck)
	case RSSEntriesFetchedMsg:
		m.state = showRSSSinceLastCheckView
		m.cfg, _ = config.Load()
		if msg.err != nil {
			m.rssfeed = rssfeed.NewStaticModel(nil, fmt.Sprintf("Error: %v", msg.err)).WithSize(m.width, m.height)
		} else {
			m.rssfeed = rssfeed.NewStaticModel(msg.entries, fmt.Sprintf("RSS since last check \u2014 %s", msg.url)).WithConfig(m.cfg).WithSize(m.width, m.height)
		}
		return m, m.rssfeed.Init()
	}

	switch m.state {
	case showDashboardView:
		m.dashboard, cmd = updateSubModel(m.dashboard, msg)
	case showFollowsView:
		m.follows, cmd = updateSubModel(m.follows, msg)
	case showFeedView:
		m.feed, cmd = updateSubModel(m.feed, msg)
	case showAccountSelectView:
		m.accountSelect, cmd = updateSubModel(m.accountSelect, msg)
	case showRSSFeedSelectView:
		m.rssFeedSelect, cmd = updateSubModel(m.rssFeedSelect, msg)
	case showSinceLastCheckView:
		m.feed, cmd = updateSubModel(m.feed, msg)
	case showSavedPostsView:
		m.feed, cmd = updateSubModel(m.feed, msg)
	case showThreadView:
		m.thread, cmd = updateSubModel(m.thread, msg)
	case showRSSView:
		m.rssfeed, cmd = updateSubModel(m.rssfeed, msg)
	case showRSSSinceLastCheckView:
		m.rssfeed, cmd = updateSubModel(m.rssfeed, msg)
	case showSavedRSSView:
		m.rssfeed, cmd = updateSubModel(m.rssfeed, msg)
	case showRSSManageView:
		m.rssfeeds, cmd = updateSubModel(m.rssfeeds, msg)
	}

	return m, cmd
}

func updateSubModel[M tea.Model](model M, msg tea.Msg) (M, tea.Cmd) {
	updated, cmd := model.Update(msg)
	return updated.(M), cmd
}

func (m MainModel) View() tea.View {
	switch m.state {
	case showDashboardView:
		return m.dashboard.View()
	case showFollowsView:
		return m.follows.View()
	case showFeedView:
		return m.feed.View()
	case showAccountSelectView:
		return m.accountSelect.View()
	case showRSSFeedSelectView:
		return m.rssFeedSelect.View()
	case showSinceLastCheckView:
		return m.feed.View()
	case showSavedPostsView:
		return m.feed.View()
	case showThreadView:
		return m.thread.View()
	case showRSSView:
		return m.rssfeed.View()
	case showRSSSinceLastCheckView:
		return m.rssfeed.View()
	case showSavedRSSView:
		return m.rssfeed.View()
	case showRSSManageView:
		return m.rssfeeds.View()
	case showLoadingView:
		return m.loading.View()
	default:
		return tea.NewView("Unknown state")
	}
}

func fetchSinceLastCheckCmd(handle string, lastCheck string) tea.Cmd {
	return func() tea.Msg {
		client := bsk.NewClient()
		posts, _, err := bsk.GetAuthorFeedCursor(context.Background(), client, handle, "", 50)
		if err != nil {
			return PostsFetchedMsg{handle: handle, err: err}
		}
		if err := config.Update(func(cfg *config.Config) {
			cfg.SetLastCheck(handle)
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to update last-check for %s: %v\n", handle, err)
		}
		if lastCheck == "" {
			return PostsFetchedMsg{handle: handle, posts: posts}
		}
		filtered := make([]bsk.FeedItem, 0, len(posts))
		for _, p := range posts {
			if p.IndexedAt > lastCheck {
				filtered = append(filtered, p)
			}
		}
		return PostsFetchedMsg{handle: handle, posts: filtered}
	}
}

func fetchRSSSinceLastCheckCmd(url string, lastCheck string) tea.Cmd {
	return func() tea.Msg {
		entries, err := rss.FetchFeed(context.Background(), url)
		if err != nil {
			return RSSEntriesFetchedMsg{url: url, err: err}
		}
		if err := config.Update(func(cfg *config.Config) {
			cfg.SetRSSLastCheck(url)
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to update last-check for %s: %v\n", url, err)
		}
		if lastCheck == "" {
			return RSSEntriesFetchedMsg{url: url, entries: entries}
		}
		filtered := make([]rss.Entry, 0, len(entries))
		for _, e := range entries {
			if e.Published > lastCheck {
				filtered = append(filtered, e)
			}
		}
		return RSSEntriesFetchedMsg{url: url, entries: filtered}
	}
}

func main() {
	p := tea.NewProgram(NewMainModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

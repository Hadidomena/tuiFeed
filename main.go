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
)

type OpenFollowsMsg struct{}
type OpenFeedMsg struct{}
type OpenAccountSelectMsg struct{}
type OpenSavedPostsMsg struct{}
type BackToDashboardMsg struct{}

type SelectAccountForSinceLastCheck struct {
	handle string
}

type PostsFetchedMsg struct {
	handle string
	posts  []bsk.FeedItem
	err    error
}

type DashboardModel struct {
	cursor  int
	choices []string
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{
		choices: []string{"View Feed", "Posts since last check", "Manage follows", "Saved posts"},
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return nil
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
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
		}
	}
	return m, nil
}

func (m DashboardModel) View() tea.View {
	s := "Bluesky TUI Feed\n\n"
	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		s += fmt.Sprintf("%s  %s\n", cursor, choice)
	}
	s += "\nPress q to quit.\n"
	return tea.NewView(s)
}

type AccountSelectModel struct {
	accounts   []string
	lastChecks map[string]string
	cursor     int
}

func NewAccountSelectModel(cfg *config.Config) AccountSelectModel {
	return AccountSelectModel{
		accounts:   cfg.Follows,
		lastChecks: cfg.LastChecks,
	}
}

func (m AccountSelectModel) Init() tea.Cmd {
	return nil
}

func (m AccountSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackToDashboardMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.accounts)-1 {
				m.cursor++
			}
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

	b.WriteString("Posts since last check\n")
	b.WriteString(strings.Repeat("\u2500", 40) + "\n\n")

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
	return tea.NewView(b.String())
}

type LoadingModel struct {
	message string
}

func (m LoadingModel) Init() tea.Cmd {
	return nil
}

func (m LoadingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyPressMsg:
		return m, nil
	}
	return m, nil
}

func (m LoadingModel) View() tea.View {
	return tea.NewView(fmt.Sprintf("\n  %s\n\n", m.message))
}

type MainModel struct {
	state         sessionState
	cfg           *config.Config
	dashboard     DashboardModel
	follows       follows.Model
	accountSelect AccountSelectModel
	feed          feed.Model
	loading       LoadingModel
}

func NewMainModel() MainModel {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	fm, err := follows.NewModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing follows: %v\n", err)
		os.Exit(1)
	}
	fd, err := feed.NewModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing feed: %v\n", err)
		os.Exit(1)
	}
	return MainModel{
		state:     showDashboardView,
		cfg:       cfg,
		dashboard: NewDashboardModel(),
		follows:   fm,
		feed:      fd,
		loading:   LoadingModel{message: "Fetching posts..."},
	}
}

func (m MainModel) Init() tea.Cmd {
	return m.dashboard.Init()
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case OpenFollowsMsg:
		m.state = showFollowsView
		fm, err := follows.NewModel()
		if err == nil {
			m.follows = fm
		}
		return m, m.follows.Init()
	case follows.BackMsg:
		m.state = showDashboardView
		m.cfg, _ = config.Load()
		return m, nil
	case OpenFeedMsg:
		m.state = showFeedView
		fd, err := feed.NewModel()
		if err == nil {
			m.feed = fd
		}
		return m, m.feed.Init()
	case OpenSavedPostsMsg:
		m.state = showSavedPostsView
		m.cfg, _ = config.Load()
		m.feed = feed.NewSavedModel(m.cfg)
		return m, m.feed.Init()
	case feed.BackMsg:
		m.state = showDashboardView
		return m, nil
	case OpenAccountSelectMsg:
		m.state = showAccountSelectView
		m.cfg, _ = config.Load()
		m.accountSelect = NewAccountSelectModel(m.cfg)
		return m, m.accountSelect.Init()
	case BackToDashboardMsg:
		m.state = showDashboardView
		return m, nil
	case SelectAccountForSinceLastCheck:
		m.state = showLoadingView
		m.loading = LoadingModel{message: fmt.Sprintf("Fetching posts for @%s...", msg.handle)}
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
			m.feed = feed.NewStaticModel(nil, fmt.Sprintf("Error: %v", msg.err))
		} else {
			m.feed = feed.NewStaticModel(msg.posts, fmt.Sprintf("Posts since last check \u2014 @%s", msg.handle)).WithConfig(m.cfg)
		}
		return m, m.feed.Init()
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
	case showSinceLastCheckView:
		m.feed, cmd = updateSubModel(m.feed, msg)
	case showSavedPostsView:
		m.feed, cmd = updateSubModel(m.feed, msg)
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
	case showSinceLastCheckView:
		return m.feed.View()
	case showSavedPostsView:
		return m.feed.View()
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
		_ = config.Update(func(cfg *config.Config) {
			cfg.SetLastCheck(handle)
		})
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

func main() {
	p := tea.NewProgram(NewMainModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

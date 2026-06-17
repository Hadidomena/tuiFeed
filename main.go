package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/feed"
	"github.com/Hadidomena/tuiFeed/follows"
)

type sessionState int

const (
	showDashboardView sessionState = iota
	showFollowsView
	showFeedView
)

type OpenFollowsMsg struct{}
type OpenFeedMsg struct{}

type DashboardModel struct {
	cursor  int
	choices []string
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{
		choices: []string{"View Feed", "Manage follows"},
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
		case "enter", "space":
			switch m.cursor {
			case 0:
				return m, func() tea.Msg { return OpenFeedMsg{} }
			case 1:
				return m, func() tea.Msg { return OpenFollowsMsg{} }
			}
		}
	}
	return m, nil
}

func (m DashboardModel) View() tea.View {
	s := "tuiFeed\n\n"
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s  %s\n", cursor, choice)
	}
	s += "\nPress q to quit.\n"
	return tea.NewView(s)
}

// MainModel stores Types of sub-models
type MainModel struct {
	state     sessionState
	dashboard DashboardModel
	follows   follows.Model
	feed      feed.Model
}

func NewMainModel() MainModel {
	fm, err := follows.NewModel()
	if err != nil {
		fm, _ = follows.NewModel()
	}
	fd, err := feed.NewModel()
	if err != nil {
		fd, _ = feed.NewModel()
	}
	return MainModel{
		state:     showDashboardView,
		dashboard: NewDashboardModel(),
		follows:   fm,
		feed:      fd,
	}
}

func (m MainModel) Init() tea.Cmd {
	return m.dashboard.Init()
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.(type) {
	case OpenFollowsMsg:
		m.state = showFollowsView
		fm, err := follows.NewModel()
		if err == nil {
			m.follows = fm
		}
		return m, m.follows.Init()
	case follows.BackMsg:
		m.state = showDashboardView
		return m, nil
	case OpenFeedMsg:
		m.state = showFeedView
		fd, err := feed.NewModel()
		if err == nil {
			m.feed = fd
		}
		return m, m.feed.Init()
	case feed.BackMsg:
		m.state = showDashboardView
		return m, nil
	}

	switch m.state {
	case showDashboardView:
		updatedModel, dashboardCmd := m.dashboard.Update(msg)
		m.dashboard = updatedModel.(DashboardModel)
		cmd = dashboardCmd
	case showFollowsView:
		updatedModel, followsCmd := m.follows.Update(msg)
		m.follows = updatedModel.(follows.Model)
		cmd = followsCmd
	case showFeedView:
		updatedModel, feedCmd := m.feed.Update(msg)
		m.feed = updatedModel.(feed.Model)
		cmd = feedCmd
	}

	return m, cmd
}

func (m MainModel) View() tea.View {
	switch m.state {
	case showDashboardView:
		return m.dashboard.View()
	case showFollowsView:
		return m.follows.View()
	case showFeedView:
		return m.feed.View()
	default:
		return tea.NewView("Unknown state")
	}
}

func main() {
	p := tea.NewProgram(NewMainModel())
	feed.Program = p
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

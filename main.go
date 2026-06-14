package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/follows"
)

type sessionState int

const (
	showDashboardView sessionState = iota
	showFollowsView
	// showFormView
)

type OpenFollowsMsg struct{}

type DashboardModel struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{
		choices:  []string{"Manage follows", "Buy celery", "Buy kohlrabi"},
		selected: make(map[int]struct{}),
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
			if m.cursor == 0 {
				return m, func() tea.Msg { return OpenFollowsMsg{} }
			}
			if _, ok := m.selected[m.cursor]; ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}
	return m, nil
}

func (m DashboardModel) View() tea.View {
	s := "What should we buy at the market?\n\n"
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}
	s += "\nPress q to quit.\n"
	return tea.NewView(s)
}

// MainModel stores Types of sub-models
type MainModel struct {
	state     sessionState
	dashboard DashboardModel
	follows   follows.Model
}

func NewMainModel() MainModel {
	fm, err := follows.NewModel()
	if err != nil {
		fm, _ = follows.NewModel()
	}
	return MainModel{
		state:     showDashboardView,
		dashboard: NewDashboardModel(),
		follows:   fm,
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
	}

	return m, cmd
}

func (m MainModel) View() tea.View {
	switch m.state {
	case showDashboardView:
		return m.dashboard.View()
	case showFollowsView:
		return m.follows.View()
	default:
		return tea.NewView("Unknown state")
	}
}

func main() {
	p := tea.NewProgram(NewMainModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

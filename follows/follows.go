package follows

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/config"
)

type mode int

const (
	modeList mode = iota
	modeInput
)

type BackMsg struct{}

type Model struct {
	cfg       *config.Config
	cursor    int
	mode      mode
	input     string
	statusMsg string
}

func NewModel() (Model, error) {
	cfg, err := config.Load()
	if err != nil {
		return Model{}, err
	}
	return Model{cfg: cfg}, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeInput:
			return m.updateInput(msg)
		}
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return BackMsg{} }
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.cfg.Follows)-1 {
			m.cursor++
		}
	case "a":
		m.mode = modeInput
		m.input = ""
		m.statusMsg = ""
	case "d":
		if len(m.cfg.Follows) > 0 {
			removed := m.cfg.Follows[m.cursor]
			handle := m.cfg.Follows[m.cursor]
			_ = config.Update(func(cfg *config.Config) {
				idx := -1
				for i, h := range cfg.Follows {
					if h == handle {
						idx = i
						break
					}
				}
				if idx >= 0 {
					cfg.RemoveFollow(idx)
				}
			})
			fresh, err := config.Load()
			if err == nil {
				m.cfg = fresh
			}
			if m.cursor >= len(m.cfg.Follows) && m.cursor > 0 {
				m.cursor--
			}
			m.statusMsg = fmt.Sprintf("Removed @%s", removed)
		}
	}
	return m, nil
}

func (m Model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.input = ""
		m.statusMsg = ""
	case "enter":
		handle := strings.TrimSpace(m.input)
		if handle == "" {
			m.statusMsg = "Handle cannot be empty"
			m.mode = modeList
			return m, nil
		}
		handle = strings.TrimPrefix(handle, "@")
		_ = config.Update(func(cfg *config.Config) {
			cfg.AddFollow(handle)
		})
		fresh, err := config.Load()
		if err == nil {
			m.cfg = fresh
		}
		m.statusMsg = fmt.Sprintf("Added @%s", handle)
		m.mode = modeList
		m.input = ""
		m.cursor = len(m.cfg.Follows) - 1
	default:
		if msg.Text != "" {
			m.input += msg.Text
		} else if msg.String() == "backspace" {
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString("Followed Accounts\n")
	b.WriteString(strings.Repeat("─", 40) + "\n\n")

	if len(m.cfg.Follows) == 0 {
		b.WriteString("  No accounts followed yet.\n")
		b.WriteString("  Press 'a' to add a Bluesky handle.\n\n")
	} else {
		for i, handle := range m.cfg.Follows {
			cursor := "  "
			if m.cursor == i && m.mode == modeList {
				cursor = "> "
			}
			b.WriteString(fmt.Sprintf("%s@%s\n", cursor, handle))
		}
		b.WriteString("\n")
	}

	if m.mode == modeInput {
		b.WriteString(fmt.Sprintf("Enter handle: @%s█\n", m.input))
	}

	b.WriteString("\n")
	if m.statusMsg != "" {
		b.WriteString(m.statusMsg + "\n")
	}

	b.WriteString("\n[a] add  [d] delete  [esc] back\n")
	return tea.NewView(b.String())
}

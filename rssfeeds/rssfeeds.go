package rssfeeds

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/config"
	"github.com/Hadidomena/tuiFeed/rss"
	"github.com/Hadidomena/tuiFeed/utils"
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
	width     int
	height    int
}

func (m Model) WithSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

func NewModel() (Model, error) {
	cfg, err := config.Load()
	if err != nil {
		return Model{}, err
	}
	return Model{
		cfg:    cfg,
		width:  utils.DefaultWidth,
		height: 24,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
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
		m.cursor = utils.CursorUp(m.cursor)
	case "down", "j":
		m.cursor = utils.CursorDown(m.cursor, len(m.cfg.RSSFeeds))
	case "a":
		m.mode = modeInput
		m.input = ""
		m.statusMsg = ""
	case "d":
		if len(m.cfg.RSSFeeds) > 0 {
			url := m.cfg.RSSFeeds[m.cursor]
			fresh, err := config.ApplyUpdateAndReload(func(cfg *config.Config) {
				idx := -1
				for i, u := range cfg.RSSFeeds {
					if u == url {
						idx = i
						break
					}
				}
				if idx >= 0 {
					cfg.RemoveRSSFeed(idx)
				}
			})
			if err != nil {
				m.statusMsg = fmt.Sprintf("Error removing feed: %v", err)
				return m, nil
			}
			m.cfg = fresh
			if m.cursor >= len(m.cfg.RSSFeeds) && m.cursor > 0 {
				m.cursor--
			}
			m.statusMsg = fmt.Sprintf("Removed %s", url)
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
		url := strings.TrimSpace(m.input)
		if url == "" {
			m.statusMsg = "URL cannot be empty"
			m.mode = modeList
			return m, nil
		}
		url = rss.NormalizeTwitterInput(url)
		fresh, err := config.ApplyUpdateAndReload(func(cfg *config.Config) {
			cfg.AddRSSFeed(url)
		})
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error adding feed: %v", err)
			return m, nil
		}
		m.cfg = fresh
		m.statusMsg = fmt.Sprintf("Added %s", url)
		m.mode = modeList
		m.input = ""
		m.cursor = len(m.cfg.RSSFeeds) - 1
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

	utils.WriteHeader(&b, "RSS Feeds", m.width)

	if len(m.cfg.RSSFeeds) == 0 {
		b.WriteString("  No RSS feeds subscribed yet.\n")
		b.WriteString("  Press 'a' to add a feed URL or X/Twitter @handle.\n\n")
	} else {
		for i, url := range m.cfg.RSSFeeds {
			cursor := "  "
			if m.cursor == i && m.mode == modeList {
				cursor = "> "
			}
			fmt.Fprintf(&b, "%s%s\n", cursor, url)
		}
		b.WriteString("\n")
	}

	if m.mode == modeInput {
		fmt.Fprintf(&b, "Enter feed URL or X/Twitter @handle: %s█\n", m.input)
	}

	b.WriteString("\n")
	if m.statusMsg != "" {
		b.WriteString(m.statusMsg + "\n")
	}

	b.WriteString("\n[a] add  [d] delete  [esc] back\n")
	return tea.NewView(utils.CenterBlock(b.String(), m.width))
}

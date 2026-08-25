package rssfeed

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/attach"
	"github.com/Hadidomena/tuiFeed/config"
	"github.com/Hadidomena/tuiFeed/rss"
	"github.com/Hadidomena/tuiFeed/utils"
)

type BackMsg struct{}

type entriesLoadedMsg struct {
	entries []rss.Entry
	warn    string
}

type loadErrorMsg string

type statusMsg string

type Model struct {
	cfg         *config.Config
	entries     []rss.Entry
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
		loading:  true,
		pageSize: utils.DefaultPageSize,
		title:    "RSS feeds",
		width:    utils.DefaultWidth,
		height:   24,
	}, nil
}

func NewStaticModel(entries []rss.Entry, title string) Model {
	return Model{
		entries:  entries,
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
		loading:     true,
		pageSize:    utils.DefaultPageSize,
		title:       "Saved RSS entries",
		isSavedView: true,
		width:       utils.DefaultWidth,
		height:      24,
	}
}

func (m Model) Init() tea.Cmd {
	if m.isSavedView {
		return m.loadSavedEntries
	}
	return m.loadEntries
}

func (m Model) loadEntries() tea.Msg {
	if m.cfg == nil {
		return loadErrorMsg("Not available in this view.")
	}
	ctx := context.Background()

	if len(m.cfg.RSSFeeds) == 0 {
		return loadErrorMsg("No RSS feeds subscribed. Add some in Manage RSS feeds.")
	}

	entries, err := rss.FetchAll(ctx, m.cfg.RSSFeeds)
	if err != nil && len(entries) == 0 {
		return loadErrorMsg(fmt.Sprintf("Error fetching feeds: %v", err))
	}
	msg := entriesLoadedMsg{entries: entries}
	if err != nil {
		msg.warn = fmt.Sprintf("Some feeds failed: %v", err)
	}
	return msg
}

func (m Model) loadSavedEntries() tea.Msg {
	if m.cfg == nil {
		return loadErrorMsg("Not available in this view.")
	}
	saved := m.cfg.GetSavedEntryIDs()
	if len(saved) == 0 {
		return entriesLoadedMsg{}
	}
	ctx := context.Background()
	entries, err := rss.FetchAll(ctx, m.cfg.RSSFeeds)
	if err != nil && len(entries) == 0 {
		return loadErrorMsg(fmt.Sprintf("Error loading saved entries: %v", err))
	}
	msg := entriesLoadedMsg{}
	if err != nil {
		msg.warn = fmt.Sprintf("Some feeds failed: %v", err)
	}
	savedSet := make(map[string]bool, len(saved))
	for _, id := range saved {
		savedSet[id] = true
	}
	filtered := make([]rss.Entry, 0, len(saved))
	for _, e := range entries {
		if savedSet[e.ID] {
			filtered = append(filtered, e)
		}
	}
	msg.entries = filtered
	return msg
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.hasRendered = false
			m.imageRows = 0
			return m, func() tea.Msg { return BackMsg{} }
		case "down", "j":
			old := m.cursor
			oldScroll := m.scrollPos
			m.cursor, m.scrollPos = utils.ScrollDown(m.cursor, m.scrollPos, m.pageSize, len(m.entries))
			if m.cursor != old || m.scrollPos != oldScroll {
				m.imgCursor = 0
				m.hasRendered = false
				m.imageRows = 0
				m.statusMsg = ""
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
			}
		case "left", "h":
			if m.hasRendered && m.imgCursor > 0 {
				m.imgCursor--
				return m, m.renderAttachment
			}
		case "right", "l":
			if m.hasRendered && m.imgCursor < len(m.entries[m.cursor].Images)-1 {
				m.imgCursor++
				return m, m.renderAttachment
			}
		case "a":
			if m.cursor < len(m.entries) && len(m.entries[m.cursor].Images) > 0 {
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
			if m.cursor < len(m.entries) && len(m.entries[m.cursor].Images) > 0 {
				if !m.hasRendered {
					m.hasRendered = true
					m.imgCursor = 0
				}
				m.statusMsg = "Opening image externally..."
				return m, m.openAttachment
			}
		case "v":
			if m.cursor < len(m.entries) && len(m.entries[m.cursor].Videos) > 0 {
				url := m.entries[m.cursor].Videos[0]
				m.statusMsg = "Launching player..."
				return m, func() tea.Msg {
					mode, err := utils.StreamVideo(url)
					if err != nil {
						return loadErrorMsg(fmt.Sprintf("Play failed: %v", err))
					}
					return statusMsg(fmt.Sprintf("Playing via %s", mode))
				}
			}
			m.statusMsg = "Entry has no video"
		case "w", "enter":
			if m.cursor < len(m.entries) {
				link := m.entries[m.cursor].Link
				if link != "" {
					return m, func() tea.Msg {
						if err := utils.OpenExternal(link); err != nil {
							return loadErrorMsg(fmt.Sprintf("Open failed: %v", err))
						}
						return statusMsg("Opened in browser")
					}
				}
				m.statusMsg = "Entry has no link"
			}
		case "s":
			if m.cfg == nil || len(m.entries) == 0 || m.cursor >= len(m.entries) {
				return m, nil
			}
			id := m.entries[m.cursor].ID
			if id == "" {
				m.statusMsg = "Cannot save entry (no ID)"
				return m, nil
			}
			if m.isSavedView {
				fresh, err := config.ApplyUpdateAndReload(func(cfg *config.Config) {
					cfg.RemoveSavedEntry(id)
				})
				if err != nil {
					m.statusMsg = fmt.Sprintf("Error removing saved entry: %v", err)
					return m, nil
				}
				m.entries = append(m.entries[:m.cursor], m.entries[m.cursor+1:]...)
				if m.cursor >= len(m.entries) && m.cursor > 0 {
					m.cursor--
				}
				if m.scrollPos >= len(m.entries) {
					m.scrollPos = len(m.entries) - m.pageSize
					if m.scrollPos < 0 {
						m.scrollPos = 0
					}
				}
				m.hasRendered = false
				m.imageRows = 0
				m.statusMsg = "Removed from saved"
				if len(m.entries) == 0 {
					m.statusMsg = "No saved entries"
				}
				m.cfg = fresh
				m = m.recalcPageSize()
				return m, nil
			}
			if m.cfg.IsEntrySaved(id) {
				if err := config.Update(func(cfg *config.Config) {
					cfg.RemoveSavedEntry(id)
				}); err != nil {
					m.statusMsg = fmt.Sprintf("Error unsaving entry: %v", err)
					return m, nil
				}
				m.statusMsg = "Unsaved"
			} else {
				if err := config.Update(func(cfg *config.Config) {
					cfg.SaveEntry(id)
				}); err != nil {
					m.statusMsg = fmt.Sprintf("Error saving entry: %v", err)
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
		case "r":
			if m.isSavedView {
				return m, nil
			}
			m.loading = true
			m.statusMsg = ""
			m.hasRendered = false
			m.imageRows = 0
			m.scrollPos = 0
			return m, m.loadEntries
		}
	case entriesLoadedMsg:
		m.entries = msg.entries
		m.loading = false
		m.cursor = 0
		m.imgCursor = 0
		m.scrollPos = 0
		m.hasRendered = false
		m.imageRows = 0
		switch {
		case msg.warn != "":
			m.statusMsg = "Warning: " + msg.warn
		case len(msg.entries) == 0:
			m.statusMsg = "No entries found."
		}
	case loadErrorMsg:
		m.loading = false
		m.statusMsg = string(msg)
	case statusMsg:
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
	if m.height <= 0 || len(m.entries) == 0 {
		m.pageSize = utils.DefaultPageSize
		return m
	}

	cw := utils.ContentWidth(m.width)
	budget := utils.TextBudget(m.height)

	idx := m.cursor
	if idx >= len(m.entries) {
		idx = 0
	}
	entryLines := strings.Count(rss.FormatEntry(m.entries[idx], -1, cw, budget), "\n")

	itemLines := strings.Count(rss.FormatEntryListItem(m.entries[idx], false, cw), "\n") + 1
	if itemLines < 1 {
		itemLines = 1
	}

	reserve := 10
	available := m.height - reserve - entryLines
	m.pageSize = available / itemLines
	if m.pageSize < 0 {
		m.pageSize = 0
	}
	return m
}

func (m Model) renderAttachment() tea.Msg {
	if len(m.entries) == 0 || m.cursor >= len(m.entries) {
		return attach.ErrorMsg("No entry selected")
	}
	yOffset := m.computeYOffset()
	maxCols := attach.ComputeMaxCols(m.width)
	maxRows := attach.ComputeMaxRows(m.height, yOffset)
	return attach.Render(m.entries[m.cursor].Images, m.imgCursor, yOffset, maxCols, maxRows)
}

func (m Model) computeYOffset() int {
	if len(m.entries) == 0 {
		return 4
	}

	cw := utils.ContentWidth(m.width)

	lines := 0

	lines += 3

	lines += 2

	end := utils.ScrollWindowEnd(m.scrollPos, m.pageSize, len(m.entries))
	for i := m.scrollPos; i < end; i++ {
		lines += strings.Count(rss.FormatEntryListItem(m.entries[i], m.cursor == i, cw), "\n")
		lines += 1
	}

	if m.scrollPos > 0 {
		lines += 1
	}
	if end < len(m.entries) {
		lines += 1
	}

	lines += 2

	idx := m.cursor
	if idx >= len(m.entries) {
		idx = 0
	}
	entryText := rss.FormatEntry(m.entries[idx], -1, cw, utils.TextBudget(m.height))
	lines += strings.Count(entryText, "\n")

	return lines
}

func (m Model) openAttachment() tea.Msg {
	entry := m.entries[m.cursor]
	return attach.Open(entry.Images, m.imgCursor)
}

func (m Model) View() tea.View {
	var b strings.Builder

	utils.WriteHeader(&b, m.title, m.width)

	if m.loading {
		if m.isSavedView {
			b.WriteString("Loading saved RSS entries...\n")
		} else {
			b.WriteString("Loading RSS feeds...\n")
		}
		return tea.NewView(b.String())
	}

	if len(m.entries) == 0 {
		b.WriteString("No entries to display.\n")
	} else {
		cw := utils.ContentWidth(m.width)
		fmt.Fprintf(&b, "%d entries\n\n", len(m.entries))

		end := utils.ScrollWindowEnd(m.scrollPos, m.pageSize, len(m.entries))

		for i := m.scrollPos; i < end; i++ {
			b.WriteString(rss.FormatEntryListItem(m.entries[i], m.cursor == i, cw))
			b.WriteString("\n")
		}

		rss.WriteMoreIndicators(&b, m.scrollPos, end, len(m.entries))
		idx := m.cursor
		if idx >= len(m.entries) {
			idx = 0
		}
		fmt.Fprintf(&b, "Entry %d/%d\n\n", idx+1, len(m.entries))
		cursor := -1
		if m.hasRendered {
			cursor = m.imgCursor
		}
		b.WriteString(rss.FormatEntry(m.entries[idx], cursor, cw, utils.TextBudget(m.height)))
	}

	if m.hasRendered && m.imageRows > 0 {
		b.WriteString(strings.Repeat("\n", m.imageRows))
	}

	if m.statusMsg != "" {
		fmt.Fprintf(&b, "%s\n", m.statusMsg)
	}

	if m.isSavedView {
		b.WriteString("\n[w] open link  [s] remove  [a] attachments  [o] open externally  [v] play video  [esc] back  [q] quit\n")
	} else {
		b.WriteString("\n[w] open link  [s] save  [a] attachments  [o] open externally  [v] play video  [r] refresh  [esc] back  [q] quit\n")
	}
	return tea.NewView(utils.CenterBlock(b.String(), m.width))
}

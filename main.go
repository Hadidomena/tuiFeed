package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/Hadidomena/tuiFeed/bsk"
)

type model struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func initialModel() model {
	return model{
		choices:  []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}
	return m, nil
}

func (m model) View() tea.View {
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

func main() {
	postUrl := "https://bsky.app/profile/p-e-g-76.bsky.social/post/3mcncopydnc2g"
	ctx := context.Background()
	client := bsk.NewClient()

	thread, err := bsk.GetPostThread(ctx, client, postUrl)
	if err != nil {
		panic(fmt.Errorf("error fetching post: %w", err))
	}

	if thread.Parent != nil {
		fmt.Println("--- PARENT POST (REPLY TO) ---")
		fmt.Printf("Author: %s (@%s)\n", thread.Parent.AuthorDisplayName, thread.Parent.AuthorHandle)
		fmt.Printf("Content: %s\n", thread.Parent.Text)
		fmt.Printf("❤️  %d | 💬 %d\n", thread.Parent.LikeCount, thread.Parent.ReplyCount)
		fmt.Printf("Embeds: %+v\n", thread.Parent.Embeds)
	}

	fmt.Println("--- FOUND POST ---")
	fmt.Printf("Author: %s (@%s)\n", thread.Post.AuthorDisplayName, thread.Post.AuthorHandle)
	fmt.Printf("Content: %s\n", thread.Post.Text)
	fmt.Printf("Likes: %d\n", thread.Post.LikeCount)
	fmt.Printf("Date:  %s\n", thread.Post.CreatedAt)
	fmt.Printf("Embeds: %+v\n", thread.Post.Embeds)

	if len(thread.Replies) > 0 {
		fmt.Printf("\n--- REPLIES (%d) ---\n", len(thread.Replies))
		for i, reply := range thread.Replies {
			fmt.Printf("\n[%d] %s (@%s):\n", i+1, reply.AuthorDisplayName, reply.AuthorHandle)
			fmt.Printf("    %s\n", reply.Text)
			fmt.Printf("    ❤️ %d | 💬 %d\n", reply.LikeCount, reply.ReplyCount)
		}
	} else {
		fmt.Println("\nNo replies.")
	}

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

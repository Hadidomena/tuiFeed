package main

import (
	"context"
	"fmt"
	utils "github.com/Hadidomena/tuiFeed/utils"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
)

type model struct {
	choices  []string         // items on the to-do list
	cursor   int              // which to-do list item our cursor is pointing at
	selected map[int]struct{} // which to-do items are selected
}

func initialModel() model {
	return model{
		// Our to-do list is a grocery list
		choices: []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyPressMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// The "enter" key and the space bar toggle the selected state
		// for the item that the cursor is pointing at.
		case "enter", "space":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() tea.View {
	// The header
	s := "What should we buy at the market?\n\n"

	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return tea.NewView(s)
}

func returnThread(ctx context.Context, client *xrpc.Client, postUrl string) (*bsky.FeedGetPostThread_Output, error) {
	parts := strings.Split(postUrl, "/")
	if len(parts) < 7 {
		panic("Invalid post URL format")
	}
	handle := parts[4]
	rkey := parts[6]

	resolveOutput, err := atproto.IdentityResolveHandle(ctx, client, handle)
	if err != nil {
		panic(fmt.Errorf("error resolving user: %w", err))
	}
	did := resolveOutput.Did
	atURI := fmt.Sprintf("at://%s/app.bsky.feed.post/%s", did, rkey)

	threadOutput, err := bsky.FeedGetPostThread(ctx, client, 0, 0, atURI)
	if err != nil {
		panic(fmt.Errorf("post not found: %w", err))
	}

	return threadOutput, nil
}

func getExtantEmbeds(Post *bsky.FeedDefs_PostView) []string {
	result := []string{}

	if Post.Embed != nil && Post.Embed.EmbedImages_View != nil {
		for _, img := range Post.Embed.EmbedImages_View.Images {
			result = append(result, img.Fullsize)
		}
	}

	if Post.Embed != nil && Post.Embed.EmbedVideo_View != nil {
		result = append(result, Post.Embed.EmbedVideo_View.Playlist)
	}

	return result
}

func main() {
	postUrl := "https://bsky.app/profile/p-e-g-76.bsky.social/post/3mcncopydnc2g"
	ctx := context.Background()
	client := &xrpc.Client{
		Host: "https://public.api.bsky.app",
	}

	threadOutput, err := returnThread(ctx, client, postUrl)
	if err != nil {
		panic(fmt.Errorf("error fetching post: %w", err))
	}

	if threadOutput.Thread.FeedDefs_ThreadViewPost != nil {
		threadView := threadOutput.Thread.FeedDefs_ThreadViewPost
		post := threadView.Post
		record := post.Record.Val.(*bsky.FeedPost)
		if threadView.Parent != nil && threadView.Parent.FeedDefs_ThreadViewPost != nil {
			parentPost := threadView.Parent.FeedDefs_ThreadViewPost.Post
			parentRecord := parentPost.Record.Val.(*bsky.FeedPost)

			fmt.Println("--- PARENT POST (REPLY TO) ---")
			fmt.Printf("Author: %s (@%s)\n", utils.StrVal(parentPost.Author.DisplayName), utils.StrVal(parentPost.Author.Handle))
			fmt.Printf("Content: %s\n", utils.StrVal(parentRecord.Text))
			fmt.Printf("❤️  %d | 💬 %d\n\n", *parentPost.LikeCount, *parentPost.ReplyCount)
			fmt.Printf("Embeds: %+v\n", getExtantEmbeds(parentPost))
		}

		fmt.Println("--- FOUND POST ---")
		fmt.Printf("Author: %s (@%s)\n", utils.StrVal(post.Author.DisplayName), utils.StrVal(post.Author.Handle))
		fmt.Printf("Content: %s\n", utils.StrVal(record.Text))
		fmt.Printf("Likes: %d\n", *post.LikeCount)
		fmt.Printf("Date:  %s\n", utils.StrVal(record.CreatedAt))
		fmt.Printf("Embeds: %+v\n", getExtantEmbeds(post))

		if len(threadView.Replies) > 0 {
			fmt.Printf("\n--- REPLIES (%d) ---\n", len(threadView.Replies))
			for i, reply := range threadView.Replies {
				if reply.FeedDefs_ThreadViewPost != nil {
					replyPost := reply.FeedDefs_ThreadViewPost.Post
					replyRecord := replyPost.Record.Val.(*bsky.FeedPost)

					fmt.Printf("\n[%d] %s (@%s):\n", i+1, utils.StrVal(replyPost.Author.DisplayName), utils.StrVal(replyPost.Author.Handle))
					fmt.Printf("    %s\n", utils.StrVal(replyRecord.Text))
					fmt.Printf("    ❤️ %d | 💬 %d\n", *replyPost.LikeCount, *replyPost.ReplyCount)
				}
			}
		} else {
			fmt.Println("\nNo replies.")
		}
	} else {
		fmt.Println("Failed to read post details.")
	}

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

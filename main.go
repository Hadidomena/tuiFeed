package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
)

func returnPost(ctx context.Context, client *xrpc.Client, postUrl string) (*bsky.FeedGetPostThread_Output, error) {
	parts := strings.Split(postUrl, "/")
	if len(parts) < 7 {
		panic("Nieprawidłowy format linku")
	}
	handle := parts[4]
	rkey := parts[6]

	resolveOutput, err := atproto.IdentityResolveHandle(ctx, client, handle)
	if err != nil {
		panic(fmt.Errorf("błąd rozwiązywania użytkownika: %w", err))
	}
	did := resolveOutput.Did
	atURI := fmt.Sprintf("at://%s/app.bsky.feed.post/%s", did, rkey)

	threadOutput, err := bsky.FeedGetPostThread(ctx, client, 0, 0, atURI)
	if err != nil {
		panic(fmt.Errorf("nie znaleziono posta: %w", err))
	}

	return threadOutput, nil
}
func main() {
	postUrl := "https://bsky.app/profile/p-e-g-76.bsky.social/post/3mcncopydnc2g"
	ctx := context.Background()
	client := &xrpc.Client{
		Host: "https://public.api.bsky.app",
	}

	threadOutput, err := returnPost(ctx, client, postUrl)
	if err != nil {
		panic(fmt.Errorf("błąd podczas pobierania posta: %w", err))
	}

	if threadOutput.Thread.FeedDefs_ThreadViewPost != nil {
		threadView := threadOutput.Thread.FeedDefs_ThreadViewPost
		post := threadView.Post
		record := post.Record.Val.(*bsky.FeedPost) // Rzutowanie na konkretny typ posta

		// 6a. Wyświetlanie posta nadrzędnego (jeśli to odpowiedź)
		if threadView.Parent != nil && threadView.Parent.FeedDefs_ThreadViewPost != nil {
			parentPost := threadView.Parent.FeedDefs_ThreadViewPost.Post
			parentRecord := parentPost.Record.Val.(*bsky.FeedPost)

			fmt.Println("--- POST NADRZĘDNY (ODPOWIADA NA) ---")
			fmt.Printf("Autor: %s (@%s)\n", parentPost.Author.DisplayName, parentPost.Author.Handle)
			fmt.Printf("Treść: %s\n", parentRecord.Text)
			fmt.Printf("❤️  %d | 💬 %d\n\n", *parentPost.LikeCount, *parentPost.ReplyCount)
		}

		fmt.Println("--- ZNALEZIONO POST ---")
		fmt.Printf("Autor: %s (%s)\n", post.Author.DisplayName, post.Author.Handle)
		fmt.Printf("Treść: %s\n", record.Text)
		fmt.Printf("Lajki: %d\n", *post.LikeCount)
		fmt.Printf("Data:  %s\n", record.CreatedAt)

		// 7. Wyświetlanie odpowiedzi
		if len(threadView.Replies) > 0 {
			fmt.Printf("\n--- ODPOWIEDZI (%d) ---\n", len(threadView.Replies))
			for i, reply := range threadView.Replies {
				if reply.FeedDefs_ThreadViewPost != nil {
					replyPost := reply.FeedDefs_ThreadViewPost.Post
					replyRecord := replyPost.Record.Val.(*bsky.FeedPost)

					fmt.Printf("\n[%d] %s (@%s):\n", i+1, replyPost.Author.DisplayName, replyPost.Author.Handle)
					fmt.Printf("    %s\n", replyRecord.Text)
					fmt.Printf("    ❤️ %d | 💬 %d\n", *replyPost.LikeCount, *replyPost.ReplyCount)
				}
			}
		} else {
			fmt.Println("\nBrak odpowiedzi.")
		}
	} else {
		fmt.Println("Nie udało się odczytać szczegółów posta.")
	}
}

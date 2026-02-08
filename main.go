package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
)

func strVal(v interface{}) string {
	switch s := v.(type) {
	case *string:
		if s == nil {
			return ""
		}
		return *s
	case string:
		return s
	default:
		return ""
	}
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
			fmt.Printf("Author: %s (@%s)\n", strVal(parentPost.Author.DisplayName), strVal(parentPost.Author.Handle))
			fmt.Printf("Content: %s\n", strVal(parentRecord.Text))
			fmt.Printf("❤️  %d | 💬 %d\n\n", *parentPost.LikeCount, *parentPost.ReplyCount)
			fmt.Printf("Embeds: %+v\n", getExtantEmbeds(parentPost))
		}

		fmt.Println("--- FOUND POST ---")
		fmt.Printf("Author: %s (@%s)\n", strVal(post.Author.DisplayName), strVal(post.Author.Handle))
		fmt.Printf("Content: %s\n", strVal(record.Text))
		fmt.Printf("Likes: %d\n", *post.LikeCount)
		fmt.Printf("Date:  %s\n", strVal(record.CreatedAt))
		fmt.Printf("Embeds: %+v\n", getExtantEmbeds(post))

		if len(threadView.Replies) > 0 {
			fmt.Printf("\n--- REPLIES (%d) ---\n", len(threadView.Replies))
			for i, reply := range threadView.Replies {
				if reply.FeedDefs_ThreadViewPost != nil {
					replyPost := reply.FeedDefs_ThreadViewPost.Post
					replyRecord := replyPost.Record.Val.(*bsky.FeedPost)

					fmt.Printf("\n[%d] %s (@%s):\n", i+1, strVal(replyPost.Author.DisplayName), strVal(replyPost.Author.Handle))
					fmt.Printf("    %s\n", strVal(replyRecord.Text))
					fmt.Printf("    ❤️ %d | 💬 %d\n", *replyPost.LikeCount, *replyPost.ReplyCount)
				}
			}
		} else {
			fmt.Println("\nNo replies.")
		}
	} else {
		fmt.Println("Failed to read post details.")
	}
}

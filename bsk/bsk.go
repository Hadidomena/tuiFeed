package bsk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
)

type PostInfo struct {
	AuthorDisplayName string
	AuthorHandle      string
	Text              string
	LikeCount         int64
	ReplyCount        int64
	CreatedAt         string
	IndexedAt         string
	Embeds            []string
}

type Thread struct {
	Post    PostInfo
	Parent  *PostInfo
	Replies []PostInfo
}

type ThreadNode struct {
	Post    PostInfo
	URI     string
	Parent  *ThreadNode
	Replies []*ThreadNode
}

func GetThreadByURI(ctx context.Context, client *xrpc.Client, atURI string) (*ThreadNode, error) {
	threadOutput, err := bsky.FeedGetPostThread(ctx, client, 0, 0, atURI)
	if err != nil {
		return nil, fmt.Errorf("fetching thread: %w", err)
	}
	if threadOutput == nil || threadOutput.Thread == nil {
		return nil, fmt.Errorf("empty response from server")
	}
	if threadOutput.Thread.FeedDefs_ThreadViewPost == nil {
		return nil, fmt.Errorf("post not found or not accessible")
	}
	return BuildThreadTree(threadOutput.Thread.FeedDefs_ThreadViewPost), nil
}

func BuildThreadTree(threadView *bsky.FeedDefs_ThreadViewPost) *ThreadNode {
	if threadView == nil || threadView.Post == nil {
		return nil
	}

	node := &ThreadNode{
		Post: ExtractPostInfo(threadView.Post),
		URI:  threadView.Post.Uri,
	}

	if threadView.Parent != nil && threadView.Parent.FeedDefs_ThreadViewPost != nil {
		parentNode := BuildThreadTree(threadView.Parent.FeedDefs_ThreadViewPost)
		if parentNode != nil {
			node.Parent = parentNode
		}
	}

	for _, reply := range threadView.Replies {
		if reply.FeedDefs_ThreadViewPost != nil {
			child := BuildThreadTree(reply.FeedDefs_ThreadViewPost)
			if child != nil {
				child.Parent = node
				node.Replies = append(node.Replies, child)
			}
		}
	}

	return node
}

func NewClient() *xrpc.Client {
	return &xrpc.Client{
		Host: "https://public.api.bsky.app",
	}
}

func GetPostThread(ctx context.Context, client *xrpc.Client, postURL string) (*Thread, error) {
	parts := strings.Split(postURL, "/")
	if len(parts) < 7 {
		return nil, fmt.Errorf("invalid post URL format: %s", postURL)
	}
	handle := parts[4]
	rkey := parts[6]

	resolveOutput, err := atproto.IdentityResolveHandle(ctx, client, handle)
	if err != nil {
		return nil, fmt.Errorf("error resolving user: %w", err)
	}
	did := resolveOutput.Did
	atURI := fmt.Sprintf("at://%s/app.bsky.feed.post/%s", did, rkey)

	threadOutput, err := bsky.FeedGetPostThread(ctx, client, 0, 0, atURI)
	if err != nil {
		return nil, fmt.Errorf("post not found: %w", err)
	}

	if threadOutput.Thread.FeedDefs_ThreadViewPost == nil {
		return nil, fmt.Errorf("failed to read post details")
	}

	threadView := threadOutput.Thread.FeedDefs_ThreadViewPost
	thread := &Thread{
		Post: ExtractPostInfo(threadView.Post),
	}

	if threadView.Parent != nil && threadView.Parent.FeedDefs_ThreadViewPost != nil {
		parent := ExtractPostInfo(threadView.Parent.FeedDefs_ThreadViewPost.Post)
		thread.Parent = &parent
	}

	for _, reply := range threadView.Replies {
		if reply.FeedDefs_ThreadViewPost != nil {
			thread.Replies = append(thread.Replies, ExtractPostInfo(reply.FeedDefs_ThreadViewPost.Post))
		}
	}

	return thread, nil
}

func ExtractPostInfo(post *bsky.FeedDefs_PostView) PostInfo {
	info := PostInfo{
		AuthorHandle: post.Author.Handle,
		LikeCount:    int64Val(post.LikeCount),
		ReplyCount:   int64Val(post.ReplyCount),
		IndexedAt:    post.IndexedAt,
	}
	if post.Author.DisplayName != nil {
		info.AuthorDisplayName = *post.Author.DisplayName
	}
	if post.Record != nil && post.Record.Val != nil {
		if feedPost, ok := post.Record.Val.(*bsky.FeedPost); ok {
			info.Text = feedPost.Text
			info.CreatedAt = feedPost.CreatedAt
		}
	}
	info.Embeds = GetExtantEmbeds(post)
	return info
}

func GetExtantEmbeds(Post *bsky.FeedDefs_PostView) []string {
	if Post == nil {
		return nil
	}
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

type FeedItem struct {
	PostInfo
	URI string
}

func GetAuthorFeed(ctx context.Context, client *xrpc.Client, actor string, limit int64) ([]FeedItem, error) {
	output, err := bsky.FeedGetAuthorFeed(ctx, client, actor, "", "posts_no_replies", false, limit)
	if err != nil {
		return nil, fmt.Errorf("error fetching feed: %w", err)
	}

	var items []FeedItem
	for _, fvp := range output.Feed {
		if fvp.Post == nil {
			continue
		}
		info := ExtractPostInfo(fvp.Post)
		items = append(items, FeedItem{
			PostInfo: info,
			URI:      fvp.Post.Uri,
		})
	}
	return items, nil
}

func FormatPost(item FeedItem, imgCursor int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("─── %s (@%s) ───\n", item.AuthorDisplayName, item.AuthorHandle))
	b.WriteString(fmt.Sprintf("❤️ %d  💬 %d  📅 %s\n", item.LikeCount, item.ReplyCount, item.CreatedAt))
	b.WriteString("\n")
	b.WriteString(item.Text)
	b.WriteString("\n\n")

	if len(item.Embeds) > 0 {
		b.WriteString(fmt.Sprintf("── %d Attachments ──\n", len(item.Embeds)))
		for i, embed := range item.Embeds {
			marker := "  "
			if i == imgCursor {
				marker = "> "
			}
			b.WriteString(fmt.Sprintf("%s%s\n", marker, embed))
		}
	}

	return b.String()
}

func RenderImage(url string, yOffset int) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read failed: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return 0, fmt.Errorf("not an image: %s", ct)
	}

	return RenderTerminalImage(data, yOffset)
}

func int64Val(p *int64) int64 {
	if p != nil {
		return *p
	}
	return 0
}

func GetAuthorFeedCursor(ctx context.Context, client *xrpc.Client, handle string, cursor string, limit int64) ([]FeedItem, string, error) {
	d, err := atproto.IdentityResolveHandle(ctx, client, handle)
	if err != nil {
		return nil, "", fmt.Errorf("resolving %s: %w", handle, err)
	}
	out, err := bsky.FeedGetAuthorFeed(ctx, client, d.Did, cursor, "posts_with_replies", false, limit)
	if err != nil {
		return nil, "", fmt.Errorf("fetching feed for %s: %w", handle, err)
	}
	posts := make([]FeedItem, 0, len(out.Feed))
	for _, f := range out.Feed {
		if f.Post != nil {
			posts = append(posts, FeedItem{
				PostInfo: ExtractPostInfo(f.Post),
				URI:      f.Post.Uri,
			})
		}
	}
	nextCursor := ""
	if out.Cursor != nil {
		nextCursor = *out.Cursor
	}
	return posts, nextCursor, nil
}

func GetPosts(ctx context.Context, client *xrpc.Client, uris []string) ([]FeedItem, error) {
	const batchSize = 25
	posts := make([]FeedItem, 0, len(uris))
	for i := 0; i < len(uris); i += batchSize {
		end := i + batchSize
		if end > len(uris) {
			end = len(uris)
		}
		chunk := uris[i:end]
		out, err := bsky.FeedGetPosts(ctx, client, chunk)
		if err != nil {
			return nil, fmt.Errorf("fetching posts: %w", err)
		}
		for _, p := range out.Posts {
			if p != nil {
				posts = append(posts, FeedItem{
					PostInfo: ExtractPostInfo(p),
					URI:      p.Uri,
				})
			}
		}
	}
	return posts, nil
}

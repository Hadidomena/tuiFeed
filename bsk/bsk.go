package bsk

import (
	"context"
	"fmt"
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
	Embeds            []string
}

type Thread struct {
	Post    PostInfo
	Parent  *PostInfo
	Replies []PostInfo
}

func NewClient() *xrpc.Client {
	return &xrpc.Client{
		Host: "https://public.api.bsky.app",
	}
}

func GetPostThread(ctx context.Context, client *xrpc.Client, postUrl string) (*Thread, error) {
	parts := strings.Split(postUrl, "/")
	if len(parts) < 7 {
		return nil, fmt.Errorf("invalid post URL format: %s", postUrl)
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

func int64Val(p *int64) int64 {
	if p != nil {
		return *p
	}
	return 0
}

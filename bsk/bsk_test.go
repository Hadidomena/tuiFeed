package bsk

import (
	"context"
	"testing"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
)

func ptr[T any](v T) *T {
	return &v
}

func makePostView(opts ...func(*bsky.FeedDefs_PostView)) *bsky.FeedDefs_PostView {
	p := &bsky.FeedDefs_PostView{
		Author: &bsky.ActorDefs_ProfileViewBasic{
			DisplayName: ptr("Test User"),
			Handle:      "test.bsky.social",
		},
		Record: &util.LexiconTypeDecoder{
			Val: &bsky.FeedPost{
				Text:      "Hello world",
				CreatedAt: "2024-01-15T10:00:00Z",
			},
		},
		LikeCount:  ptr[int64](42),
		ReplyCount: ptr[int64](7),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func TestExtractPostInfo_basic(t *testing.T) {
	post := makePostView()
	info := ExtractPostInfo(post)

	if info.AuthorDisplayName != "Test User" {
		t.Errorf("expected 'Test User', got %q", info.AuthorDisplayName)
	}
	if info.AuthorHandle != "test.bsky.social" {
		t.Errorf("expected 'test.bsky.social', got %q", info.AuthorHandle)
	}
	if info.Text != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", info.Text)
	}
	if info.LikeCount != 42 {
		t.Errorf("expected 42, got %d", info.LikeCount)
	}
	if info.ReplyCount != 7 {
		t.Errorf("expected 7, got %d", info.ReplyCount)
	}
	if info.CreatedAt != "2024-01-15T10:00:00Z" {
		t.Errorf("expected '2024-01-15T10:00:00Z', got %q", info.CreatedAt)
	}
	if len(info.Embeds) != 0 {
		t.Errorf("expected 0 embeds, got %d", len(info.Embeds))
	}
}

func TestExtractPostInfo_nilDisplayName(t *testing.T) {
	post := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Author.DisplayName = nil
	})
	info := ExtractPostInfo(post)

	if info.AuthorDisplayName != "" {
		t.Errorf("expected empty display name, got %q", info.AuthorDisplayName)
	}
}

func TestExtractPostInfo_nilCounts(t *testing.T) {
	post := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.LikeCount = nil
		p.ReplyCount = nil
	})
	info := ExtractPostInfo(post)

	if info.LikeCount != 0 {
		t.Errorf("expected 0, got %d", info.LikeCount)
	}
	if info.ReplyCount != 0 {
		t.Errorf("expected 0, got %d", info.ReplyCount)
	}
}

func TestExtractPostInfo_noRecord(t *testing.T) {
	post := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Record = nil
	})
	info := ExtractPostInfo(post)
	if info.Text != "" {
		t.Errorf("expected empty text, got %q", info.Text)
	}
	if info.CreatedAt != "" {
		t.Errorf("expected empty CreatedAt, got %q", info.CreatedAt)
	}
}

func TestGetExtantEmbeds_nil(t *testing.T) {
	post := makePostView()
	embeds := GetExtantEmbeds(post)
	if len(embeds) != 0 {
		t.Errorf("expected 0 embeds, got %d", len(embeds))
	}
}

func TestGetExtantEmbeds_images(t *testing.T) {
	post := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Embed = &bsky.FeedDefs_PostView_Embed{
			EmbedImages_View: &bsky.EmbedImages_View{
				Images: []*bsky.EmbedImages_ViewImage{
					{Fullsize: "https://example.com/img1.jpg"},
					{Fullsize: "https://example.com/img2.jpg"},
				},
			},
		}
	})
	embeds := GetExtantEmbeds(post)
	if len(embeds) != 2 {
		t.Fatalf("expected 2 embeds, got %d", len(embeds))
	}
	if embeds[0] != "https://example.com/img1.jpg" {
		t.Errorf("expected img1, got %q", embeds[0])
	}
	if embeds[1] != "https://example.com/img2.jpg" {
		t.Errorf("expected img2, got %q", embeds[1])
	}
}

func TestGetExtantEmbeds_video(t *testing.T) {
	post := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Embed = &bsky.FeedDefs_PostView_Embed{
			EmbedVideo_View: &bsky.EmbedVideo_View{
				Playlist: "https://example.com/video.m3u8",
			},
		}
	})
	embeds := GetExtantEmbeds(post)
	if len(embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(embeds))
	}
	if embeds[0] != "https://example.com/video.m3u8" {
		t.Errorf("expected video playlist, got %q", embeds[0])
	}
}

func TestGetExtantEmbeds_imageAndVideo(t *testing.T) {
	post := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Embed = &bsky.FeedDefs_PostView_Embed{
			EmbedImages_View: &bsky.EmbedImages_View{
				Images: []*bsky.EmbedImages_ViewImage{
					{Fullsize: "https://example.com/img.jpg"},
				},
			},
			EmbedVideo_View: &bsky.EmbedVideo_View{
				Playlist: "https://example.com/video.m3u8",
			},
		}
	})
	embeds := GetExtantEmbeds(post)
	if len(embeds) != 2 {
		t.Fatalf("expected 2 embeds, got %d", len(embeds))
	}
}

func TestGetExtantEmbeds_nilPost(t *testing.T) {
	embeds := GetExtantEmbeds(nil)
	if embeds != nil {
		t.Errorf("expected nil, got %v", embeds)
	}
}

func TestGetPostThread_invalidURL(t *testing.T) {
	_, err := GetPostThread(context.TODO(), nil, "invalid-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

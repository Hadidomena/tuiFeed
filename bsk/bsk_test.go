package bsk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/api/atproto"
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
	post := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.IndexedAt = "2024-01-15T10:00:01Z"
	})
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
	if info.IndexedAt != "2024-01-15T10:00:01Z" {
		t.Errorf("expected '2024-01-15T10:00:01Z', got %q", info.IndexedAt)
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
		p.IndexedAt = "2024-06-01T12:00:00Z"
	})
	info := ExtractPostInfo(post)
	if info.Text != "" {
		t.Errorf("expected empty text, got %q", info.Text)
	}
	if info.CreatedAt != "" {
		t.Errorf("expected empty CreatedAt, got %q", info.CreatedAt)
	}
	if info.IndexedAt != "2024-06-01T12:00:00Z" {
		t.Errorf("expected '2024-06-01T12:00:00Z', got %q", info.IndexedAt)
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

func TestGetAuthorFeedCursor_success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "com.atproto.identity.resolveHandle"):
			_ = json.NewEncoder(w).Encode(atproto.IdentityResolveHandle_Output{
				Did: "did:plc:test123",
			})
		case strings.Contains(r.URL.Path, "app.bsky.feed.getAuthorFeed"):
			_ = json.NewEncoder(w).Encode(&bsky.FeedGetAuthorFeed_Output{
				Cursor: ptr("next-cursor-abc"),
				Feed: []*bsky.FeedDefs_FeedViewPost{
					{
						Post: &bsky.FeedDefs_PostView{
							Uri:       "at://did:plc:test123/app.bsky.feed.post/1",
							Cid:       "bafy-test",
							IndexedAt: "2024-06-01T12:00:00Z",
							Author: &bsky.ActorDefs_ProfileViewBasic{
								Did:         "did:plc:test123",
								Handle:      "test.bsky.social",
								DisplayName: ptr("Test User"),
							},
							Record: &util.LexiconTypeDecoder{
								Val: &bsky.FeedPost{
									Text:      "Hello world",
									CreatedAt: "2024-06-01T12:00:00Z",
								},
							},
							LikeCount:  ptr[int64](42),
							ReplyCount: ptr[int64](7),
						},
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.Host = server.URL

	ctx := context.Background()
	posts, cursor, err := GetAuthorFeedCursor(ctx, client, "test.bsky.social", "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if cursor != "next-cursor-abc" {
		t.Errorf("expected cursor 'next-cursor-abc', got %q", cursor)
	}
	p := posts[0]
	if p.AuthorHandle != "test.bsky.social" {
		t.Errorf("expected handle 'test.bsky.social', got %q", p.AuthorHandle)
	}
	if p.IndexedAt != "2024-06-01T12:00:00Z" {
		t.Errorf("expected IndexedAt '2024-06-01T12:00:00Z', got %q", p.IndexedAt)
	}
	if p.Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got %q", p.Text)
	}
	if p.LikeCount != 42 {
		t.Errorf("expected LikeCount 42, got %d", p.LikeCount)
	}
	if p.CreatedAt != "2024-06-01T12:00:00Z" {
		t.Errorf("expected CreatedAt '2024-06-01T12:00:00Z', got %q", p.CreatedAt)
	}
}

func TestGetAuthorFeedCursor_emptyFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "com.atproto.identity.resolveHandle"):
			_ = json.NewEncoder(w).Encode(atproto.IdentityResolveHandle_Output{
				Did: "did:plc:test123",
			})
		case strings.Contains(r.URL.Path, "app.bsky.feed.getAuthorFeed"):
			_ = json.NewEncoder(w).Encode(&bsky.FeedGetAuthorFeed_Output{
				Feed: []*bsky.FeedDefs_FeedViewPost{},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.Host = server.URL

	ctx := context.Background()
	posts, cursor, err := GetAuthorFeedCursor(ctx, client, "test.bsky.social", "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
	if cursor != "" {
		t.Errorf("expected empty cursor, got %q", cursor)
	}
}

func TestGetAuthorFeedCursor_noCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "com.atproto.identity.resolveHandle"):
			_ = json.NewEncoder(w).Encode(atproto.IdentityResolveHandle_Output{
				Did: "did:plc:test123",
			})
		case strings.Contains(r.URL.Path, "app.bsky.feed.getAuthorFeed"):
			_ = json.NewEncoder(w).Encode(&bsky.FeedGetAuthorFeed_Output{
				Cursor: nil,
				Feed: []*bsky.FeedDefs_FeedViewPost{
					{
						Post: &bsky.FeedDefs_PostView{
							Uri:       "at://did:plc:test123/app.bsky.feed.post/1",
							Cid:       "bafy-test",
							IndexedAt: "2024-06-01T12:00:00Z",
							Author: &bsky.ActorDefs_ProfileViewBasic{
								Did:    "did:plc:test123",
								Handle: "test.bsky.social",
							},
							Record: &util.LexiconTypeDecoder{
								Val: &bsky.FeedPost{
									Text:      "Last page",
									CreatedAt: "2024-06-01T12:00:00Z",
								},
							},
						},
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.Host = server.URL

	ctx := context.Background()
	posts, cursor, err := GetAuthorFeedCursor(ctx, client, "test.bsky.social", "some-cursor", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if cursor != "" {
		t.Errorf("expected empty cursor for nil response cursor, got %q", cursor)
	}
}

func TestGetAuthorFeedCursor_resolveError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "com.atproto.identity.resolveHandle") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "HandleNotFound",
				"message": "handle not found",
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.Host = server.URL

	ctx := context.Background()
	_, _, err := GetAuthorFeedCursor(ctx, client, "nonexistent.bsky.social", "", 50)
	if err == nil {
		t.Fatal("expected error for failed handle resolution")
	}
}

func TestGetAuthorFeedCursor_feedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "com.atproto.identity.resolveHandle"):
			_ = json.NewEncoder(w).Encode(atproto.IdentityResolveHandle_Output{
				Did: "did:plc:test123",
			})
		case strings.Contains(r.URL.Path, "app.bsky.feed.getAuthorFeed"):
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "InternalError",
				"message": "something went wrong",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.Host = server.URL

	ctx := context.Background()
	_, _, err := GetAuthorFeedCursor(ctx, client, "test.bsky.social", "", 50)
	if err == nil {
		t.Fatal("expected error for failed feed fetch")
	}
}

func TestGetAuthorFeedCursor_nilPosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "com.atproto.identity.resolveHandle"):
			_ = json.NewEncoder(w).Encode(atproto.IdentityResolveHandle_Output{
				Did: "did:plc:test123",
			})
		case strings.Contains(r.URL.Path, "app.bsky.feed.getAuthorFeed"):
			_ = json.NewEncoder(w).Encode(&bsky.FeedGetAuthorFeed_Output{
				Cursor: ptr("next-cursor"),
				Feed: []*bsky.FeedDefs_FeedViewPost{
					{Post: nil},
					{
						Post: &bsky.FeedDefs_PostView{
							Uri:       "at://did:plc:test123/app.bsky.feed.post/1",
							Cid:       "bafy-test",
							IndexedAt: "2024-06-01T12:00:00Z",
							Author: &bsky.ActorDefs_ProfileViewBasic{
								Did:    "did:plc:test123",
								Handle: "test.bsky.social",
							},
							Record: &util.LexiconTypeDecoder{
								Val: &bsky.FeedPost{
									Text:      "Real post",
									CreatedAt: "2024-06-01T12:00:00Z",
								},
							},
						},
					},
					{Post: nil},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient()
	client.Host = server.URL

	ctx := context.Background()
	posts, cursor, err := GetAuthorFeedCursor(ctx, client, "test.bsky.social", "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post after skipping nil entries, got %d", len(posts))
	}
	if posts[0].Text != "Real post" {
		t.Errorf("expected 'Real post', got %q", posts[0].Text)
	}
	if cursor != "next-cursor" {
		t.Errorf("expected cursor 'next-cursor', got %q", cursor)
	}
}

package bsk

import (
	"context"
	"encoding/json"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"

	"github.com/Hadidomena/tuiFeed/internal/testutil"
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

func newTestClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *xrpc.Client) {
	t.Helper()
	server := testutil.NewTestServer(t, handler)
	client := NewClient()
	client.Host = server.URL
	return server, client
}

type route struct {
	path    string
	handler http.HandlerFunc
}

func newJSONRouter(routes ...route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		for _, rt := range routes {
			if strings.Contains(r.URL.Path, rt.path) {
				rt.handler(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

func writeResolveHandle(w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(atproto.IdentityResolveHandle_Output{
		Did: "did:plc:test123",
	})
}

func withImageEnv(t *testing.T, kittyVal, termProgVal, termVal string, fn func()) {
	t.Helper()
	orig1 := os.Getenv("KITTY_WINDOW_ID")
	orig2 := os.Getenv("TERM_PROGRAM")
	orig3 := os.Getenv("TERM")
	t.Cleanup(func() {
		os.Setenv("KITTY_WINDOW_ID", orig1)
		os.Setenv("TERM_PROGRAM", orig2)
		os.Setenv("TERM", orig3)
	})
	detectedProto = -1
	os.Setenv("KITTY_WINDOW_ID", kittyVal)
	os.Setenv("TERM_PROGRAM", termProgVal)
	os.Setenv("TERM", termVal)
	fn()
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

func TestGetPostThread_success(t *testing.T) {
	_, client := newTestClient(t, newJSONRouter(
		route{"com.atproto.identity.resolveHandle", func(w http.ResponseWriter, r *http.Request) {
			writeResolveHandle(w)
		}},
		route{"app.bsky.feed.getPostThread", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(&bsky.FeedGetPostThread_Output{
				Thread: &bsky.FeedGetPostThread_Output_Thread{
					FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
						Post: makePostView(func(p *bsky.FeedDefs_PostView) {
							p.Uri = "at://did:plc:test123/app.bsky.feed.post/abc"
							p.Cid = "bafy-test"
							p.IndexedAt = "2024-06-01T12:00:00Z"
							p.Author.Did = "did:plc:test123"
							p.Record.Val.(*bsky.FeedPost).Text = "This is a thread post"
							p.Record.Val.(*bsky.FeedPost).CreatedAt = "2024-06-01T12:00:00Z"
							p.LikeCount = ptr[int64](10)
							p.ReplyCount = ptr[int64](3)
						}),
						Parent: &bsky.FeedDefs_ThreadViewPost_Parent{
							FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
								Post: &bsky.FeedDefs_PostView{
									Uri:       "at://did:plc:parent/app.bsky.feed.post/parent",
									IndexedAt: "2024-06-01T11:00:00Z",
									Author: &bsky.ActorDefs_ProfileViewBasic{
										Handle: "parent.bsky.social",
									},
									Record: &util.LexiconTypeDecoder{
										Val: &bsky.FeedPost{
											Text:      "Parent post",
											CreatedAt: "2024-06-01T11:00:00Z",
										},
									},
								},
							},
						},
						Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
							{
								FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
									Post: &bsky.FeedDefs_PostView{
										Uri:       "at://did:plc:reply/app.bsky.feed.post/reply",
										IndexedAt: "2024-06-01T13:00:00Z",
										Author: &bsky.ActorDefs_ProfileViewBasic{
											Handle: "replier.bsky.social",
										},
										Record: &util.LexiconTypeDecoder{
											Val: &bsky.FeedPost{
												Text:      "Reply post",
												CreatedAt: "2024-06-01T13:00:00Z",
											},
										},
									},
								},
							},
						},
					},
				},
			})
		}},
	))
	thread, err := GetPostThread(context.Background(), client, "https://bsky.app/profile/test.bsky.social/post/abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thread.Post.Text != "This is a thread post" {
		t.Errorf("expected 'This is a thread post', got %q", thread.Post.Text)
	}
	if thread.Parent == nil {
		t.Fatal("expected parent post")
	}
	if thread.Parent.Text != "Parent post" {
		t.Errorf("expected 'Parent post', got %q", thread.Parent.Text)
	}
	if len(thread.Replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(thread.Replies))
	}
	if thread.Replies[0].Text != "Reply post" {
		t.Errorf("expected 'Reply post', got %q", thread.Replies[0].Text)
	}
}

func TestGetPostThread_shortURL(t *testing.T) {
	_, err := GetPostThread(context.Background(), nil, "https://bsky.app/test")
	if err == nil {
		t.Fatal("expected error for short URL")
	}
}

func TestGetPostThread_resolveError(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	_, err := GetPostThread(context.Background(), client, "https://bsky.app/profile/test.bsky.social/post/abc")
	if err == nil {
		t.Fatal("expected error for failed handle resolution")
	}
}

func TestGetPostThread_threadError(t *testing.T) {
	_, client := newTestClient(t, newJSONRouter(
		route{"com.atproto.identity.resolveHandle", func(w http.ResponseWriter, r *http.Request) {
			writeResolveHandle(w)
		}},
	))
	_, err := GetPostThread(context.Background(), client, "https://bsky.app/profile/test.bsky.social/post/abc")
	if err == nil {
		t.Fatal("expected error for failed thread fetch")
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

func TestBuildThreadTree_nil(t *testing.T) {
	node := BuildThreadTree(nil)
	if node != nil {
		t.Errorf("expected nil, got %v", node)
	}
}

func TestBuildThreadTree_nilPost(t *testing.T) {
	node := BuildThreadTree(&bsky.FeedDefs_ThreadViewPost{Post: nil})
	if node != nil {
		t.Errorf("expected nil, got %v", node)
	}
}

func TestBuildThreadTree_leaf(t *testing.T) {
	post := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:test/app.bsky.feed.post/leaf"
		p.IndexedAt = "2024-06-01T12:00:00Z"
	})

	node := BuildThreadTree(&bsky.FeedDefs_ThreadViewPost{
		Post: post,
	})

	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.URI != "at://did:plc:test/app.bsky.feed.post/leaf" {
		t.Errorf("expected URI, got %q", node.URI)
	}
	if node.Parent != nil {
		t.Errorf("expected nil parent, got %v", node.Parent)
	}
	if len(node.Replies) != 0 {
		t.Errorf("expected 0 replies, got %d", len(node.Replies))
	}
	if node.Post.Text != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", node.Post.Text)
	}
}

func TestBuildThreadTree_withParent(t *testing.T) {
	parent := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:parent/app.bsky.feed.post/parent"
		p.Author.Handle = "parent.bsky.social"
		p.IndexedAt = "2024-06-01T11:00:00Z"
	})
	child := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:child/app.bsky.feed.post/child"
		p.IndexedAt = "2024-06-01T12:00:00Z"
	})

	node := BuildThreadTree(&bsky.FeedDefs_ThreadViewPost{
		Post: child,
		Parent: &bsky.FeedDefs_ThreadViewPost_Parent{
			FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
				Post: parent,
			},
		},
	})

	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.Parent == nil {
		t.Fatal("expected parent node")
	}
	if node.Parent.URI != "at://did:plc:parent/app.bsky.feed.post/parent" {
		t.Errorf("expected parent URI, got %q", node.Parent.URI)
	}
	if node.Parent.Post.AuthorHandle != "parent.bsky.social" {
		t.Errorf("expected parent handle, got %q", node.Parent.Post.AuthorHandle)
	}
}

func TestBuildThreadTree_withReplies(t *testing.T) {
	root := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:root/app.bsky.feed.post/root"
		p.IndexedAt = "2024-06-01T12:00:00Z"
	})
	reply1 := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:r1/app.bsky.feed.post/r1"
		p.Author.Handle = "replier1.bsky.social"
		p.IndexedAt = "2024-06-01T13:00:00Z"
	})
	reply2 := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:r2/app.bsky.feed.post/r2"
		p.Author.Handle = "replier2.bsky.social"
		p.IndexedAt = "2024-06-01T14:00:00Z"
	})

	node := BuildThreadTree(&bsky.FeedDefs_ThreadViewPost{
		Post: root,
		Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
			{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{Post: reply1}},
			{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{Post: reply2}},
		},
	})

	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if len(node.Replies) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(node.Replies))
	}
	if node.Replies[0].URI != "at://did:plc:r1/app.bsky.feed.post/r1" {
		t.Errorf("expected reply1 URI, got %q", node.Replies[0].URI)
	}
	if node.Replies[1].Post.AuthorHandle != "replier2.bsky.social" {
		t.Errorf("expected replier2 handle, got %q", node.Replies[1].Post.AuthorHandle)
	}
}

func TestBuildThreadTree_nestedReplies(t *testing.T) {
	root := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:root/app.bsky.feed.post/root"
		p.IndexedAt = "2024-06-01T12:00:00Z"
	})
	reply := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:reply/app.bsky.feed.post/reply"
		p.IndexedAt = "2024-06-01T13:00:00Z"
	})
	nested := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:nested/app.bsky.feed.post/nested"
		p.IndexedAt = "2024-06-01T14:00:00Z"
	})

	node := BuildThreadTree(&bsky.FeedDefs_ThreadViewPost{
		Post: root,
		Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
			{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
				Post: reply,
				Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
					{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{Post: nested}},
				},
			}},
		},
	})

	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if len(node.Replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(node.Replies))
	}
	if node.Replies[0].URI != "at://did:plc:reply/app.bsky.feed.post/reply" {
		t.Errorf("expected reply URI, got %q", node.Replies[0].URI)
	}
	if len(node.Replies[0].Replies) != 1 {
		t.Fatalf("expected 1 nested reply, got %d", len(node.Replies[0].Replies))
	}
	if node.Replies[0].Replies[0].URI != "at://did:plc:nested/app.bsky.feed.post/nested" {
		t.Errorf("expected nested URI, got %q", node.Replies[0].Replies[0].URI)
	}
}

func TestBuildThreadTree_skipsNonPostReplies(t *testing.T) {
	root := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:root/app.bsky.feed.post/root"
		p.IndexedAt = "2024-06-01T12:00:00Z"
	})

	node := BuildThreadTree(&bsky.FeedDefs_ThreadViewPost{
		Post: root,
		Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
			{FeedDefs_NotFoundPost: &bsky.FeedDefs_NotFoundPost{}},
			{FeedDefs_BlockedPost: &bsky.FeedDefs_BlockedPost{}},
		},
	})

	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if len(node.Replies) != 0 {
		t.Errorf("expected 0 replies (skipped non-post types), got %d", len(node.Replies))
	}
}

func TestBuildThreadTree_childParentLink(t *testing.T) {
	root := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:root/app.bsky.feed.post/root"
		p.IndexedAt = "2024-06-01T12:00:00Z"
	})
	reply := makePostView(func(p *bsky.FeedDefs_PostView) {
		p.Uri = "at://did:plc:reply/app.bsky.feed.post/reply"
		p.IndexedAt = "2024-06-01T13:00:00Z"
	})

	node := BuildThreadTree(&bsky.FeedDefs_ThreadViewPost{
		Post: root,
		Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
			{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{Post: reply}},
		},
	})

	if len(node.Replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(node.Replies))
	}
	if node.Replies[0].Parent != node {
		t.Error("expected child.Parent to point to root node")
	}
}

func TestGetThreadByURI_success(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePostView(func(p *bsky.FeedDefs_PostView) {
						p.Uri = "at://did:plc:test/app.bsky.feed.post/abc"
						p.Cid = "bafy-test"
						p.IndexedAt = "2024-06-01T12:00:00Z"
						p.Author.Did = "did:plc:test"
						p.Record.Val.(*bsky.FeedPost).Text = "Thread root"
						p.Record.Val.(*bsky.FeedPost).CreatedAt = "2024-06-01T12:00:00Z"
						p.LikeCount = ptr[int64](10)
						p.ReplyCount = ptr[int64](2)
					}),
					Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
						{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
							Post: &bsky.FeedDefs_PostView{
								Uri:       "at://did:plc:reply/app.bsky.feed.post/r1",
								IndexedAt: "2024-06-01T13:00:00Z",
								Author: &bsky.ActorDefs_ProfileViewBasic{
									Handle: "replier.bsky.social",
								},
								Record: &util.LexiconTypeDecoder{
									Val: &bsky.FeedPost{
										Text:      "A reply",
										CreatedAt: "2024-06-01T13:00:00Z",
									},
								},
							},
						}},
					},
				},
			},
		})
	}))
	node, err := GetThreadByURI(context.Background(), client, "at://did:plc:test/app.bsky.feed.post/abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.URI != "at://did:plc:test/app.bsky.feed.post/abc" {
		t.Errorf("expected URI, got %q", node.URI)
	}
	if node.Post.Text != "Thread root" {
		t.Errorf("expected 'Thread root', got %q", node.Post.Text)
	}
	if len(node.Replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(node.Replies))
	}
	if node.Replies[0].Post.Text != "A reply" {
		t.Errorf("expected 'A reply', got %q", node.Replies[0].Post.Text)
	}
	if node.Replies[0].Parent != node {
		t.Error("expected reply.Parent to point to root")
	}
}

func TestGetThreadByURI_apiError(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	_, err := GetThreadByURI(context.Background(), client, "at://did:plc:test/app.bsky.feed.post/abc")
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestGetThreadByURI_nilThread(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: nil,
			},
		})
	}))
	_, err := GetThreadByURI(context.Background(), client, "at://did:plc:test/app.bsky.feed.post/abc")
	if err == nil {
		t.Fatal("expected error for nil thread")
	}
}

func TestGetThreadByURI_nilOutputThread(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetPostThread_Output{
			Thread: nil,
		})
	}))
	_, err := GetThreadByURI(context.Background(), client, "at://did:plc:test/app.bsky.feed.post/abc")
	if err == nil {
		t.Fatal("expected error for nil Thread")
	}
}
func TestGetThreadByURI_nilPostInThreadView(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: nil,
				},
			},
		})
	}))
	_, err := GetThreadByURI(context.Background(), client, "at://did:plc:test/app.bsky.feed.post/abc")
	if err == nil {
		t.Fatal("expected error for nil Post in ThreadViewPost")
	}
}
func TestGetAuthorFeedCursor_success(t *testing.T) {
	_, client := newTestClient(t, newJSONRouter(
		route{"com.atproto.identity.resolveHandle", func(w http.ResponseWriter, r *http.Request) {
			writeResolveHandle(w)
		}},
		route{"app.bsky.feed.getAuthorFeed", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(&bsky.FeedGetAuthorFeed_Output{
				Cursor: ptr("next-cursor-abc"),
				Feed: []*bsky.FeedDefs_FeedViewPost{
					{
						Post: makePostView(func(p *bsky.FeedDefs_PostView) {
							p.Uri = "at://did:plc:test123/app.bsky.feed.post/1"
							p.Cid = "bafy-test"
							p.IndexedAt = "2024-06-01T12:00:00Z"
							p.Author.Did = "did:plc:test123"
							p.Record.Val.(*bsky.FeedPost).CreatedAt = "2024-06-01T12:00:00Z"
						}),
					},
				},
			})
		}},
	))
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
	_, client := newTestClient(t, newJSONRouter(
		route{"com.atproto.identity.resolveHandle", func(w http.ResponseWriter, r *http.Request) {
			writeResolveHandle(w)
		}},
		route{"app.bsky.feed.getAuthorFeed", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(&bsky.FeedGetAuthorFeed_Output{
				Feed: []*bsky.FeedDefs_FeedViewPost{},
			})
		}},
	))
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
	_, client := newTestClient(t, newJSONRouter(
		route{"com.atproto.identity.resolveHandle", func(w http.ResponseWriter, r *http.Request) {
			writeResolveHandle(w)
		}},
		route{"app.bsky.feed.getAuthorFeed", func(w http.ResponseWriter, r *http.Request) {
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
		}},
	))
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
	_, client := newTestClient(t, newJSONRouter(
		route{"com.atproto.identity.resolveHandle", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "HandleNotFound",
				"message": "handle not found",
			})
		}},
	))
	ctx := context.Background()
	_, _, err := GetAuthorFeedCursor(ctx, client, "nonexistent.bsky.social", "", 50)
	if err == nil {
		t.Fatal("expected error for failed handle resolution")
	}
}

func TestGetAuthorFeedCursor_feedError(t *testing.T) {
	_, client := newTestClient(t, newJSONRouter(
		route{"com.atproto.identity.resolveHandle", func(w http.ResponseWriter, r *http.Request) {
			writeResolveHandle(w)
		}},
		route{"app.bsky.feed.getAuthorFeed", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "InternalError",
				"message": "something went wrong",
			})
		}},
	))
	ctx := context.Background()
	_, _, err := GetAuthorFeedCursor(ctx, client, "test.bsky.social", "", 50)
	if err == nil {
		t.Fatal("expected error for failed feed fetch")
	}
}

func TestFormatPost(t *testing.T) {
	item := FeedItem{
		PostInfo: PostInfo{
			AuthorDisplayName: "Test User",
			AuthorHandle:      "test.bsky.social",
			Text:              "Hello world",
			LikeCount:         42,
			ReplyCount:        7,
			CreatedAt:         "2024-01-15T10:00:00Z",
			Embeds:            []string{"https://example.com/img.jpg"},
		},
		URI: "at://did:plc:test/app.bsky.feed.post/1",
	}
	result := FormatPost(item, -1, 0, 0)
	if !strings.Contains(result, "Test User") {
		t.Errorf("expected display name in output")
	}
	if !strings.Contains(result, "test.bsky.social") {
		t.Errorf("expected handle in output")
	}
	if !strings.Contains(result, "Hello world") {
		t.Errorf("expected text in output")
	}
	if !strings.Contains(result, "42") {
		t.Errorf("expected like count in output")
	}
	if !strings.Contains(result, "https://example.com/img.jpg") {
		t.Errorf("expected embed URL in output")
	}
}

func TestFormatPost_empty(t *testing.T) {
	item := FeedItem{
		PostInfo: PostInfo{},
	}
	result := FormatPost(item, -1, 0, 0)
	if !strings.Contains(result, "(@)") {
		t.Errorf("expected empty handle format in output, got: %s", result)
	}
}

func TestGetAuthorFeed_success(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetAuthorFeed_Output{
			Feed: []*bsky.FeedDefs_FeedViewPost{
				{
					Post: makePostView(func(p *bsky.FeedDefs_PostView) {
						p.Uri = "at://did:plc:test/app.bsky.feed.post/1"
						p.Cid = "bafy-test"
						p.IndexedAt = "2024-06-01T12:00:00Z"
						p.Author.Did = "did:plc:test"
						p.Record.Val.(*bsky.FeedPost).Text = "Hello from feed"
						p.Record.Val.(*bsky.FeedPost).CreatedAt = "2024-06-01T12:00:00Z"
						p.LikeCount = ptr[int64](10)
						p.ReplyCount = ptr[int64](2)
					}),
				},
			},
		})
	}))
	items, err := GetAuthorFeed(context.Background(), client, "test.bsky.social", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Text != "Hello from feed" {
		t.Errorf("expected 'Hello from feed', got %q", items[0].Text)
	}
	if items[0].URI != "at://did:plc:test/app.bsky.feed.post/1" {
		t.Errorf("expected URI, got %q", items[0].URI)
	}
}

func TestGetAuthorFeed_withNilPosts(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetAuthorFeed_Output{
			Feed: []*bsky.FeedDefs_FeedViewPost{
				{Post: nil},
				{
					Post: &bsky.FeedDefs_PostView{
						Uri:       "at://did:plc:test/app.bsky.feed.post/1",
						Cid:       "bafy-test",
						IndexedAt: "2024-06-01T12:00:00Z",
						Author: &bsky.ActorDefs_ProfileViewBasic{
							Did:    "did:plc:test",
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
	}))
	items, err := GetAuthorFeed(context.Background(), client, "test.bsky.social", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Text != "Real post" {
		t.Errorf("expected 'Real post', got %q", items[0].Text)
	}
}

func TestGetAuthorFeed_error(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	_, err := GetAuthorFeed(context.Background(), client, "test.bsky.social", 50)
	if err == nil {
		t.Fatal("expected error for feed fetch failure")
	}
}

func TestDetectImageProtocol_kittyWindowId(t *testing.T) {
	withImageEnv(t, "1234", "", "", func() {
		proto := DetectImageProtocol()
		if proto != ProtoKitty {
			t.Errorf("expected ProtoKitty, got %v", proto)
		}
	})
}

func TestDetectImageProtocol_wezterm(t *testing.T) {
	withImageEnv(t, "", "WezTerm", "xterm-256color", func() {
		proto := DetectImageProtocol()
		if proto != ProtoKitty {
			t.Errorf("expected ProtoKitty, got %v", proto)
		}
	})
}

func TestDetectImageProtocol_kittyTerm(t *testing.T) {
	withImageEnv(t, "", "", "xterm-kitty", func() {
		proto := DetectImageProtocol()
		if proto != ProtoKitty {
			t.Errorf("expected ProtoKitty, got %v", proto)
		}
	})
}

func TestDetectImageProtocol_sixelDefault(t *testing.T) {
	withImageEnv(t, "", "", "xterm-256color", func() {
		proto := DetectImageProtocol()
		if proto != ProtoSixel {
			t.Errorf("expected ProtoSixel, got %v", proto)
		}
	})
}

func TestDetectImageProtocol_cached(t *testing.T) {
	withImageEnv(t, "", "", "xterm-256color", func() {
		detectedProto = ProtoKitty
		proto := DetectImageProtocol()
		if proto != ProtoKitty {
			t.Errorf("expected cached ProtoKitty, got %v", proto)
		}
		detectedProto = -1
	})
}

func TestDetectImageProtocol_iterm(t *testing.T) {
	withImageEnv(t, "", "iTerm.app", "", func() {
		proto := DetectImageProtocol()
		if proto != ProtoSixel {
			t.Errorf("expected ProtoSixel for iTerm, got %v", proto)
		}
	})
}

func TestDetectImageProtocol_ghostty(t *testing.T) {
	withImageEnv(t, "", "ghostty", "xterm-ghostty", func() {
		proto := DetectImageProtocol()
		if proto != ProtoKitty {
			t.Errorf("expected ProtoKitty for Ghostty, got %v", proto)
		}
	})
}

func TestDetectImageProtocol_appleTerminal(t *testing.T) {
	withImageEnv(t, "", "Apple_Terminal", "xterm-256color", func() {
		proto := DetectImageProtocol()
		if proto != ProtoNone {
			t.Errorf("expected ProtoNone for Apple Terminal, got %v", proto)
		}
	})
}

func TestOpenerFor(t *testing.T) {
	tests := []struct {
		goos string
		cmd  string
		args []string
	}{
		{"darwin", "open", nil},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler"}},
		{"linux", "xdg-open", nil},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			cmd, args := openerFor(tt.goos)
			if cmd != tt.cmd {
				t.Errorf("expected cmd %q, got %q", tt.cmd, cmd)
			}
			if len(args) != len(tt.args) {
				t.Fatalf("expected args %v, got %v", tt.args, args)
			}
			for i := range args {
				if args[i] != tt.args[i] {
					t.Errorf("expected arg %q, got %q", tt.args[i], args[i])
				}
			}
		})
	}
}
func TestRenderImage_downloadError(t *testing.T) {
	server := testutil.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	server.Close()

	_, err := RenderImage(server.URL+"/nonexistent", 0, 40, 18)
	if err == nil {
		t.Fatal("expected download error")
	}
}

func TestRenderImage_notAnImage(t *testing.T) {
	server := testutil.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write([]byte("<html></html>"))
		if err != nil {
			t.Fatal("error during Write")
		}
	}))

	_, err := RenderImage(server.URL, 0, 40, 18)
	if err == nil {
		t.Fatal("expected 'not an image' error")
	}
}

func TestRenderImage_decodeError(t *testing.T) {
	server := testutil.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, err := w.Write([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0})
		if err != nil {
			t.Fatal("error during Write")
		}
	}))

	_, err := RenderImage(server.URL, 0, 40, 18)
	if err == nil {
		t.Fatal("expected decode error for invalid image data")
	}
}

func TestResizeToFit_noScalingNeeded(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 90, 100))
	resized := resizeToFit(img, 40, 18)
	if resized != img {
		t.Error("expected original image when no scaling needed")
	}
}

func TestResizeToFit_scaleDown(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 500))
	resized := resizeToFit(img, 40, 18)
	if resized == img {
		t.Error("expected a resized image")
	}
	bounds := resized.Bounds()
	targetW := 40 * 9
	targetH := 18 * 20
	if bounds.Dx() > targetW || bounds.Dy() > targetH {
		t.Errorf("expected within %dx%d, got %dx%d", targetW, targetH, bounds.Dx(), bounds.Dy())
	}
}

func TestResizeToFit_verySmall(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4000, 5000))
	resized := resizeToFit(img, 1, 1)
	if resized == img {
		t.Error("expected a resized image")
	}
	bounds := resized.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 {
		t.Errorf("expected at least 1x1, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestResizeToFit_heightConstrained(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 400))
	resized := resizeToFit(img, 40, 5)
	if resized == img {
		t.Error("expected a resized image")
	}
	bounds := resized.Bounds()
	targetW := 40 * 9
	targetH := 5 * 20
	if bounds.Dx() > targetW || bounds.Dy() > targetH {
		t.Errorf("expected within %dx%d, got %dx%d", targetW, targetH, bounds.Dx(), bounds.Dy())
	}
}

func TestGetAuthorFeedCursor_nilPosts(t *testing.T) {
	_, client := newTestClient(t, newJSONRouter(
		route{"com.atproto.identity.resolveHandle", func(w http.ResponseWriter, r *http.Request) {
			writeResolveHandle(w)
		}},
		route{"app.bsky.feed.getAuthorFeed", func(w http.ResponseWriter, r *http.Request) {
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
		}},
	))
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

func TestGetPosts_success(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetPosts_Output{
			Posts: []*bsky.FeedDefs_PostView{
				{
					Uri:       "at://did:plc:test/app.bsky.feed.post/1",
					Cid:       "bafy-test",
					IndexedAt: "2024-06-01T12:00:00Z",
					Author: &bsky.ActorDefs_ProfileViewBasic{
						Did:         "did:plc:test",
						Handle:      "test.bsky.social",
						DisplayName: ptr("Test User"),
					},
					Record: &util.LexiconTypeDecoder{
						Val: &bsky.FeedPost{
							Text:      "Hello from getPosts",
							CreatedAt: "2024-06-01T12:00:00Z",
						},
					},
					LikeCount:  ptr[int64](42),
					ReplyCount: ptr[int64](7),
				},
				{
					Uri:       "at://did:plc:test/app.bsky.feed.post/2",
					Cid:       "bafy-test2",
					IndexedAt: "2024-06-02T12:00:00Z",
					Author: &bsky.ActorDefs_ProfileViewBasic{
						Did:    "did:plc:test",
						Handle: "test.bsky.social",
					},
					Record: &util.LexiconTypeDecoder{
						Val: &bsky.FeedPost{
							Text:      "Second post",
							CreatedAt: "2024-06-02T12:00:00Z",
						},
					},
				},
			},
		})
	}))
	items, err := GetPosts(context.Background(), client, []string{"at://uri/1", "at://uri/2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].URI != "at://did:plc:test/app.bsky.feed.post/1" {
		t.Errorf("expected URI, got %q", items[0].URI)
	}
	if items[0].Text != "Hello from getPosts" {
		t.Errorf("expected 'Hello from getPosts', got %q", items[0].Text)
	}
	if items[1].Text != "Second post" {
		t.Errorf("expected 'Second post', got %q", items[1].Text)
	}
}

func TestGetPosts_empty(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetPosts_Output{
			Posts: []*bsky.FeedDefs_PostView{},
		})
	}))
	items, err := GetPosts(context.Background(), client, []string{"at://uri/1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestGetPosts_nilEntries(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetPosts_Output{
			Posts: []*bsky.FeedDefs_PostView{
				nil,
				{
					Uri:       "at://did:plc:test/app.bsky.feed.post/1",
					Cid:       "bafy-test",
					IndexedAt: "2024-06-01T12:00:00Z",
					Author: &bsky.ActorDefs_ProfileViewBasic{
						Did:    "did:plc:test",
						Handle: "test.bsky.social",
					},
					Record: &util.LexiconTypeDecoder{
						Val: &bsky.FeedPost{
							Text:      "Real post",
							CreatedAt: "2024-06-01T12:00:00Z",
						},
					},
				},
				nil,
			},
		})
	}))
	items, err := GetPosts(context.Background(), client, []string{"at://uri/1", "at://uri/2", "at://uri/3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after skipping nil, got %d", len(items))
	}
	if items[0].Text != "Real post" {
		t.Errorf("expected 'Real post', got %q", items[0].Text)
	}
}

func TestGetPosts_error(t *testing.T) {
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	_, err := GetPosts(context.Background(), client, []string{"at://uri/1"})
	if err == nil {
		t.Fatal("expected error for failed fetch")
	}
}

func TestGetPosts_batching(t *testing.T) {
	requestCount := 0
	_, client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		query := r.URL.Query()
		uris := query["uris"]
		w.Header().Set("Content-Type", "application/json")
		posts := make([]*bsky.FeedDefs_PostView, 0, len(uris))
		for _, uri := range uris {
			posts = append(posts, &bsky.FeedDefs_PostView{
				Uri:       uri,
				Cid:       "bafy-test",
				IndexedAt: "2024-06-01T12:00:00Z",
				Author: &bsky.ActorDefs_ProfileViewBasic{
					Did:    "did:plc:test",
					Handle: "test.bsky.social",
				},
				Record: &util.LexiconTypeDecoder{
					Val: &bsky.FeedPost{
						Text:      "Post " + uri,
						CreatedAt: "2024-06-01T12:00:00Z",
					},
				},
			})
		}
		_ = json.NewEncoder(w).Encode(&bsky.FeedGetPosts_Output{
			Posts: posts,
		})
	}))
	uris := make([]string, 30)
	for i := range uris {
		uris[i] = "at://uri/" + string(rune('A'+i))
	}

	items, err := GetPosts(context.Background(), client, uris)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 30 {
		t.Errorf("expected 30 items, got %d", len(items))
	}
	if requestCount != 2 {
		t.Errorf("expected 2 batch requests, got %d", requestCount)
	}
}

func TestFormatPostListItem_basic(t *testing.T) {
	post := PostInfo{
		AuthorDisplayName: "Test User",
		AuthorHandle:      "test.bsky.social",
		Text:              "Hello world",
		LikeCount:         42,
		ReplyCount:        7,
		IndexedAt:         "2024-01-15T10:00:00Z",
	}
	result := FormatPostListItem(post, true, 0)
	if !strings.Contains(result, "> @test.bsky.social") {
		t.Errorf("expected cursor and handle, got: %s", result)
	}
	if !strings.Contains(result, "Hello world") {
		t.Errorf("expected text in output, got: %s", result)
	}
	result = FormatPostListItem(post, false, 0)
	if strings.Contains(result, "> @") {
		t.Errorf("expected no cursor when cursor=false, got: %s", result)
	}
}

func TestFormatPostListItem_truncation(t *testing.T) {
	longText := ""
	for i := 0; i < 200; i++ {
		longText += "a"
	}
	post := PostInfo{
		AuthorHandle: "test.bsky.social",
		Text:         longText,
		IndexedAt:    "2024-01-15T10:00:00Z",
	}
	result := FormatPostListItem(post, false, 0)
	if !strings.Contains(result, "...") {
		t.Errorf("expected truncated text, got: %s", result)
	}
}

func TestFormatPostListItem_newlinesReplaced(t *testing.T) {
	post := PostInfo{
		AuthorHandle: "test.bsky.social",
		Text:         "line1\nline2\nline3",
		IndexedAt:    "2024-01-15T10:00:00Z",
	}
	result := FormatPostListItem(post, false, 0)
	if strings.Contains(result, "\n") && !strings.HasSuffix(result, "\n") {
		t.Error("expected newlines in text to be replaced with spaces")
	}
}

func TestFormatPostListItem_embeds(t *testing.T) {
	post := PostInfo{
		AuthorHandle: "test.bsky.social",
		Text:         "Hello",
		Embeds:       []string{"https://example.com/img.jpg"},
		IndexedAt:    "2024-01-15T10:00:00Z",
	}
	result := FormatPostListItem(post, false, 0)
	if !strings.Contains(result, "1 attachment") {
		t.Errorf("expected attachment indicator, got: %s", result)
	}
}

func TestFormatPostListItem_emptyDisplayName(t *testing.T) {
	post := PostInfo{
		AuthorHandle: "test.bsky.social",
		Text:         "Hello",
		IndexedAt:    "2024-01-15T10:00:00Z",
	}
	result := FormatPostListItem(post, false, 0)
	if !strings.Contains(result, "(test.bsky.social)") {
		t.Errorf("expected handle as fallback display name, got: %s", result)
	}
}

func TestFormatPostListItem_shortDate(t *testing.T) {
	post := PostInfo{
		AuthorHandle: "test.bsky.social",
		Text:         "Hello",
		CreatedAt:    "2024-01",
	}
	result := FormatPostListItem(post, false, 0)
	if !strings.Contains(result, "📅 2024-01") {
		t.Errorf("expected short date, got: %s", result)
	}
}

func TestWriteMoreIndicators(t *testing.T) {
	var b strings.Builder
	WriteMoreIndicators(&b, 5, 15, 25)
	if !strings.Contains(b.String(), "more above") {
		t.Errorf("expected 'more above' when scrollPos > 0, got: %s", b.String())
	}
	if !strings.Contains(b.String(), "more below") {
		t.Errorf("expected 'more below' when end < total, got: %s", b.String())
	}
}

func TestWriteMoreIndicators_noIndicators(t *testing.T) {
	var b strings.Builder
	WriteMoreIndicators(&b, 0, 10, 10)
	if b.Len() > 0 {
		t.Errorf("expected empty output when at boundaries, got: %s", b.String())
	}
}

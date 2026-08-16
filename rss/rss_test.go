package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/extensions"
)

const rssFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Hello World</title>
      <link>https://example.com/hello</link>
      <guid>https://example.com/hello</guid>
      <pubDate>Mon, 02 Jan 2006 15:04:05 +0000</pubDate>
      <description><![CDATA[<p>Some <b>text</b> here.</p>]]></description>
      <enclosure url="https://example.com/img.png" type="image/png" length="123" />
    </item>
    <item>
      <title>Second Post</title>
      <link>https://example.com/second</link>
      <guid>https://example.com/second</guid>
      <pubDate>Tue, 03 Jan 2006 15:04:05 +0000</pubDate>
      <description>No html here</description>
    </item>
  </channel>
</rss>`

const atomFeed = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Feed</title>
  <link href="https://example.com/atom"/>
  <entry>
    <title>Atom Entry</title>
    <link href="https://example.com/atom/1"/>
    <id>tag:example.com,2006:1</id>
    <updated>2006-01-03T15:04:05Z</updated>
    <content type="html">&lt;p&gt;Atom content &lt;b&gt;here&lt;/b&gt;&lt;/p&gt;</content>
    <media:thumbnail xmlns:media="http://search.yahoo.com/mrss/" url="https://example.com/thumb.jpg"/>
  </entry>
</feed>`

func feedServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	}))
}

func TestFetchFeed_rss(t *testing.T) {
	srv := feedServer(t, rssFeed, http.StatusOK)
	defer srv.Close()

	entries, err := FetchFeed(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchFeed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	e := entries[0]
	if e.Title != "Hello World" {
		t.Errorf("unexpected title %q", e.Title)
	}
	if e.FeedTitle != "Test Feed" {
		t.Errorf("unexpected feed title %q", e.FeedTitle)
	}
	if e.ID != "https://example.com/hello" {
		t.Errorf("unexpected ID %q", e.ID)
	}
	if e.Published != "2006-01-02T15:04:05Z" {
		t.Errorf("unexpected published %q", e.Published)
	}
	if !strings.Contains(e.Text, "Some text here.") {
		t.Errorf("expected stripped text, got %q", e.Text)
	}
	if strings.Contains(e.Text, "<") {
		t.Errorf("expected HTML stripped, got %q", e.Text)
	}
	if len(e.Images) != 1 || e.Images[0] != "https://example.com/img.png" {
		t.Errorf("unexpected images %v", e.Images)
	}
}

func TestFetchFeed_atom(t *testing.T) {
	srv := feedServer(t, atomFeed, http.StatusOK)
	defer srv.Close()

	entries, err := FetchFeed(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchFeed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Title != "Atom Entry" {
		t.Errorf("unexpected title %q", e.Title)
	}
	if e.FeedTitle != "Atom Feed" {
		t.Errorf("unexpected feed title %q", e.FeedTitle)
	}
	if e.Published != "2006-01-03T15:04:05Z" {
		t.Errorf("unexpected published %q", e.Published)
	}
	if !strings.Contains(e.Text, "Atom content here") {
		t.Errorf("expected decoded text, got %q", e.Text)
	}
	if len(e.Images) != 1 || e.Images[0] != "https://example.com/thumb.jpg" {
		t.Errorf("unexpected images %v", e.Images)
	}
}

func TestFetchFeed_missingGUID(t *testing.T) {
	feed := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>No GUID</title>
    <item>
      <title>Item One</title>
      <link>https://example.com/one</link>
    </item>
  </channel>
</rss>`
	srv := feedServer(t, feed, http.StatusOK)
	defer srv.Close()

	entries, err := FetchFeed(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchFeed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "https://example.com/one" {
		t.Errorf("expected ID fallback to link, got %q", entries[0].ID)
	}
}

func TestFetchFeed_statusError(t *testing.T) {
	srv := feedServer(t, "not found", http.StatusNotFound)
	defer srv.Close()

	_, err := FetchFeed(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected status code in error, got %v", err)
	}
}

func TestFetchFeed_malformedXML(t *testing.T) {
	srv := feedServer(t, "<rss><channel>", http.StatusOK)
	defer srv.Close()

	_, err := FetchFeed(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for malformed XML")
	}
}

func TestFetchFeed_networkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijack", http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	srv.Close()

	_, err := FetchFeed(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for closed connection")
	}
}

func TestFetchFeed_ctxCanceled(t *testing.T) {
	srv := feedServer(t, rssFeed, http.StatusOK)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FetchFeed(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestFetchAll_sortsAndMerges(t *testing.T) {
	srv := feedServer(t, rssFeed, http.StatusOK)
	defer srv.Close()

	entries, err := FetchAll(context.Background(), []string{srv.URL, srv.URL})
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].Published > entries[i-1].Published {
			t.Errorf("entries not sorted descending at %d", i)
		}
	}
}

func TestFetchAll_partialFailure(t *testing.T) {
	good := feedServer(t, rssFeed, http.StatusOK)
	defer good.Close()
	bad := feedServer(t, "oops", http.StatusInternalServerError)
	defer bad.Close()

	entries, err := FetchAll(context.Background(), []string{good.URL, bad.URL})
	if err == nil {
		t.Fatal("expected aggregated error when a feed fails")
	}
	if len(entries) != 2 {
		t.Fatalf("expected partial entries from good feed, got %d", len(entries))
	}
}

func TestEntryID(t *testing.T) {
	tests := []struct {
		name string
		item *gofeed.Item
		want string
	}{
		{"guid", &gofeed.Item{GUID: "g", Link: "l"}, "g"},
		{"link fallback", &gofeed.Item{Link: "l"}, "l"},
		{"title fallback", &gofeed.Item{Title: "t"}, "t"},
		{"empty", &gofeed.Item{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EntryID(tt.item); got != tt.want {
				t.Errorf("EntryID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<p>Hello <b>World</b></p>", "Hello World"},
		{"A &amp; B", "A & B"},
		{"<img src='https://x.com/i.png'/>text", "text"},
		{"", ""},
		{"plain text", "plain text"},
	}
	for _, tt := range tests {
		if got := stripHTML(tt.in); got != tt.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatEntry_empty(t *testing.T) {
	e := Entry{Title: "T", Text: "body"}
	out := FormatEntry(e, -1, 0, 0)
	if !strings.Contains(out, "T") || !strings.Contains(out, "body") {
		t.Errorf("unexpected FormatEntry output: %q", out)
	}
}

func TestFormatEntryListItem(t *testing.T) {
	e := Entry{Title: "Title", Text: "body text", Published: "2024-01-15T10:00:00Z"}
	out := FormatEntryListItem(e, true, 80)
	if !strings.Contains(out, "Title") {
		t.Errorf("expected title in output: %q", out)
	}
	if !strings.Contains(out, "2024-01-15") {
		t.Errorf("expected date in output: %q", out)
	}
	if !strings.HasPrefix(out, ">") {
		t.Errorf("expected cursor marker: %q", out)
	}
}

func TestFormatPublished(t *testing.T) {
	if got := formatPublished("2024-01-15T10:00:00Z"); got != "2024-01-15" {
		t.Errorf("expected date-only, got %q", got)
	}
	if got := formatPublished(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestPublishedRFC3339_parsed(t *testing.T) {
	item := &gofeed.Item{Published: "Mon, 02 Jan 2006 15:04:05 +0000"}
	parsed, err := time.Parse(time.RFC1123Z, item.Published)
	if err != nil {
		t.Fatalf("setup parse: %v", err)
	}
	item.PublishedParsed = &parsed
	if got := publishedRFC3339(item); got != "2006-01-02T15:04:05Z" {
		t.Errorf("expected normalized RFC3339, got %q", got)
	}
}

func TestPublishedRFC3339_rawFallback(t *testing.T) {
	item := &gofeed.Item{Published: "2024-01-15T10:00:00Z"}
	if got := publishedRFC3339(item); got != "2024-01-15T10:00:00Z" {
		t.Errorf("expected raw published fallback, got %q", got)
	}
}

func TestExtractImages_mediaAndImg(t *testing.T) {
	item := &gofeed.Item{
		Content: `<p><img src="https://x.com/a.png"/>inline</p>`,
		Extensions: map[string]map[string][]ext.Extension{
			"media": {
				"content": {{Name: "content", Attrs: map[string]string{"url": "https://x.com/m.png"}}},
			},
		},
	}
	imgs := extractImages(item)
	want := []string{"https://x.com/m.png", "https://x.com/a.png"}
	if len(imgs) != len(want) {
		t.Fatalf("expected %d images, got %d: %v", len(want), len(imgs), imgs)
	}
	for i := range want {
		if imgs[i] != want[i] {
			t.Errorf("image[%d] = %q, want %q", i, imgs[i], want[i])
		}
	}
}

func TestExtractImages_dedupe(t *testing.T) {
	item := &gofeed.Item{
		Enclosures: []*gofeed.Enclosure{
			{URL: "https://x.com/same.png", Type: "image/png"},
		},
		Content: `<img src="https://x.com/same.png">`,
	}
	imgs := extractImages(item)
	if len(imgs) != 1 {
		t.Fatalf("expected deduped images, got %v", imgs)
	}
}

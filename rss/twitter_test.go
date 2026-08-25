package rss

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestNormalizeTwitterInput(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"@GTAGolden_", "https://xcancel.com/GTAGolden_/rss"},
		{"GTAGolden_", "https://xcancel.com/GTAGolden_/rss"},
		{"@handle/media", "https://xcancel.com/handle/media/rss"},
		{"handle/with_replies", "https://xcancel.com/handle/with_replies/rss"},
		{"x.com/GTAGolden_", "https://xcancel.com/GTAGolden_/rss"},
		{"https://x.com/GTAGolden_", "https://xcancel.com/GTAGolden_/rss"},
		{"https://www.twitter.com/GTAGolden_/media/", "https://xcancel.com/GTAGolden_/media/rss"},
		{"@Handle/Highlights", "https://xcancel.com/Handle/highlights/rss"},
		{"https://hnrss.org/frontpage", "https://hnrss.org/frontpage"},
		{"https://example.com/feed.xml", "https://example.com/feed.xml"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeTwitterInput(tt.in); got != tt.want {
			t.Errorf("NormalizeTwitterInput(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

const nitterGateFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:dc="http://purl.org/dc/elements/1.1/" version="2.0">
  <channel>
    <atom:link href="https://rss.xcancel.com/test/rss" rel="self" type="application/rss+xml" />
    <title>RSS reader not yet whitelisted!</title>
    <link>https://rss.xcancel.com/test/rss</link>
    <description>RSS reader not yet whitelist!</description>
    <language>en-us</language>
    <ttl>1</ttl>
    <item>
      <title>RSS reader not yet whitelisted!</title>
      <dc:creator>@xcancel</dc:creator>
      <description>RSS reader not yet whitelist! Please send an email rss [AT] xcancel [DOT] com with this ID to get your RSS feed reader whitelisted: 9db973f922ce8235f03b6f77657cbf46203f3fd7209e347a7db04516f8d3018d9eba452be149899c0d957d7010399de77044fb4aeb361428be2779f8e0d4ef0e</description>
    </item>
  </channel>
</rss>`

func TestFetchFeed_nitterGate(t *testing.T) {
	srv := feedServer(t, nitterGateFeed, http.StatusOK)
	defer srv.Close()

	feed, err := gofeed.NewParser().ParseString(nitterGateFeed)
	if err != nil {
		t.Fatalf("parse nitter gate feed: %v", err)
	}

	err = detectNitterGate("https://xcancel.com/test/rss", feed)
	if err == nil {
		t.Fatal("expected whitelist gate error")
	}
	if !strings.Contains(err.Error(), "rss@xcancel.com") {
		t.Errorf("expected instructions in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "9db973f922") {
		t.Errorf("expected whitelist ID in error, got %v", err)
	}
}

func TestFetchFeed_nitterGateNotAppliedToOtherFeeds(t *testing.T) {
	feed := &gofeed.Feed{Title: "RSS reader not yet whitelisted!"}
	if err := detectNitterGate("https://example.com/feed.xml", feed); err != nil {
		t.Errorf("expected no gate error for non-nitter feed, got %v", err)
	}
}

func TestExtractVideos(t *testing.T) {
	item := &gofeed.Item{
		Content: `<video src="https://x.com/a.mp4" loop muted></video><source src="https://x.com/b.mp4">text`,
	}
	videos := extractVideos(item)
	if len(videos) != 2 || videos[0] != "https://x.com/a.mp4" || videos[1] != "https://x.com/b.mp4" {
		t.Errorf("unexpected videos %v", videos)
	}
}

func TestExtractVideos_dedupeAndEnclosures(t *testing.T) {
	item := &gofeed.Item{
		Enclosures: []*gofeed.Enclosure{
			{URL: "https://x.com/same.mp4", Type: "video/mp4"},
		},
		Content: `<video src="https://x.com/same.mp4"></video>`,
	}
	videos := extractVideos(item)
	if len(videos) != 1 || videos[0] != "https://x.com/same.mp4" {
		t.Errorf("unexpected videos %v", videos)
	}
}

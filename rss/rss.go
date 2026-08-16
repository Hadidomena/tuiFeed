package rss

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Hadidomena/tuiFeed/utils"
)

const fetchTimeout = 20 * time.Second

type Entry struct {
	FeedTitle string
	FeedURL   string
	Title     string
	Link      string
	ID        string
	Published string
	Text      string
	Images    []string
}

var (
	imgTagRe = regexp.MustCompile(`(?is)<img\b[^>]*?\bsrc=["']([^"']+)["']`)
	tagRe    = regexp.MustCompile(`(?s)<[^>]*>`)
)

func FetchFeed(ctx context.Context, url string) ([]Entry, error) {
	client := &http.Client{Timeout: fetchTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", "tuiFeed/1.0 (+https://github.com/Hadidomena/tuiFeed)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("fetching %s: unexpected status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", url, err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing response body: %w", closeErr)
	}

	parser := gofeed.NewParser()
	feed, err := parser.ParseString(string(body))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", url, err)
	}

	feedTitle := strings.TrimSpace(feed.Title)
	entries := make([]Entry, 0, len(feed.Items))
	for _, item := range feed.Items {
		if item == nil {
			continue
		}
		entries = append(entries, mapEntry(feedTitle, url, item))
	}
	return entries, nil
}

func FetchAll(ctx context.Context, urls []string) ([]Entry, error) {
	var all []Entry
	var errs []string
	for _, url := range urls {
		entries, err := FetchFeed(ctx, url)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		all = append(all, entries...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Published > all[j].Published
	})

	if len(errs) > 0 {
		return all, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return all, nil
}

func mapEntry(feedTitle, feedURL string, item *gofeed.Item) Entry {
	e := Entry{
		FeedTitle: feedTitle,
		FeedURL:   feedURL,
		Title:     strings.TrimSpace(item.Title),
		Link:      strings.TrimSpace(item.Link),
		ID:        EntryID(item),
		Published: publishedRFC3339(item),
		Text:      cleanText(item),
	}
	e.Images = extractImages(item)
	return e
}

func EntryID(item *gofeed.Item) string {
	if id := strings.TrimSpace(item.GUID); id != "" {
		return id
	}
	if link := strings.TrimSpace(item.Link); link != "" {
		return link
	}
	if title := strings.TrimSpace(item.Title); title != "" {
		return title
	}
	return item.Published
}

func publishedRFC3339(item *gofeed.Item) string {
	if item.PublishedParsed != nil {
		return item.PublishedParsed.UTC().Format(time.RFC3339)
	}
	if item.UpdatedParsed != nil {
		return item.UpdatedParsed.UTC().Format(time.RFC3339)
	}
	if p := strings.TrimSpace(item.Published); p != "" {
		return p
	}
	return strings.TrimSpace(item.Updated)
}

func cleanText(item *gofeed.Item) string {
	content := item.Content
	if strings.TrimSpace(content) == "" {
		content = item.Description
	}
	return stripHTML(content)
}

func extractImages(item *gofeed.Item) []string {
	seen := make(map[string]bool)
	var images []string
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		images = append(images, u)
	}

	for _, enc := range item.Enclosures {
		if enc != nil && strings.HasPrefix(enc.Type, "image/") {
			add(enc.URL)
		}
	}

	for ns, groups := range item.Extensions {
		if ns != "media" {
			continue
		}
		for name, exts := range groups {
			switch name {
			case "content", "thumbnail":
				for _, ext := range exts {
					add(ext.Attrs["url"])
				}
			}
		}
	}

	content := item.Content
	if strings.TrimSpace(content) == "" {
		content = item.Description
	}
	for _, m := range imgTagRe.FindAllStringSubmatch(content, -1) {
		add(m[1])
	}

	return images
}

func stripHTML(s string) string {
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.TrimSpace(s)
	return s
}

func FormatEntry(e Entry, imgCursor int, wrapWidth int, maxTextLines int) string {
	var b strings.Builder
	source := e.FeedTitle
	if source == "" {
		source = e.FeedURL
	}
	fmt.Fprintf(&b, "─── %s ───\n", source)
	fmt.Fprintf(&b, "📅 %s\n", formatPublished(e.Published))
	if e.Link != "" {
		fmt.Fprintf(&b, "🔗 %s\n", e.Link)
	}
	b.WriteString("\n")

	title := e.Title
	if wrapWidth > 0 {
		title = utils.WrapText(title, wrapWidth)
	}
	b.WriteString(title)
	b.WriteString("\n\n")

	text := e.Text
	if wrapWidth > 0 {
		text = utils.WrapText(text, wrapWidth)
	}
	if maxTextLines > 0 {
		text = utils.TruncateLines(text, maxTextLines)
	}
	b.WriteString(text)
	b.WriteString("\n\n")

	if len(e.Images) > 0 {
		current := imgCursor
		if current < 0 || current >= len(e.Images) {
			current = 0
		}
		fmt.Fprintf(&b, "── Attachment %d/%d ──\n", current+1, len(e.Images))
		marker := "  "
		if imgCursor == current {
			marker = "> "
		}
		url := e.Images[current]
		if wrapWidth > 0 {
			url = utils.TruncateWidth(url, wrapWidth)
		}
		fmt.Fprintf(&b, "%s%s\n", marker, url)
	}

	return b.String()
}

func FormatEntryListItem(e Entry, cursor bool, wrapWidth int) string {
	var b strings.Builder

	cursorStr := "  "
	if cursor {
		cursorStr = "> "
	}

	fmt.Fprintf(&b, "%s%s (%s)\n", cursorStr, firstNonEmpty(e.Title, e.FeedTitle, e.Link), formatPublished(e.Published))

	text := strings.TrimSpace(e.Text)
	text = strings.ReplaceAll(text, "\n", " ")
	if wrapWidth > 0 {
		effectiveWidth := wrapWidth - 4
		if effectiveWidth < 10 {
			effectiveWidth = 10
		}
		text = utils.WrapText(text, effectiveWidth)
		text = utils.TruncateLines(text, 2)
	} else if len(text) > 120 {
		text = text[:120] + "..."
	}
	fmt.Fprintf(&b, "    %s\n", text)

	if len(e.Images) > 0 {
		fmt.Fprintf(&b, "    [%d attachment(s)]\n", len(e.Images))
	}

	return b.String()
}

func formatPublished(p string) string {
	if p == "" {
		return ""
	}
	if len(p) >= 10 {
		return p[:10]
	}
	return p
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

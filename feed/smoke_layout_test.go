package feed

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Hadidomena/tuiFeed/bsk"
)

func longPostWithManyEmbeds() []bsk.FeedItem {
	longText := strings.Repeat("This is a fairly long sentence that should wrap across multiple terminal lines. ", 30)
	embeds := []string{
		"https://example.com/image1.jpg",
		"https://example.com/image2.jpg",
		"https://example.com/image3.jpg",
		"https://example.com/image4.jpg",
		"https://example.com/image5.jpg",
	}
	return []bsk.FeedItem{{
		PostInfo: bsk.PostInfo{
			AuthorHandle:      "test.bsky.social",
			AuthorDisplayName: "Test User",
			Text:              longText,
			LikeCount:         10,
			ReplyCount:        2,
			IndexedAt:         "2024-01-15T10:00:00Z",
			Embeds:            embeds,
		},
		URI: "at://uri/1",
	}}
}

func TestLayout_fitsSmallScreen(t *testing.T) {
	for _, h := range []int{24, 40, 60} {
		m := NewStaticModel(longPostWithManyEmbeds(), "Test")
		// trigger resize -> recalc
		um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: h})
		m = um.(Model)
		view := m.View().Content
		lines := strings.Count(view, "\n")
		if lines > h {
			t.Errorf("height=%d: view has %d lines, exceeds screen", h, lines)
		}
		t.Logf("height=%d: view lines=%d pageSize=%d", h, lines, m.pageSize)
	}
}

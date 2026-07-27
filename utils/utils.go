package utils

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const DefaultPageSize = 10
const DefaultWidth = 80

func ContentWidth(termWidth int) int {
	if termWidth <= 0 {
		return DefaultWidth - 4
	}
	available := termWidth - 4
	if available > 120 {
		return 120
	}
	return available
}

func PageSize(termHeight int) int {
	if termHeight <= 0 {
		return DefaultPageSize
	}
	usable := termHeight - 6
	if usable < 3 {
		return 3
	}
	return usable
}

func DownloadURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	return data, nil
}

func Pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func ScrollWindowEnd(scrollPos, pageSize, length int) int {
	end := scrollPos + pageSize
	if end > length {
		end = length
	}
	return end
}

func CursorUp(cursor int) int {
	if cursor > 0 {
		return cursor - 1
	}
	return cursor
}

func CursorDown(cursor, length int) int {
	if cursor < length-1 {
		return cursor + 1
	}
	return cursor
}

func ScrollDown(cursor, scrollPos, pageSize, length int) (int, int) {
	if cursor >= length-1 {
		return cursor, scrollPos
	}
	cursor++
	if cursor >= scrollPos+pageSize {
		scrollPos++
	}
	return cursor, scrollPos
}

func ScrollUp(cursor, scrollPos int) (int, int) {
	if cursor <= 0 {
		return cursor, scrollPos
	}
	cursor--
	if cursor < scrollPos {
		scrollPos--
	}
	return cursor, scrollPos
}

func WriteHeader(b *strings.Builder, title string, width int) {
	b.WriteString(title + "\n")
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n\n")
}

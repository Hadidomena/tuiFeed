package utils

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mattn/go-runewidth"
)

const DefaultPageSize = 10
const DefaultWidth = 80

func ContentWidth(termWidth int) int {
	if termWidth <= 0 {
		return DefaultWidth - 4
	}
	available := termWidth - 4
	if available < 0 {
		available = 0
	}
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

func OpenExternal(target string) error {
	cmd, args := openerFor(runtime.GOOS)
	args = append(args, target)
	return exec.Command(cmd, args...).Start()
}

func openerFor(goos string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		return "xdg-open", nil
	}
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

func CenterBlock(s string, termWidth int) string {
	cw := ContentWidth(termWidth)
	leftPad := (termWidth - cw) / 2
	if leftPad <= 0 {
		return s
	}
	pad := strings.Repeat(" ", leftPad)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = pad + line
	}
	return strings.Join(lines, "\n")
}

func WrapText(text string, maxWidth int) string {
	if maxWidth <= 0 || runewidth.StringWidth(text) <= maxWidth {
		return text
	}
	var result strings.Builder
	paraLines := strings.Split(text, "\n")
	for pi, para := range paraLines {
		if pi > 0 {
			result.WriteString("\n")
		}
		result.WriteString(wrapLine(para, maxWidth))
	}
	return result.String()
}

func TextBudget(termHeight int) int {
	b := termHeight / 4
	if b < 3 {
		b = 3
	}
	if b > 18 {
		b = 18
	}
	return b
}

func TruncateLines(s string, maxLines int) string {
	if maxLines <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	truncated := lines[:maxLines]
	truncated[maxLines-1] = truncated[maxLines-1] + " ..."
	return strings.Join(truncated, "\n")
}

func TruncateWidth(s string, maxWidth int) string {
	if maxWidth <= 0 || runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	out := ""
	w := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > maxWidth-1 {
			break
		}
		out += string(r)
		w += rw
	}
	return out + "…"
}

func wrapLine(line string, maxWidth int) string {
	line = strings.TrimRight(line, " \t")
	if runewidth.StringWidth(line) <= maxWidth {
		return line
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return line
	}
	var b strings.Builder
	lineLen := 0
	for i, word := range words {
		wordWidth := runewidth.StringWidth(word)
		if i == 0 {
			b.WriteString(word)
			lineLen = wordWidth
		} else if lineLen+1+wordWidth <= maxWidth {
			b.WriteString(" ")
			b.WriteString(word)
			lineLen += 1 + wordWidth
		} else {
			b.WriteString("\n")
			b.WriteString(word)
			lineLen = wordWidth
		}
	}
	return b.String()
}

func WriteHeader(b *strings.Builder, title string, termWidth int) {
	width := ContentWidth(termWidth)
	b.WriteString(title + "\n")
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n\n")
}

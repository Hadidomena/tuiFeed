package utils

import (
	"strings"
	"testing"
)

func TestCursorUp(t *testing.T) {
	if got := CursorUp(5); got != 4 {
		t.Errorf("CursorUp(5) = %d, want 4", got)
	}
	if got := CursorUp(0); got != 0 {
		t.Errorf("CursorUp(0) = %d, want 0", got)
	}
}

func TestCursorDown(t *testing.T) {
	if got := CursorDown(0, 5); got != 1 {
		t.Errorf("CursorDown(0, 5) = %d, want 1", got)
	}
	if got := CursorDown(4, 5); got != 4 {
		t.Errorf("CursorDown(4, 5) = %d, want 4", got)
	}
}

func TestScrollDown(t *testing.T) {
	cursor, scroll := ScrollDown(0, 0, 10, 20)
	if cursor != 1 || scroll != 0 {
		t.Errorf("ScrollDown = (%d, %d), want (1, 0)", cursor, scroll)
	}
	cursor, scroll = ScrollDown(9, 0, 10, 20)
	if cursor != 10 || scroll != 1 {
		t.Errorf("ScrollDown at boundary = (%d, %d), want (10, 1)", cursor, scroll)
	}
	cursor, scroll = ScrollDown(19, 10, 10, 20)
	if cursor != 19 || scroll != 10 {
		t.Errorf("ScrollDown at end = (%d, %d), want (19, 10)", cursor, scroll)
	}
}

func TestScrollUp(t *testing.T) {
	cursor, scroll := ScrollUp(5, 0)
	if cursor != 4 || scroll != 0 {
		t.Errorf("ScrollUp = (%d, %d), want (4, 0)", cursor, scroll)
	}
	cursor, scroll = ScrollUp(0, 0)
	if cursor != 0 || scroll != 0 {
		t.Errorf("ScrollUp at zero = (%d, %d), want (0, 0)", cursor, scroll)
	}
}

func TestScrollWindowEnd(t *testing.T) {
	if got := ScrollWindowEnd(0, 10, 20); got != 10 {
		t.Errorf("ScrollWindowEnd = %d, want 10", got)
	}
	if got := ScrollWindowEnd(15, 10, 20); got != 20 {
		t.Errorf("ScrollWindowEnd near end = %d, want 20", got)
	}
}

func TestPluralize(t *testing.T) {
	if got := Pluralize(1, "y", "ies"); got != "y" {
		t.Errorf("Pluralize(1) = %q, want 'y'", got)
	}
	if got := Pluralize(0, "y", "ies"); got != "ies" {
		t.Errorf("Pluralize(0) = %q, want 'ies'", got)
	}
	if got := Pluralize(2, "y", "ies"); got != "ies" {
		t.Errorf("Pluralize(2) = %q, want 'ies'", got)
	}
}

func TestContentWidth(t *testing.T) {
	cases := []struct {
		termWidth int
		want      int
	}{
		{0, DefaultWidth - 4},
		{-5, DefaultWidth - 4},
		{3, 0},
		{4, 0},
		{20, 16},
		{80, 76},
		{130, 120},
	}
	for _, c := range cases {
		if got := ContentWidth(c.termWidth); got != c.want {
			t.Errorf("ContentWidth(%d) = %d, want %d", c.termWidth, got, c.want)
		}
	}
}

func TestPageSize(t *testing.T) {
	if got := PageSize(0); got != DefaultPageSize {
		t.Errorf("PageSize(0) = %d, want %d", got, DefaultPageSize)
	}
	if got := PageSize(3); got != 3 {
		t.Errorf("PageSize(3) = %d, want 3", got)
	}
	if got := PageSize(20); got != 14 {
		t.Errorf("PageSize(20) = %d, want 14", got)
	}
}

func TestCenterBlock(t *testing.T) {
	if got := CenterBlock("hi", 0); got != "hi" {
		t.Errorf("CenterBlock(hi, 0) = %q, want %q", got, "hi")
	}
	if got := CenterBlock("hi", 3); got != " hi" {
		t.Errorf("CenterBlock(hi, 3) = %q, want %q", got, " hi")
	}
	if got := CenterBlock("hi", 20); got != "  hi" {
		t.Errorf("CenterBlock(hi, 20) = %q, want %q", got, "  hi")
	}
	want := strings.Repeat(" ", 2) + "a\n" + strings.Repeat(" ", 2) + "b"
	if got := CenterBlock("a\nb", 20); got != want {
		t.Errorf("CenterBlock(a\\nb, 20) = %q, want %q", got, want)
	}
}

func TestWrapTextNarrow(t *testing.T) {
	if got := WrapText("hello", 0); got != "hello" {
		t.Errorf("WrapText(hello, 0) = %q, want %q", got, "hello")
	}
	if got := WrapText("hello", 10); got != "hello" {
		t.Errorf("WrapText(hello, 10) = %q, want %q", got, "hello")
	}
	if got := WrapText("hello world", 5); got != "hello\nworld" {
		t.Errorf("WrapText(hello world, 5) = %q, want %q", got, "hello\nworld")
	}
	if got := WrapText("a b c d", 3); got != "a b\nc d" {
		t.Errorf("WrapText(a b c d, 3) = %q, want %q", got, "a b\nc d")
	}
}

func TestWrapTextWideRunes(t *testing.T) {
	if got := WrapText("你 好 世 界", 3); got != "你\n好\n世\n界" {
		t.Errorf("WrapText(wide, 3) = %q, want %q", got, "你\n好\n世\n界")
	}
	if got := WrapText("一二三四五六", 5); got != "一二三四五六" {
		t.Errorf("WrapText(cjk single word, 5) = %q, want %q", got, "一二三四五六")
	}
	if got := WrapText("一二三 四五六", 5); got != "一二三\n四五六" {
		t.Errorf("WrapText(cjk words, 5) = %q, want %q", got, "一二三\n四五六")
	}
}

func TestWrapTextMultiPara(t *testing.T) {
	got := WrapText("hello world\nfoo bar", 6)
	want := "hello\nworld\nfoo\nbar"
	if got != want {
		t.Errorf("WrapText(paras, 6) = %q, want %q", got, want)
	}
}

func TestTruncateWidth(t *testing.T) {
	if got := TruncateWidth("short", 10); got != "short" {
		t.Errorf("TruncateWidth(short, 10) = %q, want %q", got, "short")
	}
	if got := TruncateWidth("abcdef", 4); got != "abc…" {
		t.Errorf("TruncateWidth(abcdef, 4) = %q, want %q", got, "abc…")
	}
	if got := TruncateWidth("一二三四", 4); got != "一…" {
		t.Errorf("TruncateWidth(cjk, 4) = %q, want %q", got, "一…")
	}
	if got := TruncateWidth("", 4); got != "" {
		t.Errorf("TruncateWidth(empty, 4) = %q, want empty", got)
	}
}

func TestSupportsKittyGraphics(t *testing.T) {
	t.Run("kitty window id", func(t *testing.T) {
		t.Setenv("KITTY_WINDOW_ID", "12345")
		t.Setenv("TERM_PROGRAM", "Apple_Terminal")
		if !supportsKittyGraphics() {
			t.Error("expected kitty support with KITTY_WINDOW_ID set")
		}
	})
	t.Run("kitty protocol terminal", func(t *testing.T) {
		t.Setenv("KITTY_WINDOW_ID", "")
		for _, prog := range []string{"kitty", "WezTerm", "ghostty", "Konsole", "alacritty"} {
			t.Setenv("TERM_PROGRAM", prog)
			if !supportsKittyGraphics() {
				t.Errorf("expected kitty support for TERM_PROGRAM=%q", prog)
			}
		}
	})
	t.Run("unsupported terminal", func(t *testing.T) {
		t.Setenv("KITTY_WINDOW_ID", "")
		t.Setenv("TERM_PROGRAM", "Apple_Terminal")
		if supportsKittyGraphics() {
			t.Error("expected no kitty support for Apple_Terminal")
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

package utils

import (
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

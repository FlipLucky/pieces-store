package tuibase

import (
	"testing"

	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/types"
	"github.com/fliplucky/pieces-store/internal/viewmanager"
)

func TestStyledLineTextNoSpansIsUnchanged(t *testing.T) {
	got := styledLineText("hello world", 0, nil, false, -1)
	if got != "hello world" {
		t.Errorf("styledLineText with no spans = %q, want unchanged %q", got, "hello world")
	}
}

func TestStyledLineTextCursorMatchesOriginalTagSequence(t *testing.T) {
	got := styledLineText("cat", 0, nil, true, 1)
	want := "c[#1a1b26:#7aa2f7]a[#a9b1d6:#1a1b26]t"
	if got != want {
		t.Errorf("styledLineText cursor = %q, want %q", got, want)
	}
}

func TestStyledLineTextCursorAtEndOfLine(t *testing.T) {
	got := styledLineText("cat", 0, nil, true, 3)
	want := "cat[#1a1b26:#7aa2f7] [#a9b1d6:#1a1b26]"
	if got != want {
		t.Errorf("styledLineText cursor-at-end = %q, want %q", got, want)
	}
}

func TestStyledLineTextAppliesSpanColor(t *testing.T) {
	spans := []viewmanager.StyledSpan{
		{TextRange: offset.TextRange{Start: 6, Length: 5}, Style: types.StyleString}, // "world"
	}
	got := styledLineText("hello world", 0, spans, false, -1)
	want := "hello [#9ece6a:#1a1b26]w[#a9b1d6:#1a1b26][#9ece6a:#1a1b26]o[#a9b1d6:#1a1b26][#9ece6a:#1a1b26]r[#a9b1d6:#1a1b26][#9ece6a:#1a1b26]l[#a9b1d6:#1a1b26][#9ece6a:#1a1b26]d[#a9b1d6:#1a1b26]"
	if got != want {
		t.Errorf("styledLineText with span = %q, want %q", got, want)
	}
}

func TestStyledLineTextCursorTakesPriorityOverSpan(t *testing.T) {
	spans := []viewmanager.StyledSpan{
		{TextRange: offset.TextRange{Start: 0, Length: 5}, Style: types.StyleKeyword}, // covers the whole word
	}
	// Cursor sits inside the styled span — the cursor cell must still get
	// cursor styling, not the span's color.
	got := styledLineText("hello", 0, spans, true, 2)
	want := "[#bb9af7:#1a1b26]h[#a9b1d6:#1a1b26][#bb9af7:#1a1b26]e[#a9b1d6:#1a1b26][#1a1b26:#7aa2f7]l[#a9b1d6:#1a1b26][#bb9af7:#1a1b26]l[#a9b1d6:#1a1b26][#bb9af7:#1a1b26]o[#a9b1d6:#1a1b26]"
	if got != want {
		t.Errorf("styledLineText cursor-over-span = %q, want %q", got, want)
	}
}

func TestStyledLineTextRespectsMultiByteRuneOffsets(t *testing.T) {
	// "héllo" — "h" (1 byte), "é" (2 bytes, U+00E9), then "llo". A span
	// starting at byte 3 (right after "h"+"é") must land on the "l", not
	// be thrown off by "é"'s 2-byte width.
	line := "héllo"
	spans := []viewmanager.StyledSpan{
		{TextRange: offset.TextRange{Start: 3, Length: 1}, Style: types.StyleNumber},
	}
	got := styledLineText(line, 0, spans, false, -1)
	want := "h" + "é" + "[#ff9e64:#1a1b26]l[#a9b1d6:#1a1b26]" + "lo"
	if got != want {
		t.Errorf("styledLineText multi-byte = %q, want %q", got, want)
	}
}

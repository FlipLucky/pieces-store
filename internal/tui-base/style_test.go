package tuibase

import (
	"testing"

	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/types"
	"github.com/fliplucky/pieces-store/internal/viewmanager"
)

func TestStyledLineTextNoSpansIsUnchanged(t *testing.T) {
	got := styledLineText("hello world", 0, nil, nil)
	if got != "hello world" {
		t.Errorf("styledLineText with no spans = %q, want unchanged %q", got, "hello world")
	}
}

func TestStyledLineTextAppliesSpanColor(t *testing.T) {
	spans := []viewmanager.StyledSpan{
		{TextRange: offset.TextRange{Start: 6, Length: 5}, Style: types.StyleString}, // "world"
	}
	got := styledLineText("hello world", 0, spans, nil)
	want := "hello [#9ece6a:#1a1b26:U]w[#a9b1d6:#1a1b26:U][#9ece6a:#1a1b26:U]o[#a9b1d6:#1a1b26:U][#9ece6a:#1a1b26:U]r[#a9b1d6:#1a1b26:U][#9ece6a:#1a1b26:U]l[#a9b1d6:#1a1b26:U][#9ece6a:#1a1b26:U]d[#a9b1d6:#1a1b26:U]"
	if got != want {
		t.Errorf("styledLineText with span = %q, want %q", got, want)
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
	got := styledLineText(line, 0, spans, nil)
	want := "h" + "é" + "[#ff9e64:#1a1b26:U]l[#a9b1d6:#1a1b26:U]" + "lo"
	if got != want {
		t.Errorf("styledLineText multi-byte = %q, want %q", got, want)
	}
}

// TestStyledLineTextDiagnosticOverridesColorToSeverityAndUnderlines
// documents a real, checked tview API constraint (see tviewStyleColor's
// doc comment): its tag syntax ties underline to the same foreground
// color as the text, with no way to underline in red while keeping a
// token's own syntax color. Given that, a diagnosed keyword's color
// becomes the diagnostic's severity color (matching how most terminal
// tools show diagnostics) rather than staying keyword-purple with an
// invisibly-same-colored underline.
func TestStyledLineTextDiagnosticOverridesColorToSeverityAndUnderlines(t *testing.T) {
	spans := []viewmanager.StyledSpan{
		{TextRange: offset.TextRange{Start: 0, Length: 5}, Style: types.StyleKeyword},
	}
	diagnostics := []viewmanager.DiagnosticSpan{
		{TextRange: offset.TextRange{Start: 0, Length: 5}, Style: types.StyleDiagnosticError},
	}
	got := styledLineText("hello", 0, spans, diagnostics)
	want := "[#f7768e:#1a1b26:u]h[#a9b1d6:#1a1b26:U][#f7768e:#1a1b26:u]e[#a9b1d6:#1a1b26:U][#f7768e:#1a1b26:u]l[#a9b1d6:#1a1b26:U][#f7768e:#1a1b26:u]l[#a9b1d6:#1a1b26:U][#f7768e:#1a1b26:u]o[#a9b1d6:#1a1b26:U]"
	if got != want {
		t.Errorf("styledLineText diagnosed keyword = %q, want %q (severity color + underline, real color visible)", got, want)
	}
}

func TestStyledLineTextDiagnosticAloneUsesSeverityColorUnderlined(t *testing.T) {
	diagnostics := []viewmanager.DiagnosticSpan{
		{TextRange: offset.TextRange{Start: 0, Length: 2}, Style: types.StyleDiagnosticWarning},
	}
	got := styledLineText("hi", 0, nil, diagnostics)
	want := "[#e0af68:#1a1b26:u]h[#a9b1d6:#1a1b26:U][#e0af68:#1a1b26:u]i[#a9b1d6:#1a1b26:U]"
	if got != want {
		t.Errorf("styledLineText plain-text diagnostic = %q, want %q", got, want)
	}
}

func TestStyledLineTextDiagnosticNoLongerFlowsThroughStyleColor(t *testing.T) {
	// tviewStyleColor itself must not have cases for the diagnostic Style
	// values any more — diagnostics render via the separate underline
	// path (styledLineText's diagnostics parameter), not by color.
	if _, ok := tviewStyleColor(types.StyleDiagnosticError); ok {
		t.Error("tviewStyleColor(StyleDiagnosticError) ok = true, want false — diagnostics render as underline, not color")
	}
	if _, ok := tviewStyleColor(types.StyleDiagnosticWarning); ok {
		t.Error("tviewStyleColor(StyleDiagnosticWarning) ok = true, want false — diagnostics render as underline, not color")
	}
}

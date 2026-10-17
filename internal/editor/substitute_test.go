package editor

import (
	"testing"

	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/types"
)

// TestSubstituteCurrentLineOnly covers :s///'s default scope (no %): only
// the line the cursor sits on is affected, matching real vim's default
// range of "." when none is given.
func TestSubstituteCurrentLineOnly(t *testing.T) {
	e := NewEditor("foo bar\nfoo baz")
	for i := 0; i < 8; i++ {
		e.MoveCursorRight() // onto the second line
	}
	if err := e.ExecuteCommand(":s/foo/qux/"); err != nil {
		t.Fatalf("ExecuteCommand(:s/foo/qux/) = %v, want nil", err)
	}
	if got := e.GetText(); got != "foo bar\nqux baz" {
		t.Errorf("GetText() = %q, want %q", got, "foo bar\nqux baz")
	}
}

// TestSubstituteWholeBufferWithPercent covers :%s///: every line in the
// document is affected, not just the current one.
func TestSubstituteWholeBufferWithPercent(t *testing.T) {
	e := NewEditor("foo bar\nfoo baz\nfoo qux")
	if err := e.ExecuteCommand(":%s/foo/XXX/"); err != nil {
		t.Fatalf("ExecuteCommand(:%%s/foo/XXX/) = %v, want nil", err)
	}
	if got := e.GetText(); got != "XXX bar\nXXX baz\nXXX qux" {
		t.Errorf("GetText() = %q, want %q", got, "XXX bar\nXXX baz\nXXX qux")
	}
}

// TestSubstituteWithoutGFlagReplacesFirstMatchPerLineOnly confirms the
// default (no g) behavior: only the first match on each in-scope line is
// replaced, matching real vim's own default.
func TestSubstituteWithoutGFlagReplacesFirstMatchPerLineOnly(t *testing.T) {
	e := NewEditor("foo foo foo")
	if err := e.ExecuteCommand(":s/foo/bar/"); err != nil {
		t.Fatalf("ExecuteCommand(:s/foo/bar/) = %v, want nil", err)
	}
	if got := e.GetText(); got != "bar foo foo" {
		t.Errorf("GetText() = %q, want %q", got, "bar foo foo")
	}
}

// TestSubstituteWithGFlagReplacesEveryMatch covers the "g" flag.
func TestSubstituteWithGFlagReplacesEveryMatch(t *testing.T) {
	e := NewEditor("foo foo foo")
	if err := e.ExecuteCommand(":s/foo/bar/g"); err != nil {
		t.Fatalf("ExecuteCommand(:s/foo/bar/g) = %v, want nil", err)
	}
	if got := e.GetText(); got != "bar bar bar" {
		t.Errorf("GetText() = %q, want %q", got, "bar bar bar")
	}
}

// TestSubstituteWholeBufferWithGFlag covers %s/// + g together: every
// match on every line.
func TestSubstituteWholeBufferWithGFlag(t *testing.T) {
	e := NewEditor("foo foo\nfoo foo")
	if err := e.ExecuteCommand(":%s/foo/bar/g"); err != nil {
		t.Fatalf("ExecuteCommand(:%%s/foo/bar/g) = %v, want nil", err)
	}
	if got := e.GetText(); got != "bar bar\nbar bar" {
		t.Errorf("GetText() = %q, want %q", got, "bar bar\nbar bar")
	}
}

// TestSubstitutePatternNotFound covers the "matched nothing" case,
// distinct from a malformed command — real vim's own E486.
func TestSubstitutePatternNotFound(t *testing.T) {
	e := NewEditor("foo bar")
	if err := e.ExecuteCommand(":s/zzz/qux/"); err != ErrPatternNotFound {
		t.Errorf("ExecuteCommand(:s/zzz/qux/) = %v, want %v", err, ErrPatternNotFound)
	}
	if got := e.GetText(); got != "foo bar" {
		t.Errorf("GetText() after a failed substitute = %q, want unchanged %q", got, "foo bar")
	}
}

// TestSubstituteMalformedCommand covers a command that isn't even
// delimiter-shaped at all.
func TestSubstituteMalformedCommand(t *testing.T) {
	e := NewEditor("foo bar")
	if err := e.ExecuteCommand(":s"); err != ErrInvalidSubstitute {
		t.Errorf("ExecuteCommand(:s) = %v, want %v", err, ErrInvalidSubstitute)
	}
}

// TestSubstitutePreviewRangesCurrentLineOnly covers the live-preview
// highlight (nvim's "inccommand") while a :s command is still being
// typed, not yet confirmed — same current-line-only default scope as
// execution, and the buffer itself must stay untouched.
func TestSubstitutePreviewRangesCurrentLineOnly(t *testing.T) {
	e := NewEditor("foo bar\nfoo baz")
	for i := 0; i < 8; i++ {
		e.MoveCursorRight() // onto the second line
	}
	e.cursor.Mode = types.ModeCommand
	e.cursor.CommandBuffer = "s/foo/qux/"

	got := e.SubstitutePreviewRanges()
	want := []offset.TextRange{{Start: 8, Length: 3}} // "foo" at the start of line 2
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("SubstitutePreviewRanges() = %v, want %v", got, want)
	}
	if text := e.GetText(); text != "foo bar\nfoo baz" {
		t.Errorf("preview must not mutate the buffer, got %q", text)
	}
}

// TestSubstitutePreviewRangesWholeBufferWithPercent covers the "%" prefix
// widening the preview to every line, matching :%s///'s own scope.
func TestSubstitutePreviewRangesWholeBufferWithPercent(t *testing.T) {
	e := NewEditor("foo bar\nfoo baz")
	e.cursor.Mode = types.ModeCommand
	e.cursor.CommandBuffer = "%s/foo/qux/"

	got := e.SubstitutePreviewRanges()
	if len(got) != 2 {
		t.Fatalf("SubstitutePreviewRanges() = %v, want 2 matches (one per line)", got)
	}
}

// TestSubstitutePreviewRangesRespectsGFlag covers every match on the
// in-scope line(s) being highlighted once "g" is typed, not just the
// first.
func TestSubstitutePreviewRangesRespectsGFlag(t *testing.T) {
	e := NewEditor("foo foo foo")
	e.cursor.Mode = types.ModeCommand
	e.cursor.CommandBuffer = "s/foo/bar/g"

	if got := e.SubstitutePreviewRanges(); len(got) != 3 {
		t.Errorf("SubstitutePreviewRanges() with g = %v, want 3 matches", got)
	}
}

// TestSubstitutePreviewRangesEmptyOutsideCommandMode covers the preview
// never firing when nothing is actually being typed as a command — it
// would be nonsensical to highlight leftover command-buffer content once
// the user has left Command mode.
func TestSubstitutePreviewRangesEmptyOutsideCommandMode(t *testing.T) {
	e := NewEditor("foo bar")
	e.cursor.Mode = types.ModeNormal
	e.cursor.CommandBuffer = "s/foo/qux/"

	if got := e.SubstitutePreviewRanges(); got != nil {
		t.Errorf("SubstitutePreviewRanges() outside Command mode = %v, want nil", got)
	}
}

// TestSubstitutePreviewRangesEmptyWhileStillTyping covers the common,
// expected-to-be-silent case: the pattern's closing delimiter hasn't been
// typed yet, so there's nothing valid to preview — must not error or
// panic on every keystroke, same as real nvim's own live preview.
func TestSubstitutePreviewRangesEmptyWhileStillTyping(t *testing.T) {
	e := NewEditor("foo bar")
	e.cursor.Mode = types.ModeCommand

	for _, partial := range []string{"s", "s/", "s/fo", "s/foo"} {
		e.cursor.CommandBuffer = CommandBuffer(partial)
		if got := e.SubstitutePreviewRanges(); got != nil {
			t.Errorf("SubstitutePreviewRanges() with buffer %q = %v, want nil", partial, got)
		}
	}
}

// TestSubstitutePreviewRangesEmptyForInvalidRegex covers an unparseable
// pattern (RE2 can't compile it) staying silent rather than erroring —
// same reasoning as the still-typing case above.
func TestSubstitutePreviewRangesEmptyForInvalidRegex(t *testing.T) {
	e := NewEditor("foo bar")
	e.cursor.Mode = types.ModeCommand
	e.cursor.CommandBuffer = "s/[foo/qux/" // unterminated character class

	if got := e.SubstitutePreviewRanges(); got != nil {
		t.Errorf("SubstitutePreviewRanges() with invalid regex = %v, want nil", got)
	}
}

// TestSubstituteInvalidRegex covers a pattern Go's regexp can't compile.
func TestSubstituteInvalidRegex(t *testing.T) {
	e := NewEditor("foo bar")
	if err := e.ExecuteCommand(":s/[/bar/"); err == nil {
		t.Errorf("ExecuteCommand with an unclosed character class = nil error, want one")
	}
}

// TestSubstituteSupportsCaptureGroups confirms Go's regexp $1-style
// capture-group substitution works transparently in the replacement —
// a real capability, not vim's own \1 syntax, documented as a deliberate
// divergence (see substitute.go's package doc comment).
func TestSubstituteSupportsCaptureGroups(t *testing.T) {
	e := NewEditor("hello world")
	if err := e.ExecuteCommand(`:s/(\w+) (\w+)/$2 $1/`); err != nil {
		t.Fatalf("ExecuteCommand = %v, want nil", err)
	}
	if got := e.GetText(); got != "world hello" {
		t.Errorf("GetText() = %q, want %q", got, "world hello")
	}
}

// TestSubstituteWithCustomDelimiter covers choosing a delimiter other
// than "/" — useful when the pattern itself contains slashes (e.g. a
// file path) without needing to escape them.
func TestSubstituteWithCustomDelimiter(t *testing.T) {
	e := NewEditor("/usr/bin")
	if err := e.ExecuteCommand(":s#/usr/bin#/opt/bin#"); err != nil {
		t.Fatalf("ExecuteCommand = %v, want nil", err)
	}
	if got := e.GetText(); got != "/opt/bin" {
		t.Errorf("GetText() = %q, want %q", got, "/opt/bin")
	}
}

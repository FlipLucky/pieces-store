package editor

import "testing"

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

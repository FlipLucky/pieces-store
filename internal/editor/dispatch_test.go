package editor

import (
	"testing"

	"github.com/fliplucky/pieces-store/internal/types"
)

// type sends each key in turn through the real HandleKey path.
func typeKeys(e *Editor, keys ...string) {
	for _, k := range keys {
		e.HandleKey(k)
	}
}

func TestInsertModeRoundTrip(t *testing.T) {
	e := NewEditor("")
	typeKeys(e, "i", "a", "b", "c", "<Esc>")

	if got := e.GetText(); got != "abc" {
		t.Errorf("GetText() = %q, want %q", got, "abc")
	}
	if mode := e.GetModeInt(); mode != int(types.ModeNormal) {
		t.Errorf("mode after <Esc> = %d, want Normal", mode)
	}
}

func TestCommandModeEscCancelsWithoutExecuting(t *testing.T) {
	e := NewEditor("hello")
	typeKeys(e, ":", "q", "<Esc>")

	if e.IsQuitRequested() {
		t.Errorf("Esc from command mode should cancel, not execute :q")
	}
	if buf := e.GetCommandBuffer(); buf != "" {
		t.Errorf("command buffer after Esc = %q, want empty", buf)
	}
}

func TestDIW(t *testing.T) {
	e := NewEditor("hello world")
	e.MoveCursorRight() // offset 1, inside "hello"
	typeKeys(e, "d", "i", "w")

	if got := e.GetText(); got != " world" {
		t.Errorf("GetText() after diw = %q, want %q", got, " world")
	}
	if off := e.GetCursor().ByteOffset; off != 0 {
		t.Errorf("cursor offset after diw = %d, want 0", off)
	}
}

func TestDAW(t *testing.T) {
	e := NewEditor("hello world")
	e.MoveCursorRight() // offset 1, inside "hello"
	typeKeys(e, "d", "a", "w")

	if got := e.GetText(); got != "world" {
		t.Errorf("GetText() after daw = %q, want %q", got, "world")
	}
}

func TestDD(t *testing.T) {
	e := NewEditor("line1\nline2\nline3")
	// Move cursor onto line2 (offset 6).
	for i := 0; i < 6; i++ {
		e.MoveCursorRight()
	}
	typeKeys(e, "d", "d")

	if got := e.GetText(); got != "line1\nline3" {
		t.Errorf("GetText() after dd = %q, want %q", got, "line1\nline3")
	}
}

func TestDIP(t *testing.T) {
	e := NewEditor("para one\nmore one\n\npara two\n")
	// Cursor stays at offset 0, inside the first paragraph.
	typeKeys(e, "d", "i", "p")

	if got := e.GetText(); got != "\npara two\n" {
		t.Errorf("GetText() after dip = %q, want %q", got, "\npara two\n")
	}
}

// TestCountedDeleteInnerWord covers T17 (3diw): resolveRange chains
// resolveRangeOnce count times, extending the range by one more "inner
// word" application each time. Matches real vim's own documented counted
// -iw behavior: a whitespace run between words counts as its own unit in
// the chain, same as a word does — so 3diw from "foo" consumes
// "foo" + " " + "bar", not three whole words.
func TestCountedDeleteInnerWord(t *testing.T) {
	e := NewEditor("foo bar baz qux")
	typeKeys(e, "3", "d", "i", "w")

	if got := e.GetText(); got != " baz qux" {
		t.Errorf("GetText() after 3diw = %q, want %q", got, " baz qux")
	}
}

// TestCountedDD covers the same chaining for the LineMotion case: 2dd
// deletes two consecutive whole lines.
func TestCountedDD(t *testing.T) {
	e := NewEditor("line1\nline2\nline3")
	typeKeys(e, "2", "d", "d")

	if got := e.GetText(); got != "line3" {
		t.Errorf("GetText() after 2dd = %q, want %q", got, "line3")
	}
}

// TestCountedDeleteStopsAtDocumentEnd covers the "stop at the edge, keep
// progress" half of the contract: a count larger than what the document
// has left still deletes whatever could be chained, rather than failing
// the whole operation.
func TestCountedDeleteStopsAtDocumentEnd(t *testing.T) {
	e := NewEditor("line1\nline2")
	typeKeys(e, "9", "d", "d")

	if got := e.GetText(); got != "" {
		t.Errorf("GetText() after 9dd on a 2-line buffer = %q, want %q", got, "")
	}
}

// TestFindCharForward covers T18b: f{char} moves the cursor to the next
// occurrence of char strictly after the cursor, on this specific keystroke
// (not a general search) — a pure motion with no buffer mutation.
func TestFindCharForward(t *testing.T) {
	e := NewEditor("foo bar baz")
	typeKeys(e, "f", "b")

	if off := e.GetCursor().ByteOffset; off != 4 {
		t.Errorf("cursor offset after fb = %d, want 4 (start of \"bar\")", off)
	}
	if got := e.GetText(); got != "foo bar baz" {
		t.Errorf("GetText() after fb = %q, want unchanged %q", got, "foo bar baz")
	}
}

// TestFindCharBackward covers F{char}: same as f, searching backward.
func TestFindCharBackward(t *testing.T) {
	e := NewEditor("foo bar baz")
	for i := 0; i < 10; i++ {
		e.MoveCursorRight() // onto "baz"
	}
	typeKeys(e, "F", "o")

	if off := e.GetCursor().ByteOffset; off != 2 {
		t.Errorf("cursor offset after Fo = %d, want 2 (the 'o' in \"foo\")", off)
	}
}

// TestFindCharEscCancelsPending confirms <Esc> after f/F cancels the
// pending state without moving the cursor, same as r's own <Esc> contract.
func TestFindCharEscCancelsPending(t *testing.T) {
	e := NewEditor("foo bar baz")
	typeKeys(e, "f", "<Esc>")

	if off := e.GetCursor().ByteOffset; off != 0 {
		t.Errorf("cursor offset after f<Esc> = %d, want 0 (unchanged)", off)
	}

	// Confirm normal parsing resumes afterward — not left stuck pending.
	typeKeys(e, "w")
	if off := e.GetCursor().ByteOffset; off != 4 {
		t.Errorf("cursor offset after f<Esc> w = %d, want 4 (w still works)", off)
	}
}

func TestWordMotions(t *testing.T) {
	e := NewEditor("foo bar baz")

	typeKeys(e, "w")
	if off := e.GetCursor().ByteOffset; off != 4 {
		t.Fatalf("after w: offset = %d, want 4 (start of \"bar\")", off)
	}

	typeKeys(e, "e")
	if off := e.GetCursor().ByteOffset; off != 6 {
		t.Fatalf("after e: offset = %d, want 6 (end of \"bar\")", off)
	}

	typeKeys(e, "b")
	if off := e.GetCursor().ByteOffset; off != 4 {
		t.Fatalf("after b: offset = %d, want 4 (start of \"bar\")", off)
	}
}

func TestUndoRedo(t *testing.T) {
	e := NewEditor("hello")
	typeKeys(e, "i", "X", "<Esc>")
	if got := e.GetText(); got != "Xhello" {
		t.Fatalf("setup failed: %q", got)
	}

	typeKeys(e, "u")
	if got := e.GetText(); got != "hello" {
		t.Errorf("GetText() after u = %q, want %q", got, "hello")
	}

	typeKeys(e, "<C-r>")
	if got := e.GetText(); got != "Xhello" {
		t.Errorf("GetText() after <C-r> = %q, want %q", got, "Xhello")
	}
}

func TestReplace(t *testing.T) {
	e := NewEditor("cat")
	typeKeys(e, "r", "b")

	if got := e.GetText(); got != "bat" {
		t.Errorf("GetText() after rb = %q, want %q", got, "bat")
	}
	if off := e.GetCursor().ByteOffset; off != 0 {
		t.Errorf("cursor offset after replace = %d, want 0 (stays in place)", off)
	}
}

func TestReplaceEscCancels(t *testing.T) {
	e := NewEditor("cat")
	typeKeys(e, "r", "<Esc>")

	if got := e.GetText(); got != "cat" {
		t.Errorf("GetText() after r<Esc> = %q, want unchanged %q", got, "cat")
	}
}

func TestOpenBelow(t *testing.T) {
	e := NewEditor("line1\nline2")
	typeKeys(e, "o", "x", "<Esc>")

	if got := e.GetText(); got != "line1\nx\nline2" {
		t.Errorf("GetText() after o = %q, want %q", got, "line1\nx\nline2")
	}
}

func TestOpenAbove(t *testing.T) {
	e := NewEditor("line1\nline2")
	// Move onto line2 first.
	for i := 0; i < 6; i++ {
		e.MoveCursorRight()
	}
	typeKeys(e, "O", "x", "<Esc>")

	if got := e.GetText(); got != "line1\nx\nline2" {
		t.Errorf("GetText() after O = %q, want %q", got, "line1\nx\nline2")
	}
}

func TestInsertLiteralText(t *testing.T) {
	e := NewEditor("hello")
	for i := 0; i < 5; i++ {
		e.MoveCursorRight() // to the end of "hello"
	}
	typeKeys(e, "i")
	e.InsertLiteralText(" pasted")
	typeKeys(e, "<Esc>")

	if got := e.GetText(); got != "hello pasted" {
		t.Errorf("GetText() after paste in Insert mode = %q, want %q", got, "hello pasted")
	}
}

func TestInsertLiteralTextDroppedInNormalMode(t *testing.T) {
	e := NewEditor("hello")
	e.InsertLiteralText("pasted") // still in Normal mode — must be a no-op

	if got := e.GetText(); got != "hello" {
		t.Errorf("GetText() after paste in Normal mode = %q, want unchanged %q", got, "hello")
	}
}

func TestMoveCursorDownStopsOnLastLine(t *testing.T) {
	e := NewEditor("line1\nline2")
	e.MoveCursorDown() // onto line2, offset 6
	if off := e.GetCursor().ByteOffset; off != 6 {
		t.Fatalf("after first MoveCursorDown: offset = %d, want 6", off)
	}

	e.MoveCursorDown() // already on the last line — must stay put
	if off := e.GetCursor().ByteOffset; off != 6 {
		t.Errorf("MoveCursorDown on the last line moved to %d, want to stay at 6 (not jump to end of buffer)", off)
	}
}

func TestCountedMotion(t *testing.T) {
	e := NewEditor("aa bb cc dd")
	typeKeys(e, "2", "w")

	if off := e.GetCursor().ByteOffset; off != 6 {
		t.Errorf("after 2w: offset = %d, want 6 (start of \"cc\")", off)
	}
}

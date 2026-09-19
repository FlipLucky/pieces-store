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

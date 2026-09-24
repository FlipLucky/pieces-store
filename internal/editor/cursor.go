package editor

import (
	"github.com/fliplucky/pieces-store/internal/lspclient"
	"github.com/fliplucky/pieces-store/internal/types"
)

// CompletionState is an in-progress :e/:w path-completion cycle — which
// candidates matched the token being completed, and which one is
// currently selected into the command buffer. Zero value means "no
// completion in progress," which is also what frontends check to decide
// whether to render a candidate popup at all.
type CompletionState struct {
	Active     bool
	Candidates []string
	Index      int
}

// HoverState is an in-progress LSP hover popup (K in Normal mode). Zero
// value means "nothing to show" — the same convention CompletionState
// uses. Text is already normalized to plain displayable text
// (lspclient.HoverResult.Text) — frontends don't need to know about
// hover's underlying markup shape.
type HoverState struct {
	Active bool
	Text   string
}

// LSPCompletionState is an in-progress LSP autocomplete popup — a separate
// type from CompletionState (not a reuse) because a real LSP completion
// item needs more than a display string: label, detail, and either plain
// insert text or a structured TextEdit (a replace-this-range instruction,
// which real servers use far more often than plain InsertText — confirmed
// against gopls, see lspclient's own doc comments). Reusing
// lspclient.CompletionItem directly rather than redefining an equivalent
// type here.
type LSPCompletionState struct {
	Active bool
	Items  []lspclient.CompletionItem
	Index  int
}

// Cursor is editor state — position, mode, the in-progress command
// buffer, and any in-progress command-line/LSP completion or hover popup —
// not view state. Frontends read it for rendering, but it lives here
// because Editor owns it end-to-end.
type Cursor struct {
	ByteOffset    int
	Row           int
	Col           int
	Mode          types.Mode
	CommandBuffer string
	Completion    CompletionState
	LSPCompletion LSPCompletionState
	Hover         HoverState
}

func NewCursor() *Cursor {
	return &Cursor{
		ByteOffset: 0,
		Row:        0,
		Col:        0,
		Mode:       types.ModeNormal,
	}
}

func (c *Cursor) Update(byteOffset, row, col int) {
	c.ByteOffset = byteOffset
	c.Row = row
	c.Col = col
}

func (c *Cursor) SetMode(m types.Mode) {
	c.Mode = m
}

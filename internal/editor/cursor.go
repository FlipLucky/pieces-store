package editor

import "github.com/fliplucky/pieces-store/internal/types"

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

// Cursor is editor state — position, mode, the in-progress command
// buffer, and any in-progress command-line completion — not view state.
// Frontends read it for rendering, but it lives here because Editor owns
// it end-to-end.
type Cursor struct {
	ByteOffset    int
	Row           int
	Col           int
	Mode          types.Mode
	CommandBuffer string
	Completion    CompletionState
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

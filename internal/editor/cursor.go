package editor

type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeVisual
	ModeCommand
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeVisual:
		return "VISUAL"
	case ModeCommand:
		return "COMMAND"
	default:
		return "UNKNOWN"
	}
}

type Cursor struct {
	ByteOffset    int
	Row           int
	Col           int
	Mode          Mode
	CommandBuffer string
}

func NewCursor() *Cursor {
	return &Cursor{
		ByteOffset: 0,
		Row:        0,
		Col:        0,
		Mode:       ModeNormal,
	}
}

func (c *Cursor) Update(byteOffset, row, col int) {
	c.ByteOffset = byteOffset
	c.Row = row
	c.Col = col
}

func (c *Cursor) SetMode(m Mode) {
	c.Mode = m
}

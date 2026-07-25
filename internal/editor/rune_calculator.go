package editor

import (
	"unicode/utf8"
)

type RuneCalculator struct{}

func NewRuneCalculator() *RuneCalculator {
	return &RuneCalculator{}
}

type Position struct {
	Row int
	Col int
}

// ByteOffsetToPosition translates a byte offset into a (Row, Col) virtual grid position
func (rc *RuneCalculator) ByteOffsetToPosition(doc Document, targetOffset int) Position {
	row := 0
	col := 0
	byteOffset := 0

	for byteOffset < doc.Len() && byteOffset < targetOffset {
		r, size := doc.GetRuneAt(byteOffset)

		// Use the init block for the second GetRuneAt call
		if r == '\r' && byteOffset+size < doc.Len() {
			if nextR, _ := doc.GetRuneAt(byteOffset + size); nextR == '\n' {
				row++
				col = 0
				byteOffset += size + 1 // Skip both \r and \n
				continue
			}
		}

		if r == '\n' {
			row++
			col = 0
		} else {
			col++
		}
		byteOffset += size
	}

	return Position{Row: row, Col: col}
}

// PositionToByteOffset translates a (Row, Col) virtual grid position into a byte offset
// We now accept optional 'startOffset' and 'startRow' as our anchors.
func (rc *RuneCalculator) PositionToByteOffset(doc Document, pos Position, anchors ...int) int {
	startOffset := 0
	startRow := 0
	if len(anchors) >= 2 {
		startOffset = anchors[0]
		startRow = anchors[1]
	}
	row := startRow
	col := 0
	byteOffset := startOffset

	// Safety check: if the requested row is before our anchor, we must start from 0
	if pos.Row < startRow {
		byteOffset = 0
		row = 0
	}

	for byteOffset < doc.Len() {
		// If we've reached our target row, we are effectively looking for the column
		if row == pos.Row {
			if col >= pos.Col {
				return byteOffset
			}
		} else if row > pos.Row {
			// We passed the target row, return the start of this line (or end of file)
			return byteOffset
		}

		r, size := doc.GetRuneAt(byteOffset)

		// Handle CRLF
		if r == '\r' && byteOffset+size < doc.Len() {
			if nextR, _ := doc.GetRuneAt(byteOffset + size); nextR == '\n' {
				if row == pos.Row {
					return byteOffset
				}
				row++
				col = 0
				byteOffset += size + 1
				continue
			}
		}

		// Handle LF
		if r == '\n' {
			if row == pos.Row {
				return byteOffset
			}
			row++
			col = 0
		} else {
			col++
		}
		byteOffset += size
	}

	return byteOffset
}

// MoveLeft returns the new byte offset when moving left by one UTF-8 rune
func (rc *RuneCalculator) MoveLeft(doc Document, currentOffset int) int {
	if currentOffset <= 0 {
		return 0
	}
	for step := 1; step <= 4 && currentOffset-step >= 0; step++ {
		tryOffset := currentOffset - step
		r, size := doc.GetRuneAt(tryOffset)
		if r != utf8.RuneError && size > 0 && tryOffset+size == currentOffset {
			return tryOffset
		}
	}
	return currentOffset - 1
}

// MoveRight returns the new byte offset when moving right by one UTF-8 rune
func (rc *RuneCalculator) MoveRight(doc Document, currentOffset int) int {
	if currentOffset >= doc.Len() {
		return doc.Len()
	}
	_, size := doc.GetRuneAt(currentOffset)
	return currentOffset + size
}

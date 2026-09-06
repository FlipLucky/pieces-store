package viewmanager

import (
	"unicode"
	"unicode/utf8"
)

type Document interface {
	GetRuneAt(offset int) (rune, int)
	Len() int
}

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
	return stepRune(doc, currentOffset, Backward)
}

// MoveRight returns the new byte offset when moving right by one UTF-8 rune
func (rc *RuneCalculator) MoveRight(doc Document, currentOffset int) int {
	return stepRune(doc, currentOffset, Forward)
}

// -------------------------------------------- //
// -- This is the main magic for keybindings -- //
// -------------------------------------------- //

type Direction int

const (
	Forward  Direction = 1
	Backward Direction = -1
)

type Predicate func(r rune) bool

// ----------------------- //
// -- Predicate Methods -- //
// ----------------------- //

func IsWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func Not(p Predicate) Predicate {
	return func(r rune) bool {
		return !p(r)
	}
}

func IsWhitespace(r rune) bool {
	return unicode.IsSpace(r)
}

// ----------------------------------------------------------- //
// -- The Offset machine to dictate allmost all keybindings -- //
// ----------------------------------------------------------- //

// stepRune moves offset by exactly one rune in the given direction, staying
// on a valid rune boundary instead of a raw byte step. It clamps to the
// document's edges (0 or doc.Len()) rather than landing mid-rune.
func stepRune(doc Document, offset int, dir Direction) int {
	if dir == Forward {
		if offset >= doc.Len() {
			return doc.Len()
		}
		_, width := doc.GetRuneAt(offset)
		if width <= 0 {
			width = 1
		}
		next := offset + width
		if next > doc.Len() {
			return doc.Len()
		}
		return next
	}

	if offset <= 0 {
		return 0
	}
	for step := 1; step <= 4 && offset-step >= 0; step++ {
		tryOffset := offset - step
		r, size := doc.GetRuneAt(tryOffset)
		if r != utf8.RuneError && size > 0 && tryOffset+size == offset {
			return tryOffset
		}
	}
	return offset - 1
}

func FindOffset(doc Document, start int, dir Direction, match Predicate) int {
	curr := start

	for {
		next := stepRune(doc, curr, dir)

		if next == curr {
			// Pinned at a document edge with no match found.
			if dir == Forward {
				return doc.Len()
			}
			return 0
		}
		curr = next

		if dir == Forward && curr >= doc.Len() {
			return doc.Len()
		}

		r, width := doc.GetRuneAt(curr)

		// Should no longer happen once stepping stays rune-aligned, but
		// bail out safely rather than loop on genuinely corrupt data.
		if r == utf8.RuneError || width == 0 {
			return start
		}

		if match(r) {
			return curr
		}
	}
}
// The only tool your calculator actually needs:
func ScanUntil(doc Document, start int, dir Direction, condition Predicate) int {
    curr := start
    for {
        curr += int(dir)
        if curr < 0 || curr >= doc.Len() {
            return curr // Hit edge of document
        }
        r, _ := doc.GetRuneAt(curr)
        if condition(r) {
            return curr
        }
    }
}

// w: Move forward to the beginning of the next word
func MotionWordForward(doc Document, currentOffset int) int {
	// Step 1: Scan forward until we exit the current word (find a non-word char)
	nextNonWord := FindOffset(doc, currentOffset, Forward, Not(IsWordChar))

	// Step 2: From that non-word char, scan forward until we find the start of the next word
	nextWordStart := FindOffset(doc, nextNonWord-1, Forward, IsWordChar)

	return nextWordStart
}

// b: Move backward to the beginning of the current or previous word
func MotionWordBackward(doc Document, currentOffset int) int {
	// If we are currently sitting on a non-word char, find the previous word character first
	r, _ := doc.GetRuneAt(currentOffset)
	if !IsWordChar(r) {
		currentOffset = FindOffset(doc, currentOffset, Backward, IsWordChar)
	}

	// Scan backward until we hit the transition boundary out of the word
	boundary := FindOffset(doc, currentOffset, Backward, Not(IsWordChar))

	// The word starts exactly one rune to the right of the boundary.
	if boundary == 0 {
		return 0
	}
	return stepRune(doc, boundary, Forward)
}

type TextRange struct {
	Start  int
	Length int
}

// iw: Inner Word (Selects/Deletes ONLY the characters of the word itself)
func RangeInnerWord(doc Document, currentOffset int) TextRange {
	r, _ := doc.GetRuneAt(currentOffset)
	// Edge case: If cursor is on a space, inner word targets the whitespace cluster instead
	if IsWhitespace(r) {
		start := stepRune(doc, FindOffset(doc, currentOffset, Backward, Not(IsWhitespace)), Forward)
		end := FindOffset(doc, currentOffset, Forward, Not(IsWhitespace))
		return TextRange{Start: start, Length: end - start}
	}

	// Standard Word scan boundaries
	wordStart := stepRune(doc, FindOffset(doc, currentOffset, Backward, Not(IsWordChar)), Forward)
	wordEnd := FindOffset(doc, currentOffset, Forward, Not(IsWordChar))

	return TextRange{Start: wordStart, Length: wordEnd - wordStart}
}

// aw: Around Word (Selects/Deletes the word PLUS its trailing whitespace)
func RangeAroundWord(doc Document, currentOffset int) TextRange {
	inner := RangeInnerWord(doc, currentOffset)

	// Scan forward from the end of the inner word to consume trailing whitespace
	trailingSpaceEnd := FindOffset(doc, inner.Start+inner.Length-1, Forward, Not(IsWhitespace))

	return TextRange{
		Start:  inner.Start,
		Length: trailingSpaceEnd - inner.Start,
	}
}

// fX: Find character X forward
func MotionFindCharForward(doc Document, currentOffset int, target string) int {
	if len(target) == 0 {
		return currentOffset
	}
	targetRune := rune(target[0])

	// Predicate function generated dynamically at runtime for this specific keystroke!
	matchTarget := func(r rune) bool {
		return r == targetRune
	}

	return FindOffset(doc, currentOffset, Forward, matchTarget)
}

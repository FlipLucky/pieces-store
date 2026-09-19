package offset

import (
	"unicode"
	"unicode/utf8"
)

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

// ScanUntil is dead code — nothing calls it, it has the same byte-stepping
// issue FindOffset used to have, and it looks like an abandoned attempt to
// simplify/replace FindOffset that never got finished. Left as-is pending
// a decision to delete it or complete the replacement.
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

	return spanStart(doc, currentOffset, IsWordChar)
}

// e: Move forward to the end of the current or next word.
func MotionWordEnd(doc Document, currentOffset int) int {
	pos := stepRune(doc, currentOffset, Forward)

	// Skip forward over any non-word characters to reach the start of a word.
	if r, _ := doc.GetRuneAt(pos); pos < doc.Len() && !IsWordChar(r) {
		pos = FindOffset(doc, pos, Forward, IsWordChar)
	}

	if pos >= doc.Len() {
		return doc.Len()
	}

	// pos now sits on a word character; scan to the first non-word rune
	// after it, then step back one rune to land on the word's last char.
	afterWord := FindOffset(doc, pos, Forward, Not(IsWordChar))
	return stepRune(doc, afterWord, Backward)
}

type TextRange struct {
	Start  int
	Length int
}

// spanStart scans backward from currentOffset for the start of the
// word/whitespace span that inSpan defines, correctly distinguishing "a
// real boundary character sits at offset 0" from "hit the start of the
// document with no boundary at all" — the latter must leave the span
// starting exactly at 0, not skip past it as though offset 0 itself were
// the boundary to exclude.
func spanStart(doc Document, currentOffset int, inSpan Predicate) int {
	boundary := FindOffset(doc, currentOffset, Backward, Not(inSpan))
	if boundary == 0 {
		if r, _ := doc.GetRuneAt(0); !inSpan(r) {
			return stepRune(doc, 0, Forward)
		}
		return 0
	}
	return stepRune(doc, boundary, Forward)
}

// iw: Inner Word (Selects/Deletes ONLY the characters of the word itself)
func RangeInnerWord(doc Document, currentOffset int) TextRange {
	r, _ := doc.GetRuneAt(currentOffset)
	// Edge case: If cursor is on a space, inner word targets the whitespace cluster instead
	if IsWhitespace(r) {
		start := spanStart(doc, currentOffset, IsWhitespace)
		end := FindOffset(doc, currentOffset, Forward, Not(IsWhitespace))
		return TextRange{Start: start, Length: end - start}
	}

	// Standard Word scan boundaries
	wordStart := spanStart(doc, currentOffset, IsWordChar)
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

// isNewline is the boundary predicate lineStart/lineEnd scan for.
func isNewline(r rune) bool { return r == '\n' }

// lineStart returns the offset of the first rune of the line containing offset.
func lineStart(doc Document, offset int) int {
	boundary := FindOffset(doc, offset, Backward, isNewline)
	if boundary == 0 {
		if r, _ := doc.GetRuneAt(0); r == '\n' {
			// A genuine newline sits at offset 0 — the line starts right after it.
			return stepRune(doc, 0, Forward)
		}
		return 0
	}
	return stepRune(doc, boundary, Forward)
}

// lineEnd returns the offset just past the last rune of the line containing
// offset (i.e. the position of the terminating '\n', or doc.Len()). A blank
// line's own newline sits exactly at its start, so it must be checked
// directly — FindOffset never tests its own starting position, only what
// comes after it, which would otherwise skip straight past a zero-length
// line to the next one.
func lineEnd(doc Document, offset int) int {
	if offset < doc.Len() {
		if r, _ := doc.GetRuneAt(offset); r == '\n' {
			return offset
		}
	}
	return FindOffset(doc, offset, Forward, isNewline)
}

// prevLineSpan returns the [start, end) span of the line immediately before
// the line starting at thisLineStart, or ok=false if there is none.
func prevLineSpan(doc Document, thisLineStart int) (start, end int, ok bool) {
	if thisLineStart <= 0 {
		return 0, 0, false
	}
	newlinePos := stepRune(doc, thisLineStart, Backward) // the '\n' ending the previous line
	end = newlinePos
	start = lineStart(doc, newlinePos)
	return start, end, true
}

// nextLineSpan returns the [start, end) span of the line immediately after
// the line ending at thisLineEnd, or ok=false if there is none.
func nextLineSpan(doc Document, thisLineEnd int) (start, end int, ok bool) {
	if thisLineEnd >= doc.Len() {
		return 0, 0, false
	}
	start = stepRune(doc, thisLineEnd, Forward) // skip the '\n'
	if start >= doc.Len() {
		return start, start, true // an empty trailing line
	}
	end = lineEnd(doc, start)
	return start, end, true
}

func isBlankSpan(doc Document, start, end int) bool {
	for o := start; o < end; {
		r, w := doc.GetRuneAt(o)
		if !IsWhitespace(r) {
			return false
		}
		if w <= 0 {
			w = 1
		}
		o += w
	}
	return true
}

// RangeLine returns the whole line containing currentOffset, including its
// trailing newline if it has one (matching dd's whole-line-including-
// newline convention — also used by o/O to find where a new line should be
// opened below/above).
func RangeLine(doc Document, currentOffset int) TextRange {
	start := lineStart(doc, currentOffset)
	end := lineEnd(doc, currentOffset)
	if end < doc.Len() {
		end = stepRune(doc, end, Forward) // include the trailing newline
	}
	return TextRange{Start: start, Length: end - start}
}

// ip: Inner Paragraph — the contiguous run of non-blank lines around
// currentOffset, including its trailing newline (matching dd's
// whole-line-including-newline convention).
func RangeInnerParagraph(doc Document, currentOffset int) TextRange {
	start := lineStart(doc, currentOffset)
	end := lineEnd(doc, currentOffset)

	for {
		pStart, pEnd, ok := prevLineSpan(doc, start)
		if !ok || isBlankSpan(doc, pStart, pEnd) {
			break
		}
		start = pStart
	}

	for {
		nStart, nEnd, ok := nextLineSpan(doc, end)
		if !ok || isBlankSpan(doc, nStart, nEnd) {
			break
		}
		end = nEnd
	}

	if end < doc.Len() {
		end = stepRune(doc, end, Forward) // include the paragraph's trailing newline
	}

	return TextRange{Start: start, Length: end - start}
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

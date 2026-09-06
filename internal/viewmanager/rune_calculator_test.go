package viewmanager

import (
	"testing"
	"unicode/utf8"
)

type ByteDocument []byte

func (b ByteDocument) Len() int {
	return len(b)
}

func (b ByteDocument) GetRuneAt(offset int) (rune, int) {
	if offset >= len(b) || offset < 0 {
		return utf8.RuneError, 0
	}
	return utf8.DecodeRune(b[offset:])
}
func TestRuneCalculator(t *testing.T) {
	rc := NewRuneCalculator()
	text := ByteDocument("Hello\n世界\r\nGo")

	// Test ByteOffsetToPosition
	p0 := rc.ByteOffsetToPosition(text, 0) // H
	if p0.Row != 0 || p0.Col != 0 {
		t.Errorf("Expected row 0 col 0, got %d, %d", p0.Row, p0.Col)
	}

	p5 := rc.ByteOffsetToPosition(text, 5) // \n
	if p5.Row != 0 || p5.Col != 5 {
		t.Errorf("Expected row 0 col 5, got %d, %d", p5.Row, p5.Col)
	}

	p6 := rc.ByteOffsetToPosition(text, 6) // 世
	if p6.Row != 1 || p6.Col != 0 {
		t.Errorf("Expected row 1 col 0, got %d, %d", p6.Row, p6.Col)
	}

	p9 := rc.ByteOffsetToPosition(text, 9) // 界
	if p9.Row != 1 || p9.Col != 1 {
		t.Errorf("Expected row 1 col 1, got %d, %d", p9.Row, p9.Col)
	}

	p14 := rc.ByteOffsetToPosition(text, 14) // G
	if p14.Row != 2 || p14.Col != 0 {
		t.Errorf("Expected row 2 col 0, got %d, %d", p14.Row, p14.Col)
	}

	// Test PositionToByteOffset
	o0 := rc.PositionToByteOffset(text, Position{Row: 0, Col: 0})
	if o0 != 0 {
		t.Errorf("Expected offset 0, got %d", o0)
	}

	o6 := rc.PositionToByteOffset(text, Position{Row: 1, Col: 0})
	if o6 != 6 {
		t.Errorf("Expected offset 6, got %d", o6)
	}

	o9 := rc.PositionToByteOffset(text, Position{Row: 1, Col: 1})
	if o9 != 9 {
		t.Errorf("Expected offset 9, got %d", o9)
	}

	o14 := rc.PositionToByteOffset(text, Position{Row: 2, Col: 0})
	if o14 != 14 {
		t.Errorf("Expected offset 14, got %d", o14)
	}

	// Test MoveLeft and MoveRight
	o12 := rc.PositionToByteOffset(text, Position{Row: 1, Col: 2}) // End of line 1 (after 界)
	left := rc.MoveLeft(text, o12)
	if left != 9 { // Steps back by one rune (世 is 6-9, 界 is 9-12)
		t.Errorf("MoveLeft expected offset 9, got %d", left)
	}

	right := rc.MoveRight(text, 6)
	if right != 9 { // Steps forward by one rune from 世 to 界
		t.Errorf("MoveRight expected offset 9, got %d", right)
	}
}

// TestMotionsWithMultiByteUTF8 exercises the word/text-object motions against
// a multi-byte rune sitting inside a word. Before the byte-stepping fix in
// FindOffset, any motion whose scan crossed a multi-byte character would
// abort immediately and silently return the wrong offset.
func TestMotionsWithMultiByteUTF8(t *testing.T) {
	// byte layout: ' '(0) c(1) a(2) f(3) é(4-5) ' '(6) b(7) a(8) r(9) ' '(10) b(11) a(12) z(13)
	text := ByteDocument(" café bar baz")

	if got := MotionWordForward(text, 1); got != 7 {
		t.Errorf(`MotionWordForward(1) = %d, want 7 (start of "bar")`, got)
	}

	if got := MotionWordBackward(text, 9); got != 7 {
		t.Errorf(`MotionWordBackward(9) = %d, want 7 (start of "bar")`, got)
	}

	if got := MotionWordBackward(text, 12); got != 11 {
		t.Errorf(`MotionWordBackward(12) = %d, want 11 (start of "baz")`, got)
	}

	if rng := RangeInnerWord(text, 4); rng.Start != 1 || rng.Length != 5 {
		t.Errorf(`RangeInnerWord(4) = %+v, want {Start:1 Length:5} ("café")`, rng)
	}

	if rng := RangeAroundWord(text, 4); rng.Start != 1 || rng.Length != 6 {
		t.Errorf(`RangeAroundWord(4) = %+v, want {Start:1 Length:6} ("café ")`, rng)
	}

	if got := MotionFindCharForward(text, 1, "r"); got != 9 {
		t.Errorf(`MotionFindCharForward(1, "r") = %d, want 9`, got)
	}
}

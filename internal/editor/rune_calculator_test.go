package editor

import (
	"testing"
)

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

package piecetable

import (
	"testing"
)

func TestInsertAndGetText(t *testing.T) {
	table := NewPieceTable([]byte("Hello World"))

	// Insert in the middle
	table.Insert(5, []byte(" Amazing"))
	expected := "Hello Amazing World"
	if got := table.CombinePieces(); got != expected {
		t.Errorf("Insert in middle failed. Got %q, expected %q", got, expected)
	}

	// Insert at the end
	table.Insert(len(expected), []byte("!"))
	expected += "!"
	if got := table.CombinePieces(); got != expected {
		t.Errorf("Insert at end failed. Got %q, expected %q", got, expected)
	}

	// Insert at the beginning
	table.Insert(0, []byte("Hey, "))
	expected = "Hey, " + expected
	if got := table.CombinePieces(); got != expected {
		t.Errorf("Insert at beginning failed. Got %q, expected %q", got, expected)
	}
}

func TestDelete(t *testing.T) {
	table := NewPieceTable([]byte("Hello World"))
	table.Delete(2, 5) // Delete "llo W" -> "Heorld"
	expected := "Heorld"
	if got := table.CombinePieces(); got != expected {
		t.Errorf("Delete failed. Got %q, expected %q", got, expected)
	}
}

func TestUndo(t *testing.T) {
	table := NewPieceTable([]byte("Hello World"))
	table.Insert(5, []byte(" Amazing"))
	if got := table.CombinePieces(); got != "Hello Amazing World" {
		t.Fatalf("Insert failed: %q", got)
	}
	if !table.Undo() {
		t.Fatalf("Undo failed")
	}
	if got := table.CombinePieces(); got != "Hello World" {
		t.Fatalf("Undo text mismatch: %q", got)
	}
}

func TestCoalesce(t *testing.T) {
	table := NewPieceTable([]byte("Hello World"))
	// Typing 5 characters sequentially at the end
	table.Insert(11, []byte("!"))
	table.Insert(12, []byte("!"))
	table.Insert(13, []byte("!"))

	// Without coalescing, this would be 4 pieces.
	// With automatic coalescing, all 3 appended '!' pieces merge into 1 piece!
	if len(table.Pieces) != 2 {
		t.Errorf("Expected 2 coalesced pieces, got %d", len(table.Pieces))
	}

	if got := table.CombinePieces(); got != "Hello World!!!" {
		t.Errorf("Text mismatch after coalesce. Got %q", got)
	}
}

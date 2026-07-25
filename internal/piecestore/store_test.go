package piecestore

import (
	"testing"
)

func TestInsertAndGetText(t *testing.T) {
	store := NewPieceStore([]byte("Hello World"))

	// Insert in the middle
	store.Insert(5, []byte(" Amazing"))
	expected := "Hello Amazing World"
	if got := store.CombinePieces(); got != expected {
		t.Errorf("Insert in middle failed. Got %q, expected %q", got, expected)
	}

	// Insert at the end
	store.Insert(len(expected), []byte("!"))
	expected += "!"
	if got := store.CombinePieces(); got != expected {
		t.Errorf("Insert at end failed. Got %q, expected %q", got, expected)
	}

	// Insert at the beginning
	store.Insert(0, []byte("Hey, "))
	expected = "Hey, " + expected
	if got := store.CombinePieces(); got != expected {
		t.Errorf("Insert at beginning failed. Got %q, expected %q", got, expected)
	}
}

func TestDelete(t *testing.T) {
	store := NewPieceStore([]byte("Hello World"))
	store.Delete(2, 5) // Delete "llo W" -> "Heorld"
	expected := "Heorld"
	if got := store.CombinePieces(); got != expected {
		t.Errorf("Delete failed. Got %q, expected %q", got, expected)
	}
}

func TestUndo(t *testing.T) {
	store := NewPieceStore([]byte("Hello World"))
	store.Insert(5, []byte(" Amazing"))
	if got := store.CombinePieces(); got != "Hello Amazing World" {
		t.Fatalf("Insert failed: %q", got)
	}
	if !store.Undo() {
		t.Fatalf("Undo failed")
	}
	if got := store.CombinePieces(); got != "Hello World" {
		t.Fatalf("Undo text mismatch: %q", got)
	}
}

func TestCoalesce(t *testing.T) {
	store := NewPieceStore([]byte("Hello World"))
	// Typing 5 characters sequentially at the end
	store.Insert(11, []byte("!"))
	store.Insert(12, []byte("!"))
	store.Insert(13, []byte("!"))

	// Without coalescing, this would be 4 pieces.
	// With automatic coalescing, all 3 appended '!' pieces merge into 1 piece!
	if len(store.Pieces) != 2 {
		t.Errorf("Expected 2 coalesced pieces, got %d", len(store.Pieces))
	}

	if got := store.CombinePieces(); got != "Hello World!!!" {
		t.Errorf("Text mismatch after coalesce. Got %q", got)
	}
}

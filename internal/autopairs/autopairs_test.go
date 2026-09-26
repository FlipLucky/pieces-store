package autopairs

import "testing"

func TestDecideOpeningBracketInsertsPair(t *testing.T) {
	cases := []struct {
		typed, wantClosing rune
	}{
		{'{', '}'},
		{'(', ')'},
		{'[', ']'},
	}
	for _, c := range cases {
		action, closing := Decide(c.typed, 0, 0)
		if action != InsertPair || closing != c.wantClosing {
			t.Errorf("Decide(%q, 0, 0) = (%v, %q), want (InsertPair, %q)", c.typed, action, closing, c.wantClosing)
		}
	}
}

func TestDecideClosingBracketAtMatchingCloserSkipsOver(t *testing.T) {
	action, _ := Decide('}', 0, '}')
	if action != SkipOver {
		t.Errorf("Decide('}', after='}') = %v, want SkipOver", action)
	}
}

func TestDecideClosingBracketWithNoMatchInsertsPlain(t *testing.T) {
	action, _ := Decide('}', 0, 0)
	if action != InsertPlain {
		t.Errorf("Decide('}', after=none) = %v, want InsertPlain", action)
	}
}

func TestDecideQuoteAtWordBoundaryInsertsPair(t *testing.T) {
	action, closing := Decide('"', ' ', ' ')
	if action != InsertPair || closing != '"' {
		t.Errorf(`Decide('"', space, space) = (%v, %q), want (InsertPair, '"')`, action, closing)
	}
}

func TestDecideQuoteAfterWordCharacterInsertsPlain(t *testing.T) {
	// "don" + typing ' — must not pair, or "don't" becomes "don''t".
	action, _ := Decide('\'', 'n', 0)
	if action != InsertPlain {
		t.Errorf("Decide(quote after word char) = %v, want InsertPlain", action)
	}
}

func TestDecideQuoteBeforeWordCharacterInsertsPlain(t *testing.T) {
	action, _ := Decide('"', 0, 'x')
	if action != InsertPlain {
		t.Errorf("Decide(quote before word char) = %v, want InsertPlain", action)
	}
}

func TestDecideQuoteAtExistingMatchingQuoteSkipsOver(t *testing.T) {
	action, _ := Decide('"', 'x', '"')
	if action != SkipOver {
		t.Errorf(`Decide('"', after='"') = %v, want SkipOver`, action)
	}
}

func TestDecidePlainRuneInsertsPlain(t *testing.T) {
	action, _ := Decide('x', 0, 0)
	if action != InsertPlain {
		t.Errorf("Decide('x') = %v, want InsertPlain", action)
	}
}

func TestIsEmptyPairBracket(t *testing.T) {
	if !IsEmptyPair('{', '}') {
		t.Error("IsEmptyPair('{', '}') = false, want true")
	}
	if IsEmptyPair('{', 'x') {
		t.Error("IsEmptyPair('{', 'x') = true, want false")
	}
}

func TestIsEmptyPairQuote(t *testing.T) {
	if !IsEmptyPair('"', '"') {
		t.Error(`IsEmptyPair('"', '"') = false, want true`)
	}
	if IsEmptyPair('"', '\'') {
		t.Error(`IsEmptyPair('"', '\'') = true, want false (mismatched quote types)`)
	}
}

func TestIsEmptyPairNeitherBracketNorQuote(t *testing.T) {
	if IsEmptyPair('a', 'b') {
		t.Error("IsEmptyPair('a', 'b') = true, want false")
	}
}

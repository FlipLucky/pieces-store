package editor

import (
	"testing"

	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/types"
	"github.com/fliplucky/pieces-store/internal/viewmanager"
)

func TestStyleSpansIsEmptyWithNoProducer(t *testing.T) {
	e := NewEditor("hello world")
	if spans := e.StyleSpans(0, 11); len(spans) != 0 {
		t.Errorf("StyleSpans = %+v, want empty (nothing produces real spans yet)", spans)
	}
}

func TestStyleSpansFiltersToRequestedRange(t *testing.T) {
	e := NewEditor("hello world")
	e.styleSpans = []viewmanager.StyledSpan{
		{TextRange: offset.TextRange{Start: 0, Length: 5}, Style: types.StyleKeyword}, // "hello"
		{TextRange: offset.TextRange{Start: 6, Length: 5}, Style: types.StyleString},  // "world"
	}

	got := e.StyleSpans(6, 11)
	if len(got) != 1 || got[0].Style != types.StyleString {
		t.Errorf("StyleSpans(6,11) = %+v, want just the \"world\" span", got)
	}
}

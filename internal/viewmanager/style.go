package viewmanager

import (
	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/types"
)

// StyledSpan is a range of the document that should render with a
// particular Style — the same offset.TextRange vocabulary used for
// motions and delete ranges, reused here for rendering rather than
// invented fresh. Nothing produces real StyledSpans yet (no syntax
// analyzer exists) — this is the contract a future one (or an LSP
// diagnostics layer) will populate.
type StyledSpan struct {
	offset.TextRange
	Style types.Style
}

// StyleAt returns the Style covering byteOffset, or types.StyleNone if
// none of spans do. Spans are assumed non-overlapping; if a future
// producer's spans do overlap, the first match wins.
func StyleAt(spans []StyledSpan, byteOffset int) types.Style {
	for _, s := range spans {
		if byteOffset >= s.Start && byteOffset < s.Start+s.Length {
			return s.Style
		}
	}
	return types.StyleNone
}

// SpansForRange filters spans to those overlapping [start, end) — the
// same viewport-bounding idea used everywhere else in this codebase
// (ViewportSlice, piecetable.GetRange/Delete), so a renderer only ever
// looks at spans relevant to what it's actually drawing: once per frame
// for the visible window (Editor.StyleSpans), and once per line to place
// them against that line's runes.
func SpansForRange(spans []StyledSpan, start, end int) []StyledSpan {
	var result []StyledSpan
	for _, s := range spans {
		if s.Start < end && s.Start+s.Length > start {
			result = append(result, s)
		}
	}
	return result
}

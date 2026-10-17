// substitute.go implements :s/pattern/replacement/flags and its
// whole-buffer form :%s/pattern/replacement/flags — the staged first pass
// per ATOMIC_TASKS.md's T13-T15: Go's own regexp syntax (RE2), not vim's
// regex dialect (a real, deliberate divergence — no backreferences or
// lookaround — documented rather than silently accepted, see
// docs/packages/editor.md), and the "g" flag only; "c"/"i" and arbitrary
// N,M line ranges are deliberately deferred, matching this file's own
// scope exactly as originally planned.
package editor

import (
	"errors"
	"regexp"
	"strings"

	"github.com/fliplucky/pieces-store/internal/exmode"
	"github.com/fliplucky/pieces-store/internal/offset"
)

var (
	// ErrInvalidSubstitute means :s's argument wasn't even shaped like
	// delim-pattern-delim-replacement[-delim-flags] at all.
	ErrInvalidSubstitute = errors.New("invalid substitute command (expected /pattern/replacement/)")
	// ErrPatternNotFound mirrors real vim's E486 — the pattern compiled
	// fine but matched nothing in the scope searched.
	ErrPatternNotFound = errors.New("pattern not found")
)

// substituteEdit is one match's byte-offset replacement — computed
// directly from regexp match indices, not lspclient.TextEdit's
// Position-based shape (that type exists for LSP responses, which arrive
// as Positions from the server; forcing a byte offset through Position
// and immediately back would buy nothing here).
type substituteEdit struct {
	Start, End int
	NewText    string
}

// executeSubstituteLocked is :s/:substitute's execution — called from
// executeCommandLocked. Caller must already hold e.mu.
func (e *Editor) executeSubstituteLocked(cmd exmode.Command) error {
	sub, ok := exmode.ParseSubstitute(cmd.Raw)
	if !ok {
		return ErrInvalidSubstitute
	}
	re, err := regexp.Compile(sub.Pattern)
	if err != nil {
		return err
	}
	global := strings.Contains(sub.Flags, "g")

	fullText := e.table.CombinePieces()
	lines := strings.Split(fullText, "\n")
	starts := make([]int, len(lines))
	pos := 0
	for i, line := range lines {
		starts[i] = pos
		pos += len(line) + 1 // +1 for the "\n" every line but the last has
	}

	lineIndices := []int{offset.ByteOffsetToPosition(e.table, e.cursor.ByteOffset).Row}
	if cmd.WholeBuffer {
		lineIndices = make([]int, len(lines))
		for i := range lines {
			lineIndices[i] = i
		}
	}

	var edits []substituteEdit
	for _, idx := range lineIndices {
		if idx < 0 || idx >= len(lines) {
			continue
		}
		line := lines[idx]
		base := starts[idx]

		var locs [][]int
		if global {
			locs = re.FindAllStringIndex(line, -1)
		} else if loc := re.FindStringIndex(line); loc != nil {
			locs = [][]int{loc}
		}
		for _, loc := range locs {
			matchText := line[loc[0]:loc[1]]
			// re.ReplaceAllString, not a literal splice: Go's regexp
			// interprets $1/$name capture-group references in the
			// replacement automatically, applied to just this one match.
			newText := re.ReplaceAllString(matchText, sub.Replacement)
			edits = append(edits, substituteEdit{Start: base + loc[0], End: base + loc[1], NewText: newText})
		}
	}

	if len(edits) == 0 {
		return ErrPatternNotFound
	}
	e.applySubstituteEditsLocked(edits)
	return nil
}

// applySubstituteEditsLocked applies edits — already in ascending offset
// order, since lines are processed top-to-bottom and matches within a
// line are found left-to-right — end-to-start, so an earlier edit's
// offset is never invalidated by a later one already applied. Same
// reasoning as applyTextEditsLocked; a separate method since the edit
// shapes genuinely differ (byte offsets here, LSP Positions there). Like
// applyTextEditsLocked, a substitute with N matches produces N separate
// undo steps, not one — an existing, already-accepted tradeoff in this
// codebase, not a new one. Caller must already hold e.mu.
func (e *Editor) applySubstituteEditsLocked(edits []substituteEdit) {
	for i := len(edits) - 1; i >= 0; i-- {
		ed := edits[i]
		e.table.Delete(ed.Start, ed.End-ed.Start)
		if len(ed.NewText) > 0 {
			e.table.Insert(ed.Start, []byte(ed.NewText))
		}
	}
	e.finishBufferMutationLocked(e.cursor.ByteOffset)
}

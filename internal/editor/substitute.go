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
	"github.com/fliplucky/pieces-store/internal/types"
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

	matches := e.findSubstituteMatchesLocked(re, global, cmd.WholeBuffer)
	if len(matches) == 0 {
		return ErrPatternNotFound
	}
	edits := make([]substituteEdit, len(matches))
	for i, m := range matches {
		// re.ReplaceAllString, not a literal splice: Go's regexp
		// interprets $1/$name capture-group references in the
		// replacement automatically, applied to just this one match.
		edits[i] = substituteEdit{Start: m.Start, End: m.End, NewText: re.ReplaceAllString(m.Text, sub.Replacement)}
	}
	e.applySubstituteEditsLocked(edits)
	return nil
}

// substituteMatchLoc is one match's byte range (plus its own matched text,
// needed to compute a replacement) within the buffer — the pure "where"
// half of a :s/:%s command, factored out so executeSubstituteLocked and
// SubstitutePreviewRanges (the live inccommand-style highlight, see below)
// share one scanning pass instead of two copies of the same line-by-line
// logic.
type substituteMatchLoc struct {
	Start, End int
	Text       string
}

// findSubstituteMatchesLocked finds every match a :s/:%s command with the
// given compiled pattern would act on — current line only by default, or
// every line when wholeBuffer (the "%" prefix); only the first match per
// line unless global. Caller must already hold e.mu (a read lock suffices
// — this never mutates the table).
func (e *Editor) findSubstituteMatchesLocked(re *regexp.Regexp, global, wholeBuffer bool) []substituteMatchLoc {
	fullText := e.table.CombinePieces()
	lines := strings.Split(fullText, "\n")
	starts := make([]int, len(lines))
	pos := 0
	for i, line := range lines {
		starts[i] = pos
		pos += len(line) + 1 // +1 for the "\n" every line but the last has
	}

	lineIndices := []int{offset.ByteOffsetToPosition(e.table, e.cursor.ByteOffset).Row}
	if wholeBuffer {
		lineIndices = make([]int, len(lines))
		for i := range lines {
			lineIndices[i] = i
		}
	}

	var matches []substituteMatchLoc
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
			matches = append(matches, substituteMatchLoc{Start: base + loc[0], End: base + loc[1], Text: line[loc[0]:loc[1]]})
		}
	}
	return matches
}

// SubstitutePreviewRanges returns the buffer ranges a :s/:%s command
// currently being typed in Command mode would act on, for a frontend to
// highlight live — nvim's own "incsearch"/"inccommand" preview, scoped
// here to highlighting the matches themselves rather than previewing the
// replaced text in place (a bigger feature — it would need a non-
// committed, purely visual overlay edit — deliberately deferred past this
// first pass). Updates progressively as the pattern itself is typed, not
// just once the whole command is complete: SplitSubstituteSegments (not
// the stricter ParseSubstitute, which requires a full pattern/replacement/
// flags triple) is used specifically so "s/hel" already highlights every
// match of "hel", exactly like typing a search pattern incrementally
// narrows/moves nvim's own incsearch highlight. The "g" flag only takes
// effect once its segment has actually been typed (i.e. once a
// replacement segment already exists) — matching how real execution reads
// it, not a moment earlier. Empty whenever the command buffer isn't even
// shaped like a substitute yet: wrong command name, no delimiter typed at
// all, an empty pattern, or a pattern that doesn't compile. Typing is
// naturally incomplete most of the time, so this stays silent on all of
// those rather than erroring on every keystroke — same as real nvim.
func (e *Editor) SubstitutePreviewRanges() []offset.TextRange {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.substitutePreviewRangesLocked()
}

func (e *Editor) substitutePreviewRangesLocked() []offset.TextRange {
	if e.cursor.Mode != types.ModeCommand {
		return nil
	}
	cmd := exmode.Parse(e.cursor.CommandBuffer.String())
	if cmd.Name != "s" && cmd.Name != "substitute" {
		return nil
	}
	segments, _, ok := exmode.SplitSubstituteSegments(cmd.Raw)
	if !ok || segments[0] == "" {
		return nil
	}
	re, err := regexp.Compile(segments[0])
	if err != nil {
		return nil
	}
	global := len(segments) >= 3 && strings.Contains(segments[2], "g")

	matches := e.findSubstituteMatchesLocked(re, global, cmd.WholeBuffer)
	ranges := make([]offset.TextRange, len(matches))
	for i, m := range matches {
		ranges[i] = offset.TextRange{Start: m.Start, Length: m.End - m.Start}
	}
	return ranges
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

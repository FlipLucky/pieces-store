// Package interpreter is Processor's home in the Validator → Processor →
// Executor key-handling pipeline (see DESIGN_KEY_PIPELINE.md and
// CONCEPTS.md). It sits above both internal/keyengine (Validator's current
// home for Normal mode) and internal/offset, importing both without either
// depending back — the same zero-dependency-sibling property those two
// packages already preserve between each other.
//
// ProcessNormal's job is resolution only: a validated keyengine command
// plus the mode and a read-only position snapshot in, a fully-resolved
// Action out. It never mutates anything and never talks to *editor.Editor
// — resolveMotion/resolveRange below are pure functions of offset.Document,
// exactly as they were when they lived in internal/editor/dispatch.go.
package interpreter

import (
	"github.com/fliplucky/pieces-store/internal/keyengine"
	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/types"
)

// Position is a read-only snapshot of where the cursor is right now —
// everything ProcessNormal needs to resolve a motion or range, without any
// access to Editor's live, mutable state. Row/Col are only needed for j/k's
// column-preserving vertical motion (see resolveLineStep); every other
// motion only needs ByteOffset.
type Position struct {
	ByteOffset int
	Row        int
	Col        int
}

// Action is Normal mode's Processor output: a fully resolved, self-
// describing unit of work — mode-blind, since mode has already done its
// job interpreting the sequence by the time one of these exists. Declared
// locally rather than imported from internal/executor, so this package
// never needs to import Executor's package to produce values it accepts —
// see DESIGN_KEY_PIPELINE.md's "Decoupled by shape, not by sender." Any
// type with a Kind() method satisfies both this and executor.Action
// structurally.
type Action interface {
	Kind() types.ActionKind
}

// MoveAction is a fully-resolved cursor target. Ok is false when the
// requested motion couldn't be resolved at all (an unknown noun, or j/k
// already at the first/last line) — a no-op, not a crash, matching
// resolveMotion's long-standing contract.
type MoveAction struct {
	Target int
	Ok     bool
}

func (a MoveAction) Kind() types.ActionKind { return types.ActionMove }
func (a MoveAction) Offset() (int, bool)    { return a.Target, a.Ok }

// DeleteAction is a fully-resolved deletion range. Ok is false when the
// Modifier+Noun combination didn't resolve to a known range.
type DeleteAction struct {
	Rng offset.TextRange
	Ok  bool
}

func (a DeleteAction) Kind() types.ActionKind          { return types.ActionDelete }
func (a DeleteAction) Range() (offset.TextRange, bool) { return a.Rng, a.Ok }

// UndoAction/RedoAction carry the repeat count straight from the validated
// command (keyengine already parsed any leading digit accumulation) —
// Executor's job is just the repeat loop, not re-deriving the count.
type UndoAction struct{ RepeatCount int }

func (a UndoAction) Kind() types.ActionKind { return types.ActionUndo }
func (a UndoAction) Count() int             { return a.RepeatCount }

type RedoAction struct{ RepeatCount int }

func (a RedoAction) Kind() types.ActionKind { return types.ActionRedo }
func (a RedoAction) Count() int             { return a.RepeatCount }

// BeginReplaceAction is r's first keystroke: no payload, just tells
// Executor to put the editor into "the next raw key is what to replace
// with" mode. The second keystroke bypasses keyengine/ProcessNormal
// entirely (there's no sequence to validate — any rune or <Esc> is
// accepted) and arrives as a ReplaceRuneAction constructed directly by
// Editor instead — see DESIGN_KEY_PIPELINE.md's "Decoupled by shape, not
// by sender": Executor accepts Actions from anything, not just Processor.
type BeginReplaceAction struct{}

func (a BeginReplaceAction) Kind() types.ActionKind { return types.ActionBeginReplace }

// ReplaceRuneAction is r's second keystroke, constructed by Editor itself
// once the raw replacement rune arrives.
type ReplaceRuneAction struct{ R rune }

func (a ReplaceRuneAction) Kind() types.ActionKind { return types.ActionReplaceRune }
func (a ReplaceRuneAction) Rune() rune             { return a.R }

type EnterInsertAction struct{}

func (a EnterInsertAction) Kind() types.ActionKind { return types.ActionEnterInsert }

type EnterCommandAction struct{}

func (a EnterCommandAction) Kind() types.ActionKind { return types.ActionEnterCommand }

// OpenLineAction is o/O: open a blank line below/above the current one.
type OpenLineAction struct{ IsBelow bool }

func (a OpenLineAction) Kind() types.ActionKind { return types.ActionOpenLine }
func (a OpenLineAction) Below() bool            { return a.IsBelow }

type HoverAction struct{}

func (a HoverAction) Kind() types.ActionKind { return types.ActionHover }

// ProcessNormal is Normal mode's Processor: a validated keyengine command,
// the document to resolve motions/ranges against, and a position snapshot,
// in — a fully-resolved Action out. Returns nil only for a Verb this
// function doesn't know about, which shouldn't happen for anything
// keyengine's own whitelist tables actually produce.
func ProcessNormal(cmd *keyengine.ExecutableCommand, doc offset.Document, at Position) Action {
	count := cmd.Count
	if count < 1 {
		count = 1
	}

	switch cmd.Verb {
	case keyengine.Move:
		return resolveMove(doc, cmd.Noun, at, count)
	case keyengine.Delete:
		rng, ok := resolveRange(doc, cmd.Modifier, cmd.Noun, at.ByteOffset)
		return DeleteAction{Rng: rng, Ok: ok}
	case keyengine.Undo:
		return UndoAction{RepeatCount: count}
	case keyengine.Redo:
		return RedoAction{RepeatCount: count}
	case keyengine.Replace:
		return BeginReplaceAction{}
	case keyengine.EnterInsert:
		return EnterInsertAction{}
	case keyengine.EnterCommand:
		return EnterCommandAction{}
	case keyengine.OpenBelow:
		return OpenLineAction{IsBelow: true}
	case keyengine.OpenAbove:
		return OpenLineAction{IsBelow: false}
	case keyengine.Hover:
		return HoverAction{}
	default:
		return nil
	}
}

// resolveMove turns a Move command's noun into a fully-resolved target
// offset. j/k (LineUp/LineDown) need Row/Col to preserve the intended
// column across lines of different lengths — not expressible as a pure
// function of byte offset alone — so they bypass the generic
// resolveMotion accumulation loop entirely, mirroring how
// executeMove/moveCursorUpLocked/moveCursorDownLocked used to split this
// same way inside internal/editor.
func resolveMove(doc offset.Document, noun keyengine.KeyActionName, at Position, count int) MoveAction {
	switch noun {
	case keyengine.LineUp:
		return resolveLineStep(doc, at, count, -1)
	case keyengine.LineDown:
		return resolveLineStep(doc, at, count, 1)
	}

	target := at.ByteOffset
	for i := 0; i < count; i++ {
		next, ok := resolveMotion(doc, noun, target)
		if !ok {
			return MoveAction{Ok: false}
		}
		target = next
	}
	return MoveAction{Target: target, Ok: true}
}

// resolveLineStep walks count lines up (dir<0) or down (dir>0) from at,
// preserving Col across lines of different lengths. A step that can't
// advance (already at the first/last line) stops the walk there, keeping
// whatever progress earlier steps made — matching
// moveCursorUpLocked/moveCursorDownLocked's original per-call guards
// exactly, just folded into one loop instead of N separate locked calls.
func resolveLineStep(doc offset.Document, at Position, count int, dir int) MoveAction {
	cur := at
	moved := false

	for i := 0; i < count; i++ {
		if dir < 0 {
			if cur.Row <= 0 {
				break
			}
			targetPos := offset.Position{Row: cur.Row - 1, Col: cur.Col}
			newOffset := offset.PositionToByteOffset(doc, targetPos)
			actual := offset.ByteOffsetToPosition(doc, newOffset)
			cur = Position{ByteOffset: newOffset, Row: actual.Row, Col: actual.Col}
			moved = true
			continue
		}

		// Down always targets the row right after the current one, so the
		// current position is a valid forward-scan anchor — resumes from
		// here instead of rescanning the whole buffer from offset 0.
		targetPos := offset.Position{Row: cur.Row + 1, Col: cur.Col}
		newOffset := offset.PositionToByteOffset(doc, targetPos, cur.ByteOffset, cur.Row)
		actual := offset.ByteOffsetToPosition(doc, newOffset)
		if actual.Row <= cur.Row {
			// Target row doesn't actually exist (already on the last
			// line) — PositionToByteOffset fell through to doc.Len().
			break
		}
		cur = Position{ByteOffset: newOffset, Row: actual.Row, Col: actual.Col}
		moved = true
	}

	if !moved {
		return MoveAction{Ok: false}
	}
	return MoveAction{Target: cur.ByteOffset, Ok: true}
}

// resolveMotion turns a Move command's noun into a target offset, given
// the current offset to move from. ok is false for a noun this doesn't
// (yet) know how to resolve — a no-op, not a crash.
func resolveMotion(doc offset.Document, noun keyengine.KeyActionName, from int) (target int, ok bool) {
	switch noun {
	case keyengine.Word:
		return offset.MotionWordForward(doc, from), true
	case keyengine.WordBackward:
		return offset.MotionWordBackward(doc, from), true
	case keyengine.WordEnd:
		return offset.MotionWordEnd(doc, from), true
	case keyengine.CharBackward:
		return offset.MoveLeft(doc, from), true
	case keyengine.CharForward:
		return offset.MoveRight(doc, from), true
	default:
		return from, false
	}
}

// resolveRange turns a Delete command's Modifier+Noun into a concrete
// TextRange at the given offset. Same no-op-on-unknown contract as
// resolveMotion.
func resolveRange(doc offset.Document, modifier, noun keyengine.KeyActionName, at int) (offset.TextRange, bool) {
	switch {
	case modifier == keyengine.Inside && noun == keyengine.Word:
		return offset.RangeInnerWord(doc, at), true
	case modifier == keyengine.Around && noun == keyengine.Word:
		return offset.RangeAroundWord(doc, at), true
	case modifier == keyengine.Inside && noun == keyengine.Paragraph:
		return offset.RangeInnerParagraph(doc, at), true
	case noun == keyengine.LineMotion:
		return offset.RangeLine(doc, at), true
	default:
		return offset.TextRange{}, false
	}
}

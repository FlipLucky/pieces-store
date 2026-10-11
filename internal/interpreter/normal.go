// Package interpreter is Processor's home in the Validator → Processor →
// Executor key-handling pipeline (see docs/concepts/validator-processor-
// executor.md and DESIGN_KEY_PIPELINE.md for the full history). It sits
// above both internal/keyengine (Validator's current
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
	clipboardregister "github.com/fliplucky/pieces-store/internal/clipboard-register"
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

// PasteClipboard is a read-only snapshot of what p/P would currently
// paste — the unnamed register's newest value, if any. Same snapshot
// pattern as Position/RuneContext: Processor never touches
// *clipboardregister.ClipboardRegister directly, Editor reads it once
// under the same lock as everything else and passes the value in.
//
// Linewise mirrors the entry's own flag (set by whichever of d/c/y
// produced it, via Noun == LineMotion) — Paste/PasteBefore branch on it to
// decide whether the stored text becomes its own fresh line or splices in
// at the cursor the way a charwise insert does.
type PasteClipboard struct {
	Value    string
	Linewise bool
	Ok       bool
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
//
// Register/StoreInRegister are deliberately zero-value by default (no
// register) rather than a required argument — Insert mode's backspace
// (insert.go's resolveBackspace) constructs a DeleteAction too and must
// NOT yank into a register the way d/c do, so the zero value has to mean
// "just delete," not "delete into the unnamed register." Normal mode's
// Delete/Change verbs below are the only producers that ever set these.
//
// IsLinewise is only meaningful when StoreInRegister is true (set from
// Noun == LineMotion, i.e. dd/cc's doubled-verb form) — Executor threads it
// through to the register entry so a later linewise paste can tell dd's
// whole-line delete apart from diw's charwise one.
type DeleteAction struct {
	Rng             offset.TextRange
	Ok              bool
	Register        clipboardregister.RegisterType
	StoreInRegister bool
	IsLinewise      bool
}

func (a DeleteAction) Kind() types.ActionKind          { return types.ActionDelete }
func (a DeleteAction) Range() (offset.TextRange, bool) { return a.Rng, a.Ok }
func (a DeleteAction) RegisterDestination() (clipboardregister.RegisterType, bool) {
	return a.Register, a.StoreInRegister
}
func (a DeleteAction) Linewise() bool { return a.IsLinewise }

// YankAction is a fully-resolved yank range: store its text into a
// register, move the cursor to its start, never touch the buffer. Ok is
// false for the same reason as DeleteAction's. IsLinewise is set from
// Noun == LineMotion (yy), same as DeleteAction's.
type YankAction struct {
	Rng        offset.TextRange
	Ok         bool
	IsLinewise bool
}

func (a YankAction) Kind() types.ActionKind          { return types.ActionYank }
func (a YankAction) Range() (offset.TextRange, bool) { return a.Rng, a.Ok }
func (a YankAction) Linewise() bool                  { return a.IsLinewise }

// UndoAction/RedoAction carry the repeat count straight from the validated
// command (keyengine already parsed any leading digit accumulation) —
// Executor's job is just the repeat loop, not re-deriving the count.
type UndoAction struct{ RepeatCount int }

func (a UndoAction) Kind() types.ActionKind { return types.ActionUndo }
func (a UndoAction) Count() int             { return a.RepeatCount }

// RedoAction mirrors UndoAction exactly, for <C-r> instead of u.
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

// BeginFindCharAction is f/F's first keystroke: tells Executor to put the
// editor into "the next raw key is the target char to search for" mode,
// same shape as BeginReplaceAction/ReplaceRuneAction for r. IsForward
// records which of f/F started the pending state, carried through so the
// second keystroke (constructed directly by Editor, bypassing this
// package's normal ProcessNormal path entirely — there's no sequence to
// validate) knows which direction to search.
type BeginFindCharAction struct{ IsForward bool }

func (a BeginFindCharAction) Kind() types.ActionKind { return types.ActionBeginFindChar }
func (a BeginFindCharAction) Forward() bool          { return a.IsForward }

// ResolveFindChar is f/F's second keystroke: given the raw rune the user
// just typed and which direction the pending state was started in,
// resolves where the cursor would land. Constructed directly by Editor
// once the target rune arrives, same as ReplaceRuneAction — f/F are
// standalone motions, not something keyengine's grammar validates a
// second keystroke against.
func ResolveFindChar(doc offset.Document, at Position, target rune, forward bool) MoveAction {
	if forward {
		return MoveAction{Target: offset.MotionFindCharForward(doc, at.ByteOffset, string(target)), Ok: true}
	}
	return MoveAction{Target: offset.MotionFindCharBackward(doc, at.ByteOffset, string(target)), Ok: true}
}

// ProcessNormal is Normal mode's Processor: a validated keyengine command,
// the document to resolve motions/ranges against, a position snapshot, and
// what p/P would currently paste, in — one or more fully-resolved Actions
// out, applied by Executor in order (e.g. Change is "delete this range"
// then "enter Insert mode"; Paste-after is "move past the cursor" then
// "insert here" — see docs/concepts/validator-processor-executor.md for
// why this is the Processor's intended output shape, not a special case).
// Returns nil only for a Verb this function doesn't know about,
// which shouldn't happen for anything keyengine's own whitelist tables
// actually produce.
func ProcessNormal(cmd *keyengine.ExecutableCommand, doc offset.Document, at Position, clip PasteClipboard) []Action {
	count := cmd.Count
	if count < 1 {
		count = 1
	}

	switch cmd.Verb {
	case keyengine.Move:
		return one(resolveMove(doc, cmd.Noun, at, count))
	case keyengine.Delete:
		rng, ok := resolveRange(doc, cmd.Modifier, cmd.Noun, at.ByteOffset, count)
		linewise := cmd.Noun == keyengine.LineMotion
		return one(DeleteAction{Rng: rng, Ok: ok, Register: clipboardregister.Delete, StoreInRegister: ok, IsLinewise: linewise})
	case keyengine.Change:
		rng, ok := resolveRange(doc, cmd.Modifier, cmd.Noun, at.ByteOffset, count)
		if !ok {
			return nil
		}
		linewise := cmd.Noun == keyengine.LineMotion
		return []Action{
			DeleteAction{Rng: rng, Ok: true, Register: clipboardregister.Delete, StoreInRegister: true, IsLinewise: linewise},
			EnterInsertAction{},
		}
	case keyengine.Yank:
		rng, ok := resolveRange(doc, cmd.Modifier, cmd.Noun, at.ByteOffset, count)
		linewise := cmd.Noun == keyengine.LineMotion
		return one(YankAction{Rng: rng, Ok: ok, IsLinewise: linewise})
	case keyengine.Undo:
		return one(UndoAction{RepeatCount: count})
	case keyengine.Redo:
		return one(RedoAction{RepeatCount: count})
	case keyengine.Replace:
		return one(BeginReplaceAction{})
	case keyengine.EnterInsert:
		return one(EnterInsertAction{})
	case keyengine.EnterCommand:
		return one(EnterCommandAction{})
	case keyengine.OpenBelow:
		return one(OpenLineAction{IsBelow: true})
	case keyengine.OpenAbove:
		return one(OpenLineAction{IsBelow: false})
	case keyengine.Hover:
		return one(HoverAction{})
	case keyengine.FindCharForward:
		return one(BeginFindCharAction{IsForward: true})
	case keyengine.FindCharBackward:
		return one(BeginFindCharAction{IsForward: false})
	case keyengine.Paste:
		if !clip.Ok {
			return nil
		}
		if clip.Linewise {
			// Linewise content already carries its own trailing "\n" (it
			// was captured via RangeLine, same as dd/yy's own delete/yank
			// range) — inserting it whole at the start of the *next* row
			// lands it as a genuine new line below the current one,
			// matching real vim's linewise p, rather than splicing into
			// whatever line the cursor happens to sit on.
			lineRange := offset.RangeLine(doc, at.ByteOffset)
			insertAt := lineRange.Start + lineRange.Length
			value := clip.Value
			caret := 0
			// The current line is the buffer's last line and has no
			// trailing "\n" of its own (RangeLine's Length never counts a
			// newline that isn't there) — insertAt sits at the literal end
			// of the buffer with nothing separating it from the line
			// above, so the pasted text's own trailing "\n" would end up
			// *after* it instead of between the two lines. A leading "\n"
			// here is the separator that newline would otherwise have
			// been; caret skips past it so the cursor still lands on the
			// pasted line's own first rune, not on the separator.
			if insertAt >= doc.Len() && insertAt > 0 {
				if r, _ := doc.GetRuneAt(insertAt - 1); r != '\n' {
					value = "\n" + value
					caret = 1
				}
			}
			return []Action{
				MoveAction{Target: insertAt, Ok: true},
				InsertTextAction{Text: value, CaretOffset: caret},
			}
		}
		return []Action{
			MoveAction{Target: offset.MoveRight(doc, at.ByteOffset), Ok: true},
			InsertTextAction{Text: clip.Value, CaretOffset: len(clip.Value)},
		}
	case keyengine.PasteBefore:
		if !clip.Ok {
			return nil
		}
		if clip.Linewise {
			lineRange := offset.RangeLine(doc, at.ByteOffset)
			return []Action{
				MoveAction{Target: lineRange.Start, Ok: true},
				InsertTextAction{Text: clip.Value, CaretOffset: 0},
			}
		}
		return one(InsertTextAction{Text: clip.Value, CaretOffset: len(clip.Value)})
	default:
		return nil
	}
}

// one wraps a single Action as the []Action slice ProcessNormal always
// returns — most verbs only ever produce one.
func one(a Action) []Action { return []Action{a} }

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

// resolveRange turns a Delete/Change/Yank command's Modifier+Noun into a
// concrete TextRange at the given offset, repeated count times by
// chaining each resolution onto the end of the last — 3diw deletes three
// consecutive "inner word" units, 3dd three consecutive lines, and so on
// (matches real vim's own documented counted-iw behavior: a whitespace
// run between words counts as its own unit in the chain, same as a word
// does). A count that runs past the document's end naturally saturates
// rather than erroring — offset.RangeLine/RangeInnerWord/etc. all clamp
// to the document boundary on their own, so chaining past it just keeps
// re-resolving the same boundary range instead of growing further (see
// TestCountedDeleteStopsAtDocumentEnd). ok is false only when the
// Modifier+Noun combination itself isn't recognized at all — fixed for
// the whole call, so once the first resolution succeeds every later one
// in the chain does too; there's no per-iteration failure case to guard.
func resolveRange(doc offset.Document, modifier, noun keyengine.KeyActionName, at int, count int) (offset.TextRange, bool) {
	if count < 1 {
		count = 1
	}
	rng, ok := resolveRangeOnce(doc, modifier, noun, at)
	if !ok {
		return offset.TextRange{}, false
	}
	for i := 1; i < count; i++ {
		next, _ := resolveRangeOnce(doc, modifier, noun, rng.Start+rng.Length)
		rng.Length = (next.Start + next.Length) - rng.Start
	}
	return rng, true
}

// resolveRangeOnce is resolveRange's single-application resolver — same
// no-op-on-unknown contract as resolveMotion.
func resolveRangeOnce(doc offset.Document, modifier, noun keyengine.KeyActionName, at int) (offset.TextRange, bool) {
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

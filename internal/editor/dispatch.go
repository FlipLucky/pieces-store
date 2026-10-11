package editor

import (
	"fmt"

	"github.com/fliplucky/pieces-store/internal/autopairs"
	"github.com/fliplucky/pieces-store/internal/executor"
	"github.com/fliplucky/pieces-store/internal/interpreter"
	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/piecetable"
	"github.com/fliplucky/pieces-store/internal/types"
)

// HandleKey is the single entry point for all keyboard input. It routes on
// the current mode: Insert/Command mode capture keys literally (no vim
// grammar applies to "just type this character"); Normal (and eventually
// Visual) mode hands the key to the keyengine parser and, once a sequence
// resolves, executes it. Returns true if the key was consumed.
func (e *Editor) HandleKey(key string) bool {
	switch types.Mode(e.GetModeInt()) {
	case types.ModeInsert:
		return e.handleInsertModeKey(key)
	case types.ModeCommand:
		return e.handleCommandModeKey(key)
	default:
		return e.handleNormalModeKey(key)
	}
}

func (e *Editor) handleInsertModeKey(key string) bool {
	switch key {
	case "<Esc>":
		e.DismissLSPCompletion()
		e.SetMode(types.ModeNormal)
	case "<BS>":
		e.deleteBackwardWithPairCollapse()
	case "<Enter>":
		if e.LSPCompletionActive() {
			e.ConfirmLSPCompletion()
			break
		}
		indent := e.IndentStringAt(e.GetCursor().ByteOffset)
		e.InsertText([]byte("\n" + indent))
	case "<C-n>":
		e.CycleLSPCompletion(1)
	case "<C-p>":
		e.CycleLSPCompletion(-1)
	case "<Left>":
		e.MoveCursorLeft()
	case "<Right>":
		e.MoveCursorRight()
	case "<Up>":
		e.MoveCursorUp()
	case "<Down>":
		e.MoveCursorDown()
	default:
		if runes := []rune(key); len(runes) == 1 {
			e.insertRuneWithAutoPairing(runes[0])
			e.maybeTriggerLSPCompletion(runes[0])
		}
	}
	return true
}

// insertRuneWithAutoPairing implements auto-closing brackets and quotes —
// the decision itself lives in internal/autopairs (a pure function: three
// runes in, an Action out), this is just the imperative shell reading the
// runes around the cursor and acting on whatever it decides. Typing an
// opening bracket ({, (, [) inserts its matching close too, in one edit,
// cursor landing between them ready to type the contents. Typing a closing
// bracket (or quote) that's already sitting right at the cursor (because
// it was just auto-inserted) moves past it instead of inserting a
// redundant second one — without that, auto-close actively fights anyone
// who types their own closing character out of habit, which is worse than
// not having the feature at all.
func (e *Editor) insertRuneWithAutoPairing(r rune) {
	before, after := e.runesAroundCursorLocked()
	switch action, closing := autopairs.Decide(r, before, after); action {
	case autopairs.InsertPair:
		e.InsertText([]byte{byte(r), byte(closing)})
		e.MoveCursorLeft()
	case autopairs.SkipOver:
		e.MoveCursorRight()
	default:
		e.InsertText([]byte(string(r)))
	}
}

// runesAroundCursorLocked returns the runes immediately before and after
// the cursor — 0 for either one that doesn't exist (start/end of buffer).
func (e *Editor) runesAroundCursorLocked() (before, after rune) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	at := e.cursor.ByteOffset
	if at > 0 {
		before, _ = e.table.GetRuneAt(at - 1)
	}
	if at < e.table.Len() {
		after, _ = e.table.GetRuneAt(at)
	}
	return before, after
}

// deleteBackwardWithPairCollapse is <BS>'s usual behavior, except when the
// cursor sits inside an empty auto-closed pair ({|}, (|), [|], "|", | =
// cursor) — then it deletes both characters in one edit, collapsing the
// pair, rather than leaving a dangling unmatched closer behind.
func (e *Editor) deleteBackwardWithPairCollapse() {
	before, after := e.runesAroundCursorLocked()
	if !autopairs.IsEmptyPair(before, after) {
		e.DeleteText()
		return
	}
	e.mu.Lock()
	at := e.cursor.ByteOffset
	e.table.Delete(at-1, 2)
	e.moveCursorToLocked(at-1, piecetable.Edit{Offset: at - 1, OldLength: 2})
	e.mu.Unlock()
}

func (e *Editor) handleCommandModeKey(key string) bool {
	switch key {
	case "<Esc>":
		e.SetMode(types.ModeNormal)
	case "<BS>":
		e.CancelCompletion()
		e.BackspaceCommandBuffer()
	case "<Enter>":
		e.CancelCompletion()
		e.ExecuteCommand(e.GetCommandBuffer())
	case "<Tab>":
		e.TriggerCompletion()
	case "<C-n>":
		e.CycleCompletion(1)
	case "<C-p>":
		e.CycleCompletion(-1)
	default:
		if runes := []rune(key); len(runes) == 1 {
			e.CancelCompletion()
			e.AppendCommandBuffer(runes[0])
		}
	}
	return true
}

func (e *Editor) handleNormalModeKey(key string) bool {
	// "r" (Replace) needs the *next* raw keystroke as its argument rather
	// than being looked up in the verb/modifier/noun tables — the same
	// bounded, one-keystroke literal-capture primitive Insert/Command mode
	// use, just scoped to exactly one key before returning to parsing. It
	// bypasses keyengine/interpreter entirely (there's no sequence to
	// validate here) and constructs the Action directly — see
	// interpreter.ReplaceRuneAction's doc comment.
	if e.pendingReplace {
		e.pendingReplace = false
		if key != "<Esc>" {
			if runes := []rune(key); len(runes) == 1 {
				e.mu.Lock()
				e.dispatchLocked(interpreter.ReplaceRuneAction{R: runes[0]})
				e.mu.Unlock()
			}
		}
		return true
	}

	e.commandContext.SetEditorMode(types.Mode(e.GetModeInt()))
	cmd, err := e.commandContext.ProcessCommand(key)
	if err != nil {
		// A whitelist/semantic-table inconsistency is a bug in the tables
		// themselves, not user input — surface it loudly rather than
		// swallow it like a plain fizzle.
		panic(fmt.Sprintf("keyengine: %v (key=%q)", err, key))
	}
	if cmd == nil {
		return true
	}

	// Resolution (interpreter.ProcessNormal, reading e.table/cursor state)
	// and application (executor.Dispatch) are one atomic section, not two
	// separately-locked steps — otherwise another goroutine (async LSP
	// work, see async.go) could mutate the table in between, resolving a
	// motion/range against state that's already stale by the time it's
	// applied. This is the same reason executeMove/executeDelete used to
	// hold e.mu for their whole resolve-then-apply body.
	e.mu.Lock()
	pos := interpreter.Position{ByteOffset: e.cursor.ByteOffset, Row: e.cursor.Row, Col: e.cursor.Col}
	action := interpreter.ProcessNormal(cmd, e.table, pos)
	e.dispatchLocked(action)
	e.mu.Unlock()
	return true
}

// Dispatch performs an already-resolved Action — the entry point for any
// producer that isn't first resolving against live Editor state the way
// interpreter.ProcessNormal does (e.g. a future LSP-driven edit, or a
// plugin/Lua call constructing an Action directly — see
// DESIGN_KEY_PIPELINE.md's "Decoupled by shape, not by sender"). Keypress-
// driven Actions go through handleNormalModeKey's own locked block instead
// of this, since resolving against e.table/cursor and applying the result
// need to be one atomic section — see its comment.
func (e *Editor) Dispatch(action executor.Action) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.dispatchLocked(action)
}

// dispatchLocked is executor.Dispatch wrapped for callers that already
// hold e.mu. Caller must already hold e.mu.
func (e *Editor) dispatchLocked(action executor.Action) {
	if action == nil {
		return
	}
	executor.Dispatch(action, lockedEditorView{e})
}

// lockedEditorView is internal/executor's Editor interface, satisfied by
// Editor without ever being imported by internal/executor (mirrors
// viewmanager.Document's precedent for the same problem). Every method
// here assumes e.mu is already held by its caller (dispatchLocked) — they
// call Editor's own already-existing *Locked bodies directly rather than
// the public self-locking versions, which would deadlock re-entering the
// same mutex (the same class of bug already caught once this session for
// :Format/StartFormat).
type lockedEditorView struct{ e *Editor }

func (v lockedEditorView) MoveCursorToOffset(target int) {
	v.e.moveCursorToLocked(target, piecetable.Edit{})
}

func (v lockedEditorView) DeleteRange(rng offset.TextRange) {
	v.e.table.Delete(rng.Start, rng.Length)
	newOffset := rng.Start
	if newOffset > v.e.table.Len() {
		newOffset = v.e.table.Len()
	}
	v.e.moveCursorToLocked(newOffset, piecetable.Edit{Offset: rng.Start, OldLength: rng.Length})
}

func (v lockedEditorView) Undo() bool { return v.e.undoLocked() }
func (v lockedEditorView) Redo() bool { return v.e.redoLocked() }

func (v lockedEditorView) SetMode(m types.Mode) { v.e.setModeLocked(m) }

func (v lockedEditorView) BeginPendingReplace() { v.e.pendingReplace = true }

// ReplaceRune implements r's second keystroke: swap exactly the rune
// under the cursor for the one just typed, leaving the cursor in place.
func (v lockedEditorView) ReplaceRune(r rune) {
	at := v.e.cursor.ByteOffset
	if at >= v.e.table.Len() {
		return
	}
	_, width := v.e.table.GetRuneAt(at)
	if width <= 0 {
		width = 1
	}
	v.e.table.Delete(at, width)
	newRune := []byte(string(r))
	v.e.table.Insert(at, newRune)
	v.e.moveCursorToLocked(at, piecetable.Edit{Offset: at, OldLength: width, NewLength: len(newRune)})
}

// OpenLine implements o (below) / O (above): insert a blank line adjacent
// to the current one, pre-indented to match the nesting at that point
// (see Editor.IndentStringAt), and switch to Insert mode with the cursor
// positioned after that indent.
//
// The indent goes *before* the inserted "\n", not after — insertAt already
// sits at the boundary where the next row starts (RangeLine's Length
// includes the trailing newline), so inserting "\n"+indent there would
// make the indent leading content of the row *after* the new blank one,
// not the blank row itself (confirmed the hard way: it silently pushed
// typed text onto the wrong line). indent+"\n" makes indent the new row's
// own content, correctly terminated by the fresh newline.
//
// Folds the mode switch into the same locked section as the insert itself
// (the original openLine unlocked and re-locked between the two, via the
// public, self-locking SetMode) — a strict atomicity improvement, not a
// behavior change: CommandBuffer/Completion/LSPCompletion are necessarily
// already zero-value coming from Normal mode, so SetMode's cleanup for
// those is a no-op here either way.
func (v lockedEditorView) OpenLine(below bool) {
	rng := offset.RangeLine(v.e.table, v.e.cursor.ByteOffset)
	insertAt := rng.Start
	if below {
		insertAt = rng.Start + rng.Length
	}
	indent := v.e.indentStringAtLocked(insertAt)
	text := indent + "\n"
	v.e.table.Insert(insertAt, []byte(text))
	v.e.moveCursorToLocked(insertAt+len(indent), piecetable.Edit{Offset: insertAt, NewLength: len(text)})
	v.e.cursor.SetMode(types.ModeInsert)
}

func (v lockedEditorView) RequestHover() { v.e.requestHoverLocked() }

// moveCursorToLocked updates the cursor to newOffset (recomputing its
// screen position), feeds edit to the syntax highlighter (if any — see
// setupHighlighterLocked), and notifies listeners of edit — the delta that
// caused this move, or piecetable.Edit{} for pure cursor movement with no
// document change. Callers must already hold e.mu.
func (e *Editor) moveCursorToLocked(newOffset int, edit piecetable.Edit) {
	gridPos := offset.ByteOffsetToPosition(e.table, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)
	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.dismissHoverLocked()

	if edit != (piecetable.Edit{}) && (e.highlighter != nil || e.lspService.Active()) {
		// CombinePieces materializes the whole buffer — a real,
		// already-tracked cost for large files (same family as
		// FindPieceAt's O(P) scan in BACKLOG.md), accepted for now since
		// the incremental parse itself is cheap regardless of document
		// size (~650ns/edit per the gotreesitter benchmarks) and this
		// isn't on the render path — it runs once per edit, not once per
		// frame. Computed once and shared by both consumers below rather
		// than each re-materializing the same buffer.
		source := []byte(e.table.CombinePieces())
		if e.highlighter != nil {
			e.highlighter.Update(edit, source)
		}
		if e.lspService.Active() {
			e.notifyLSPDidChangeLocked(source)
		}
	}

	e.notifyChange(edit)
}

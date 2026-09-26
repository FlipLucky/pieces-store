package editor

import (
	"fmt"

	"github.com/fliplucky/pieces-store/internal/autopairs"
	"github.com/fliplucky/pieces-store/internal/keyengine"
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
	// use, just scoped to exactly one key before returning to parsing.
	if e.pendingReplace {
		e.pendingReplace = false
		if key != "<Esc>" {
			if runes := []rune(key); len(runes) == 1 {
				e.replaceCurrentRune(runes[0])
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
	e.execute(cmd)
	return true
}

// execute applies a fully-resolved command from the key engine. This is the
// "action" side of action+offset: it resolves whatever offset/range a
// command's Modifier+Noun implies (via the offset package) and applies the
// command's Verb to it.
func (e *Editor) execute(cmd *keyengine.ExecutableCommand) {
	count := cmd.Count
	if count < 1 {
		count = 1
	}

	switch cmd.Verb {
	case keyengine.Move:
		e.executeMove(cmd.Noun, count)
	case keyengine.Delete:
		e.executeDelete(cmd.Modifier, cmd.Noun)
	case keyengine.Undo:
		for i := 0; i < count; i++ {
			if !e.Undo() {
				break
			}
		}
	case keyengine.Redo:
		for i := 0; i < count; i++ {
			if !e.Redo() {
				break
			}
		}
	case keyengine.Replace:
		e.pendingReplace = true
	case keyengine.EnterInsert:
		e.SetMode(types.ModeInsert)
	case keyengine.EnterCommand:
		e.SetMode(types.ModeCommand)
	case keyengine.OpenBelow:
		e.openLine(true)
	case keyengine.OpenAbove:
		e.openLine(false)
	case keyengine.Hover:
		e.RequestHover()
	}
}

// resolveMotion turns a Move command's noun into a target offset, given the
// current offset to move from. ok is false for a noun this executor
// doesn't (yet) know how to resolve — a no-op, not a crash, so a
// keybinding the parser accepts but the executor hasn't implemented simply
// does nothing rather than misbehaving.
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

func (e *Editor) executeMove(noun keyengine.KeyActionName, count int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// j/k need the cursor's current Row/Col to preserve the intended
	// column across lines of different lengths — not expressible as a
	// pure function of "current byte offset" the way every other motion
	// is, so they bypass resolveMotion's generic offset-accumulation loop
	// entirely and go straight to the same Row/Col-aware logic
	// MoveCursorUp/Down use.
	switch noun {
	case keyengine.LineUp:
		for i := 0; i < count; i++ {
			e.moveCursorUpLocked()
		}
		return
	case keyengine.LineDown:
		for i := 0; i < count; i++ {
			e.moveCursorDownLocked()
		}
		return
	}

	target := e.cursor.ByteOffset
	for i := 0; i < count; i++ {
		next, ok := resolveMotion(e.table, noun, target)
		if !ok {
			return
		}
		target = next
	}
	e.moveCursorToLocked(target, piecetable.Edit{})
}

func (e *Editor) executeDelete(modifier, noun keyengine.KeyActionName) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rng, ok := resolveRange(e.table, modifier, noun, e.cursor.ByteOffset)
	if !ok {
		return
	}

	e.table.Delete(rng.Start, rng.Length)

	newOffset := rng.Start
	if newOffset > e.table.Len() {
		newOffset = e.table.Len()
	}
	e.moveCursorToLocked(newOffset, piecetable.Edit{Offset: rng.Start, OldLength: rng.Length})
}

// replaceCurrentRune implements r's second keystroke: swap exactly the
// rune under the cursor for the one just typed, leaving the cursor in
// place.
func (e *Editor) replaceCurrentRune(r rune) {
	e.mu.Lock()
	defer e.mu.Unlock()

	at := e.cursor.ByteOffset
	if at >= e.table.Len() {
		return
	}
	_, width := e.table.GetRuneAt(at)
	if width <= 0 {
		width = 1
	}
	e.table.Delete(at, width)
	newRune := []byte(string(r))
	e.table.Insert(at, newRune)
	e.moveCursorToLocked(at, piecetable.Edit{Offset: at, OldLength: width, NewLength: len(newRune)})
}

// openLine implements o (below) / O (above): insert a blank line adjacent
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
func (e *Editor) openLine(below bool) {
	e.mu.Lock()
	rng := offset.RangeLine(e.table, e.cursor.ByteOffset)
	insertAt := rng.Start
	if below {
		insertAt = rng.Start + rng.Length
	}
	indent := e.indentStringAtLocked(insertAt)
	text := indent + "\n"
	e.table.Insert(insertAt, []byte(text))
	e.moveCursorToLocked(insertAt+len(indent), piecetable.Edit{Offset: insertAt, NewLength: len(text)})
	e.mu.Unlock()

	e.SetMode(types.ModeInsert)
}

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

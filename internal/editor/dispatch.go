package editor

import (
	"fmt"

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
		e.SetMode(types.ModeNormal)
	case "<BS>":
		e.DeleteText()
	case "<Enter>":
		e.InsertText([]byte("\n"))
	default:
		if runes := []rune(key); len(runes) == 1 {
			e.InsertText([]byte(string(runes[0])))
		}
	}
	return true
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
// to the current one and switch to Insert mode with the cursor on it.
func (e *Editor) openLine(below bool) {
	e.mu.Lock()
	rng := offset.RangeLine(e.table, e.cursor.ByteOffset)
	insertAt := rng.Start
	if below {
		insertAt = rng.Start + rng.Length
	}
	e.table.Insert(insertAt, []byte("\n"))
	e.moveCursorToLocked(insertAt, piecetable.Edit{Offset: insertAt, NewLength: 1})
	e.mu.Unlock()

	e.SetMode(types.ModeInsert)
}

// moveCursorToLocked updates the cursor to newOffset (recomputing its
// screen position) and notifies listeners of edit — the delta that caused
// this move, or piecetable.Edit{} for pure cursor movement with no
// document change. Callers must already hold e.mu.
func (e *Editor) moveCursorToLocked(newOffset int, edit piecetable.Edit) {
	gridPos := offset.ByteOffsetToPosition(e.table, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)
	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange(edit)
}

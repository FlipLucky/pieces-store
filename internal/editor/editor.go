package editor

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fliplucky/pieces-store/internal/keyengine"
	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/piecetable"
	"github.com/fliplucky/pieces-store/internal/types"
	"github.com/fliplucky/pieces-store/internal/viewmanager"
)

type Editor struct {
	mu             sync.RWMutex
	table          *piecetable.Table
	virtualGrid    *viewmanager.VirtualGrid
	cursor         *Cursor
	commandContext *keyengine.CommandContext
	pendingReplace bool
	// completionLineHead is the command buffer content before the token a
	// completion cycle is replacing — only meaningful while
	// cursor.Completion.Active. Kept off Cursor itself since it's pure
	// internal bookkeeping frontends never need.
	completionLineHead string
	isQuitRequested    bool
	lastCommandError   error
	change             chan ChangeEvent
}

// ChangeEvent describes one change to the editor's state, delivered on
// ChangeChan. Edit is the precise byte-range delta when the change was a
// text edit (Insert/Delete/Undo/Redo/replace) — its zero value means the
// change was cursor movement, a mode switch, a whole-document replacement
// (:e, OpenFile), or similar: nothing for a future incremental consumer
// (e.g. a syntax tree) to re-parse against, as opposed to something to
// re-derive from scratch.
//
// Like the channel itself, this coalesces: a burst of edits arriving
// faster than a consumer drains the channel collapses to the latest one
// (see notifyChange's non-blocking send), matching today's "please
// redraw" behavior. A consumer that must see every edit exactly once, in
// order — e.g. incremental treesitter re-parsing — needs a different,
// non-coalescing mechanism; this one intentionally doesn't guarantee that.
type ChangeEvent struct {
	Edit piecetable.Edit
}

func NewEditor(initialText string) *Editor {
	ed := &Editor{
		table:          piecetable.NewPieceTable([]byte(initialText)),
		virtualGrid:    viewmanager.NewVirtualGrid(),
		cursor:         NewCursor(),
		commandContext: keyengine.CreateCommandContext(),
		change:         make(chan ChangeEvent, 1),
	}

	// Ticker simulation - only starts if SIMULATE=true is set in the environment
	if os.Getenv("SIMULATE") == "true" {
		go func() {
			chars := []byte(" Typing simulation active! Now powered by the real Piece Table under the hood.")
			idx := 0
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for range ticker.C {
				ed.mu.Lock()
				var edit piecetable.Edit
				if idx >= len(chars) {
					// Reset to initial text when simulation reaches the end
					// — a whole-buffer replacement, not an incremental
					// edit, so no delta to report (zero value).
					ed.table = piecetable.NewPieceTable([]byte(initialText))
					ed.cursor.Update(0, 0, 0)
					idx = 0
				} else {
					// Calculate total length and insert the character at the end of the buffer
					insertOffset := ed.table.Len()
					ed.table.Insert(insertOffset, []byte{chars[idx]})
					idx++

					// Update cursor position to the end of the text
					newOffset := ed.table.Len()
					gridPos := offset.ByteOffsetToPosition(ed.table, newOffset)
					screenPos := ed.virtualGrid.GetScreenPosition(gridPos)
					ed.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
					edit = piecetable.Edit{Offset: insertOffset, NewLength: 1}
				}
				ed.mu.Unlock()
				ed.notifyChange(edit)
			}
		}()
	}

	return ed
}

func NewEditorFromFile(filePath string) (*Editor, error) {
	table, err := piecetable.NewPieceTableFromFile(filePath)
	if err != nil {
		return nil, err
	}
	return &Editor{
		table:          table,
		virtualGrid:    viewmanager.NewVirtualGrid(),
		cursor:         NewCursor(),
		commandContext: keyengine.CreateCommandContext(),
		change:         make(chan ChangeEvent, 1),
	}, nil
}

func (e *Editor) GetText() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.table.CombinePieces()
}

func (e *Editor) GetFilePath() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.table.FilePath
}

func (e *Editor) SaveFile() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.table.Save()
}

func (e *Editor) SaveFileAs(filePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.table.SaveAs(filePath)
}

func (e *Editor) OpenFile(filePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	newTable, err := piecetable.NewPieceTableFromFile(filePath)
	if err != nil {
		return err
	}
	e.table = newTable
	e.cursor.Update(0, 0, 0)
	e.notifyChange(piecetable.Edit{}) // whole-document replacement, not an incremental delta
	return nil
}

func (e *Editor) GetCursor() Cursor {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return *e.cursor
}

// Viewport returns just the lines worth rendering for a window into the
// document — see viewmanager.ViewportSlice. Frontends call this instead of
// GetText() so rendering stays bounded to the visible lines (+ margin)
// regardless of document size, without exposing the piece table itself.
func (e *Editor) Viewport(topRow, visibleRows, margin int) viewmanager.Slice {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return viewmanager.ViewportSlice(e.table, topRow, visibleRows, margin)
}

func (e *Editor) ChangeChan() <-chan ChangeEvent {
	return e.change
}

// notifyChange publishes edit as the latest ChangeEvent, replacing
// whatever was already buffered rather than blocking. Non-blocking send
// alone would silently drop *this* (newest) event and leave a stale one
// behind once the buffer fills — harmless when the channel carried no
// payload (any wake-up meant the same thing: "go re-read current state"),
// but wrong now that the payload itself matters: a consumer must always
// see the latest edit, never a stale one.
func (e *Editor) notifyChange(edit piecetable.Edit) {
	event := ChangeEvent{Edit: edit}
	for {
		select {
		case e.change <- event:
			return
		default:
			select {
			case <-e.change:
			default:
			}
		}
	}
}

// --- Navigation / Vim Movement Engine ---

func (e *Editor) MoveCursorLeft() {
	e.mu.Lock()
	defer e.mu.Unlock()

	newOffset := offset.MoveLeft(e.table, e.cursor.ByteOffset)
	e.moveCursorToLocked(newOffset, piecetable.Edit{})
}

func (e *Editor) MoveCursorRight() {
	e.mu.Lock()
	defer e.mu.Unlock()

	newOffset := offset.MoveRight(e.table, e.cursor.ByteOffset)
	e.moveCursorToLocked(newOffset, piecetable.Edit{})
}

func (e *Editor) MoveCursorUp() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cursor.Row <= 0 {
		return
	}

	// No anchor passed here: PositionToByteOffset's anchor only helps when
	// the target row is at or after it (a forward scan) — Up always
	// targets a row *before* the current one, so an anchor at the current
	// position would just trip the "target row is before the anchor"
	// fallback and rescan from 0 anyway. Genuinely nothing to anchor to
	// without also tracking each line's start offset.
	targetPos := offset.Position{Row: e.cursor.Row - 1, Col: e.cursor.Col}
	newOffset := offset.PositionToByteOffset(e.table, targetPos)
	e.moveCursorToLocked(newOffset, piecetable.Edit{})
}

func (e *Editor) MoveCursorDown() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Down always targets the row right after the current one, so the
	// current position is a valid forward-scan anchor — this resumes from
	// here instead of rescanning the whole buffer from offset 0.
	targetPos := offset.Position{Row: e.cursor.Row + 1, Col: e.cursor.Col}
	newOffset := offset.PositionToByteOffset(e.table, targetPos, e.cursor.ByteOffset, e.cursor.Row)

	// If the target row doesn't actually exist (already on the last line),
	// PositionToByteOffset falls through to doc.Len() — snapping to the
	// end of the buffer instead of staying put. Detect that and no-op,
	// matching MoveCursorUp's symmetric guard.
	if actual := offset.ByteOffsetToPosition(e.table, newOffset); actual.Row <= e.cursor.Row {
		return
	}
	e.moveCursorToLocked(newOffset, piecetable.Edit{})
}

// --- Text Modification Engine ---

func (e *Editor) InsertText(data []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(data) == 0 {
		return
	}

	insertOffset := e.cursor.ByteOffset
	e.table.Insert(insertOffset, data)

	// Recalculate and update cursor position after insertion
	newOffset := insertOffset + len(data)
	e.moveCursorToLocked(newOffset, piecetable.Edit{Offset: insertOffset, NewLength: len(data)})
}

// InsertLiteralText inserts a chunk of text as-is at the cursor — for
// input that already carries real content rather than a single semantic
// keystroke (e.g. a paste or IME composition arriving as one frontend
// text-input event). Unlike HandleKey, this bypasses the key engine
// entirely: there's no vim-grammar interpretation of pasted text, only
// Insert mode's own literal-capture rule extended to more than one rune
// at a time. A no-op outside Insert mode — pasting into Normal mode isn't
// given vim-command semantics (real vim has a dedicated "paste mode"
// specifically because interpreting pasted characters as commands is
// dangerous); it's simply dropped here rather than guessed at. Frontends
// calling this never need to check the mode themselves, matching how
// HandleKey already keeps all mode-awareness inside Editor.
func (e *Editor) InsertLiteralText(text string) {
	e.mu.RLock()
	mode := e.cursor.Mode
	e.mu.RUnlock()
	if mode != types.ModeInsert {
		return
	}
	e.InsertText([]byte(text))
}

func (e *Editor) DeleteText() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cursor.ByteOffset <= 0 {
		return
	}

	newOffset := offset.MoveLeft(e.table, e.cursor.ByteOffset)
	lengthToDelete := e.cursor.ByteOffset - newOffset

	e.table.Delete(newOffset, lengthToDelete)
	e.moveCursorToLocked(newOffset, piecetable.Edit{Offset: newOffset, OldLength: lengthToDelete})
}

func (e *Editor) SetMode(m types.Mode) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cursor.SetMode(m)
	if m != types.ModeCommand {
		e.cursor.CommandBuffer = ""
		e.cursor.Completion = CompletionState{}
		e.completionLineHead = ""
	}
	e.notifyChange(piecetable.Edit{})
}

// --- Command Mode & Buffer Operations ---

var (
	ErrQuit           = errors.New("quit requested")
	ErrUnknownCommand = errors.New("unknown command")
	ErrUnsavedChanges = errors.New("unsaved changes (use :q! to discard them)")
)

func (e *Editor) AppendCommandBuffer(r rune) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cursor.CommandBuffer += string(r)
	e.notifyChange(piecetable.Edit{})
}

func (e *Editor) BackspaceCommandBuffer() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.cursor.CommandBuffer) > 0 {
		runes := []rune(e.cursor.CommandBuffer)
		e.cursor.CommandBuffer = string(runes[:len(runes)-1])
	}
	e.notifyChange(piecetable.Edit{})
}

// TriggerCompletion is <Tab>'s behavior: start a completion cycle if none
// is active, or — if one already is — confirm the currently selected
// candidate. Confirming a directory descends into it (a fresh candidate
// list computed for its contents, selection reset to the first entry, the
// popup effectively "refreshing" one level deeper); confirming a file ends
// the cycle, since there's nothing left to narrow down and the buffer
// already holds the full path. This is deliberately distinct from
// CycleCompletion (<C-n>/<C-p>), which only moves the selection within
// the current list without ever descending — matching vim's wildmenu,
// where Tab advances into a match and Ctrl-N/Ctrl-P just browse it.
func (e *Editor) TriggerCompletion() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.cursor.Completion.Active {
		e.startCompletionLocked()
		return
	}

	selected := e.cursor.Completion.Candidates[e.cursor.Completion.Index]
	if !strings.HasSuffix(selected, "/") {
		// A file: the buffer already holds it in full, nothing to descend into.
		e.cursor.Completion = CompletionState{}
		e.completionLineHead = ""
		e.notifyChange(piecetable.Edit{})
		return
	}

	// A directory: descend — the buffer already ends in exactly this
	// directory's path, so recomputing candidates for it lists its
	// contents.
	e.startCompletionLocked()
}

// CycleCompletion is <C-n>/<C-p>'s behavior: move the selection within an
// already-active completion's candidate list (direction +1/-1, wrapping
// at either end), or start a cycle if none is active yet — same as
// vim's Ctrl-N/Ctrl-P also being valid ways to trigger command-line
// completion in the first place.
func (e *Editor) CycleCompletion(direction int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.cursor.Completion.Active {
		e.startCompletionLocked()
		return
	}

	n := len(e.cursor.Completion.Candidates)
	e.cursor.Completion.Index = ((e.cursor.Completion.Index+direction)%n + n) % n
	e.cursor.CommandBuffer = e.completionLineHead + e.cursor.Completion.Candidates[e.cursor.Completion.Index]
	e.notifyChange(piecetable.Edit{})
}

// startCompletionLocked computes candidates for the command buffer's
// current final token and selects the first one, rewriting the buffer to
// match. Caller must already hold e.mu.
func (e *Editor) startCompletionLocked() {
	lineHead, candidates := completionCandidates(e.cursor.CommandBuffer)
	if len(candidates) == 0 {
		e.cursor.Completion = CompletionState{}
		e.completionLineHead = ""
		e.notifyChange(piecetable.Edit{})
		return
	}
	e.completionLineHead = lineHead
	e.cursor.Completion = CompletionState{Active: true, Candidates: candidates}
	e.cursor.CommandBuffer = lineHead + candidates[0]
	e.notifyChange(piecetable.Edit{})
}

// CancelCompletion ends an in-progress completion cycle, if any, without
// touching the command buffer's current content — the already-selected
// candidate stays, same as vim leaving wildmenu text in place once you
// type past it.
func (e *Editor) CancelCompletion() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.cursor.Completion.Active {
		return
	}
	e.cursor.Completion = CompletionState{}
	e.completionLineHead = ""
	e.notifyChange(piecetable.Edit{})
}

func (e *Editor) ClearCommandBuffer() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cursor.CommandBuffer = ""
	e.notifyChange(piecetable.Edit{})
}

func (e *Editor) ExecuteCommand(rawCmd string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	err := e.executeCommandLocked(rawCmd)
	e.lastCommandError = err
	return err
}

// GetLastCommandError returns the error (if any) from the most recently
// executed command, e.g. so a frontend can surface why :q was refused.
func (e *Editor) GetLastCommandError() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastCommandError
}

func (e *Editor) executeCommandLocked(rawCmd string) error {
	trimmed := strings.TrimSpace(rawCmd)
	if strings.HasPrefix(trimmed, ":") {
		trimmed = trimmed[1:]
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		e.cursor.SetMode(types.ModeNormal)
		e.cursor.CommandBuffer = ""
		return nil
	}

	command := parts[0]
	var err error

	switch command {
	case "w", "write":
		if len(parts) > 1 {
			err = e.table.SaveAs(parts[1])
		} else {
			err = e.table.Save()
		}
	case "e", "edit":
		if len(parts) > 1 {
			newTable, openErr := piecetable.NewPieceTableFromFile(parts[1])
			if openErr != nil {
				err = openErr
			} else {
				e.table = newTable
				e.cursor.Update(0, 0, 0)
			}
		} else {
			err = errors.New("no file specified for :e")
		}
	case "q", "quit":
		if e.table.Dirty {
			err = ErrUnsavedChanges
		} else {
			err = ErrQuit
			e.isQuitRequested = true
		}
	case "q!", "quit!":
		err = ErrQuit
		e.isQuitRequested = true
	case "wq":
		if saveErr := e.table.Save(); saveErr != nil {
			err = saveErr
		} else {
			err = ErrQuit
			e.isQuitRequested = true
		}
	default:
		err = ErrUnknownCommand
	}

	e.cursor.SetMode(types.ModeNormal)
	e.cursor.CommandBuffer = ""
	e.notifyChange(piecetable.Edit{})
	return err
}

func (e *Editor) IsQuitRequested() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isQuitRequested
}

func (e *Editor) Undo() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	edit, success := e.table.Undo()
	if success {
		newOffset := e.cursor.ByteOffset
		if newOffset > e.table.Len() {
			newOffset = e.table.Len()
		}
		e.moveCursorToLocked(newOffset, edit)
	}
	return success
}

// Redo re-applies the last edit undone by Undo. See piecetable.Table.Redo
// for the invalidation rule (a new edit clears anything left to redo).
func (e *Editor) Redo() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	edit, success := e.table.Redo()
	if success {
		newOffset := e.cursor.ByteOffset
		if newOffset > e.table.Len() {
			newOffset = e.table.Len()
		}
		e.moveCursorToLocked(newOffset, edit)
	}
	return success
}

func (e *Editor) SetModeInt(m int) {
	e.SetMode(types.Mode(m))
}

func (e *Editor) GetModeInt() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return int(e.cursor.Mode)
}

func (e *Editor) GetCommandBuffer() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cursor.CommandBuffer
}

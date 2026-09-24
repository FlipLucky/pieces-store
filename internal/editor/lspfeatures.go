// lspfeatures.go builds the actual user-facing LSP features on top of
// lsp.go's lifecycle/diagnostics wiring: hover (K), formatting (:Format),
// and autocomplete (triggered on the server's own trigger characters,
// browsed with <C-n>/<C-p>, confirmed with <Enter>, while in Insert mode).
package editor

import (
	"fmt"
	"sort"

	"github.com/fliplucky/pieces-store/internal/lspclient"
	"github.com/fliplucky/pieces-store/internal/piecetable"
)

// finishBufferMutationLocked is the common tail of every LSP-driven text
// mutation (a confirmed completion, an applied formatting edit): reparse
// for real (not incrementally — these edits didn't come from a single
// piecetable.Edit delta the highlighter's incremental path understands),
// resync the LSP server with the new full content, reposition the cursor,
// and notify. Caller must already hold e.mu.
func (e *Editor) finishBufferMutationLocked(newOffset int) {
	source := []byte(e.table.CombinePieces())
	if e.highlighter != nil {
		e.highlighter.Reparse(source)
	}
	if e.lspServer != nil {
		e.notifyLSPDidChangeLocked(source)
	}
	if newOffset > e.table.Len() {
		newOffset = e.table.Len()
	}
	e.moveCursorToLocked(newOffset, piecetable.Edit{})
}

// ---- Hover ----

// RequestHover is K's behavior in Normal mode: ask the active LSP server
// (if any) what's at the cursor, asynchronously, and show the result as a
// popup once it arrives. A silent no-op when no server is running or the
// server doesn't advertise hover support — same "a keybinding the parser
// accepts but the executor can't act on right now does nothing" precedent
// as an unresolved Move noun.
func (e *Editor) RequestHover() {
	e.mu.RLock()
	server := e.lspServer
	uri := e.lspURI
	caps := e.lspCapabilities
	at := e.cursor.ByteOffset
	var source []byte
	if server != nil {
		source = []byte(e.table.CombinePieces())
	}
	e.mu.RUnlock()

	if server == nil || !lspclient.Supported(caps.HoverProvider) {
		return
	}
	pos := lspclient.OffsetToPosition(source, at)

	e.runAsync(func() asyncApply {
		result, ok, err := server.Hover(uri, pos)
		if err != nil {
			return func(ed *Editor) { ed.setStatusMessageLocked(fmt.Sprintf("hover failed: %v", err)) }
		}
		text := ""
		if ok {
			text = result.Text()
		}
		return func(ed *Editor) {
			if ed.lspServer != server {
				return // a different buffer/server is active now — stale
			}
			ed.cursor.Hover = HoverState{Active: text != "", Text: text}
			ed.notifyChange(piecetable.Edit{})
		}
	})
}

// dismissHoverLocked clears any shown hover popup — called on essentially
// every other action (see moveCursorToLocked/SetMode) so it behaves like a
// transient tooltip rather than a state the user has to explicitly close.
func (e *Editor) dismissHoverLocked() {
	if e.cursor.Hover.Active {
		e.cursor.Hover = HoverState{}
	}
}

// ---- Formatting ----

// StartFormat is :Format's behavior, for callers that don't already hold
// e.mu (there are none in this codebase today — command dispatch goes
// through startFormatLocked directly via executeCommandLocked, which
// already holds the lock — but this stays the public entry point for
// symmetry with the rest of Editor's API and any future caller, e.g. a
// keybinding, that isn't already inside a locked method).
func (e *Editor) StartFormat() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.startFormatLocked()
}

// startFormatLocked requests textDocument/formatting from the active
// server, asynchronously, and applies the resulting edits once they
// arrive. Returns an error synchronously only for the cases known without
// asking the server at all (no server running, or it doesn't advertise
// formatting support) — the request itself, and any failure from it,
// happens on the async path and is reported via StatusMessage, matching
// :LspInstall's own pattern. Caller must already hold e.mu.
func (e *Editor) startFormatLocked() error {
	server := e.lspServer
	uri := e.lspURI
	caps := e.lspCapabilities
	baseSource := []byte(e.table.CombinePieces())

	if server == nil {
		return fmt.Errorf("no LSP server running for this buffer")
	}
	if !lspclient.Supported(caps.DocumentFormattingProvider) {
		return fmt.Errorf("the running language server doesn't support formatting")
	}

	e.setStatusMessageLocked("formatting...")

	e.runAsync(func() asyncApply {
		edits, err := server.Formatting(uri)
		if err != nil {
			return func(ed *Editor) { ed.setStatusMessageLocked(fmt.Sprintf("format failed: %v", err)) }
		}
		return func(ed *Editor) {
			if ed.lspServer != server {
				return // stale — a different buffer/server is active now
			}
			if len(edits) == 0 {
				ed.setStatusMessageLocked("already formatted")
				return
			}
			ed.applyTextEditsLocked(edits, baseSource)
			ed.setStatusMessageLocked("formatted")
		}
	})
	return nil
}

// applyTextEditsLocked applies a server's edit list against baseSource —
// the document content as it was when the request that produced edits was
// sent, needed to convert each edit's LSP Position range into a byte
// offset. Edits are applied from the end of the document backward so an
// earlier edit's offset is never invalidated by a later one already
// applied. Caller must already hold e.mu.
//
// Known, accepted imprecision: if the user kept editing while formatting's
// request/response round-trip was in flight, baseSource is stale relative
// to the live buffer and applied offsets could land slightly wrong — a
// real race in principle, not fully guarded against here (formatting is a
// user-initiated, one-off, normally-fast action; the same class of
// accepted risk as everything else in this codebase's "best-effort, not
// exhaustively race-proof" async model).
func (e *Editor) applyTextEditsLocked(edits []lspclient.TextEdit, baseSource []byte) {
	if len(edits) == 0 {
		return
	}
	sorted := append([]lspclient.TextEdit(nil), edits...)
	sort.Slice(sorted, func(i, j int) bool {
		return lspclient.PositionToOffset(baseSource, sorted[i].Range.Start) >
			lspclient.PositionToOffset(baseSource, sorted[j].Range.Start)
	})

	for _, ed := range sorted {
		start := lspclient.PositionToOffset(baseSource, ed.Range.Start)
		end := lspclient.PositionToOffset(baseSource, ed.Range.End)
		if end > start {
			e.table.Delete(start, end-start)
		}
		if len(ed.NewText) > 0 {
			e.table.Insert(start, []byte(ed.NewText))
		}
	}
	e.finishBufferMutationLocked(e.cursor.ByteOffset)
}

// ---- Autocomplete ----

// maybeTriggerLSPCompletionLocked fires an async completion request if r is
// one of the active server's advertised trigger characters — called right
// after a rune is inserted in Insert mode. A silent no-op otherwise (most
// keystrokes, or no server/no completion support at all). Caller must
// already hold e.mu (called from handleInsertModeKey's existing
// already-unlocked call site the same way insertRuneWithAutoPairing is —
// see its call site in dispatch.go).
func (e *Editor) maybeTriggerLSPCompletion(r rune) {
	e.mu.Lock()
	server := e.lspServer
	uri := e.lspURI
	caps := e.lspCapabilities
	e.mu.Unlock()

	if server == nil || caps.CompletionProvider == nil {
		return
	}
	triggered := false
	for _, ch := range caps.CompletionProvider.TriggerCharacters {
		if len(ch) > 0 && rune(ch[0]) == r {
			triggered = true
			break
		}
	}
	if !triggered {
		return
	}

	e.mu.Lock()
	e.lspCompletionGeneration++
	generation := e.lspCompletionGeneration
	at := e.cursor.ByteOffset
	source := []byte(e.table.CombinePieces())
	e.mu.Unlock()
	pos := lspclient.OffsetToPosition(source, at)

	e.runAsync(func() asyncApply {
		list, err := server.Completion(uri, pos)
		if err != nil || len(list.Items) == 0 {
			return nil
		}
		return func(ed *Editor) {
			if ed.lspServer != server || ed.lspCompletionGeneration != generation {
				return // superseded by a newer keystroke — discard
			}
			ed.cursor.LSPCompletion = LSPCompletionState{Active: true, Items: list.Items}
			ed.notifyChange(piecetable.Edit{})
		}
	})
}

// CycleLSPCompletion moves the highlighted candidate — <C-n>/<C-p> in
// Insert mode while a completion popup is active, direction +1/-1,
// wrapping at either end. A no-op if no completion is active.
func (e *Editor) CycleLSPCompletion(direction int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.cursor.LSPCompletion.Active {
		return
	}
	n := len(e.cursor.LSPCompletion.Items)
	e.cursor.LSPCompletion.Index = ((e.cursor.LSPCompletion.Index+direction)%n + n) % n
	e.notifyChange(piecetable.Edit{})
}

// LSPCompletionActive reports whether a completion popup is currently
// shown — handleInsertModeKey checks this to decide whether <Enter>
// confirms a completion or does its normal auto-indent-newline thing.
func (e *Editor) LSPCompletionActive() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cursor.LSPCompletion.Active
}

// ConfirmLSPCompletion applies the currently-highlighted candidate — a
// structured TextEdit if the server provided one (confirmed the common
// case against real gopls output), otherwise plain InsertText (falling
// back to the item's Label if even that's empty) inserted at the cursor.
func (e *Editor) ConfirmLSPCompletion() {
	e.mu.Lock()
	defer e.mu.Unlock()

	state := e.cursor.LSPCompletion
	e.cursor.LSPCompletion = LSPCompletionState{}
	if !state.Active || state.Index < 0 || state.Index >= len(state.Items) {
		return
	}
	item := state.Items[state.Index]

	source := []byte(e.table.CombinePieces())
	var start, end int
	var newText string
	if item.TextEdit != nil {
		start = lspclient.PositionToOffset(source, item.TextEdit.Range.Start)
		end = lspclient.PositionToOffset(source, item.TextEdit.Range.End)
		newText = item.TextEdit.NewText
	} else {
		newText = item.InsertText
		if newText == "" {
			newText = item.Label
		}
		start = e.cursor.ByteOffset
		end = start
	}

	if end > start {
		e.table.Delete(start, end-start)
	}
	if len(newText) > 0 {
		e.table.Insert(start, []byte(newText))
	}
	e.finishBufferMutationLocked(start + len(newText))
}

// DismissLSPCompletion cancels an in-progress completion popup without
// touching the buffer — <Esc> in Insert mode.
func (e *Editor) DismissLSPCompletion() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cursor.LSPCompletion.Active {
		e.cursor.LSPCompletion = LSPCompletionState{}
		e.notifyChange(piecetable.Edit{})
	}
}

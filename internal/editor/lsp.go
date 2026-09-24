// lsp.go wires internal/lspclient into Editor: a lazily-started server per
// buffer language (mirroring setupHighlighterLocked's own lazy-per-buffer
// pattern), diagnostics flowing into StyleSpans, and didChange kept in
// sync on every edit. Only one server is ever active at a time, matching
// Editor's own single-buffer model today — see ensureLSPForCurrentBufferLocked's
// comment for what happens when that buffer's language changes.
package editor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fliplucky/pieces-store/internal/lspclient"
	"github.com/fliplucky/pieces-store/internal/lspmanager"
	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/piecetable"
	"github.com/fliplucky/pieces-store/internal/types"
	"github.com/fliplucky/pieces-store/internal/viewmanager"
	"github.com/fliplucky/pieces-store/platform"
)

// languageID maps a types.Language to LSP's own languageId string
// (textDocument/didOpen's "languageId" field) — real values per widely-used
// LSP client convention (the same ids VS Code sends), not invented.
func languageID(lang types.Language) string {
	switch lang {
	case types.LanguageGo:
		return "go"
	case types.LanguageJavaScript:
		return "javascript"
	case types.LanguageTypeScript:
		return "typescript"
	case types.LanguageTSX:
		return "typescriptreact"
	case types.LanguageHTML:
		return "html"
	case types.LanguageCSS:
		return "css"
	case types.LanguageSCSS:
		return "scss"
	case types.LanguageJSON:
		return "json"
	case types.LanguagePHP:
		return "php"
	case types.LanguageDart:
		return "dart"
	case types.LanguageC:
		return "c"
	case types.LanguageCPP:
		return "cpp"
	case types.LanguageYAML:
		return "yaml"
	case types.LanguageDockerfile:
		return "dockerfile"
	case types.LanguageMarkdown:
		return "markdown"
	default:
		return "plaintext"
	}
}

// ensureLSPForCurrentBufferLocked (re)targets the LSP layer at the table's
// current FilePath — called everywhere setupHighlighterLocked already is
// (construction from a file, OpenFile, :e), since a new file may need a
// different language's server than whatever was already running. Three
// cases:
//   - No FilePath, or a language with no catalog entry: stop whatever was
//     running (nothing to serve an unsaved buffer or an unsupported
//     language) and stay idle.
//   - Same language as the currently-active server: reuse it — the real
//     server process stays running and indexed, just told to close the old
//     document and open the new one (didClose/didOpen on the same
//     connection), rather than paying a fresh spawn+initialize cost.
//   - A different language: stop the old server (fire-and-forget; a
//     graceful LSP shutdown isn't worth blocking the file-open path on) and
//     start a new one, asynchronously — spawning a process and completing
//     the initialize handshake is real, unbounded-latency work that has no
//     place running inline inside a locked method.
//
// Caller must already hold e.mu.
func (e *Editor) ensureLSPForCurrentBufferLocked() {
	filePath := e.table.FilePath
	lang := types.DetectLanguage(filePath)

	if filePath == "" {
		e.stopLSPServerLocked()
		return
	}
	entry, ok := lspmanager.EntryFor(lang)
	if !ok {
		e.stopLSPServerLocked()
		return
	}

	if e.lspServer != nil && e.lspLanguage == lang {
		e.reopenLSPDocumentLocked(filePath)
		return
	}

	e.stopLSPServerLocked()
	e.startLSPAsyncLocked(lang, entry, filePath)
}

// stopLSPServerLocked clears the active server, if any, and stops it on
// its own goroutine — a graceful LSP shutdown (a request/response plus a
// bounded wait, see lspclient.Server.Stop) has no place blocking whatever
// locked method triggered the switch.
func (e *Editor) stopLSPServerLocked() {
	if e.lspServer == nil {
		return
	}
	old := e.lspServer
	e.lspServer = nil
	e.lspLanguage = types.LanguagePlainText
	e.lspURI = ""
	e.lspCapabilities = lspclient.Capabilities{}
	e.diagnostics = nil
	e.cursor.Hover = HoverState{}
	e.cursor.LSPCompletion = LSPCompletionState{}
	go func() { _ = old.Stop() }()
}

// reopenLSPDocumentLocked keeps the same running server but points it at a
// newly-opened file of the same language — didClose the old document (if
// any), didOpen the new one, and reset diagnostics, which describe the
// document that's no longer open.
func (e *Editor) reopenLSPDocumentLocked(filePath string) {
	server := e.lspServer
	if e.lspURI != "" {
		_ = server.DidClose(e.lspURI)
	}
	e.lspURI = "file://" + filePath
	e.lspVersion = 1
	e.diagnostics = nil
	_ = server.DidOpen(e.lspURI, languageID(e.lspLanguage), e.lspVersion, e.table.CombinePieces())
}

// startLSPAsyncLocked resolves entry's installed binary, spawns it, and
// completes the initialize/didOpen sequence — all on a background
// goroutine via runAsync, since none of that has a bounded latency worth
// blocking a keystroke on. Caller must already hold e.mu (runAsync itself
// only touches Editor state from its returned closure, under the async
// apply goroutine's own lock).
func (e *Editor) startLSPAsyncLocked(lang types.Language, entry lspmanager.Entry, filePath string) {
	e.setStatusMessageLocked(fmt.Sprintf("starting %s...", displayPackageName(entry)))

	e.runAsync(func() asyncApply {
		binPath, err := resolveLSPBinary(entry)
		if err != nil {
			return func(ed *Editor) {
				ed.setStatusMessageLocked(fmt.Sprintf("no LSP for %s — %v", lang, err))
			}
		}

		server, err := lspclient.Start(binPath)
		if err != nil {
			return func(ed *Editor) {
				ed.setStatusMessageLocked(fmt.Sprintf("%s failed to start: %v", displayPackageName(entry), err))
			}
		}

		diagnosticsCh := make(chan lspclient.PublishDiagnosticsParams, 8)
		server.OnNotification(func(n lspclient.Notification) {
			if n.Method != "textDocument/publishDiagnostics" {
				return
			}
			if params, err := lspclient.ParsePublishDiagnostics(n.Params); err == nil {
				select {
				case diagnosticsCh <- params:
				default: // a slow consumer drops an intermediate update; the next one still arrives
				}
			}
		})

		caps, err := server.Initialize("file://" + filepath.Dir(filePath))
		if err != nil {
			return func(ed *Editor) {
				ed.setStatusMessageLocked(fmt.Sprintf("%s failed to initialize: %v", displayPackageName(entry), err))
			}
		}

		return func(ed *Editor) {
			// The buffer may have moved on (a different file opened, or
			// this server superseded by a newer switch) while the spawn
			// above was in flight — discard rather than attach a
			// now-stale server to current state.
			if ed.table.FilePath != filePath {
				go func() { _ = server.Stop() }()
				return
			}
			ed.lspServer = server
			ed.lspLanguage = lang
			ed.lspURI = "file://" + filePath
			ed.lspVersion = 1
			ed.lspCapabilities = caps
			ed.diagnostics = nil
			if err := server.DidOpen(ed.lspURI, languageID(lang), ed.lspVersion, ed.table.CombinePieces()); err != nil {
				ed.setStatusMessageLocked(fmt.Sprintf("%s: didOpen failed: %v", displayPackageName(entry), err))
				return
			}
			ed.setStatusMessageLocked(fmt.Sprintf("%s ready", displayPackageName(entry)))
			go ed.watchLSPDiagnostics(server, diagnosticsCh)
		}
	})
}

// resolveLSPBinary looks up entry's installed binary path — MethodNone
// (Dart) resolves via PATH directly since there's nothing installed to
// look up; everything else reads the persisted install index.
func resolveLSPBinary(entry lspmanager.Entry) (string, error) {
	if entry.Method == lspmanager.MethodNone {
		installed, err := lspmanager.Install("", lspmanager.PackageSpec{}, entry, nil)
		if err != nil {
			return "", err
		}
		return installed.BinPath, nil
	}
	serversDir, err := platform.LSPServersDir()
	if err != nil {
		return "", err
	}
	idx, err := lspmanager.LoadIndex(serversDir)
	if err != nil {
		return "", err
	}
	rec, ok := idx[entry.PackageName]
	if !ok {
		return "", fmt.Errorf("not installed (:LspInstall %s to add one)", strings.ToLower(entry.PackageName))
	}
	return rec.BinPath, nil
}

// watchLSPDiagnostics runs for server's whole lifetime, feeding each
// publishDiagnostics notification back into Editor through the async
// bridge (the notification itself arrives on lspclient's own read-loop
// goroutine, which must never touch Editor state directly). Returns once
// diagnosticsCh is closed — which never happens today (Editor has no
// explicit Close), matching the same "lives as long as the process" scope
// as startAsyncLoop's own drain goroutine.
func (e *Editor) watchLSPDiagnostics(server *lspclient.Server, ch <-chan lspclient.PublishDiagnosticsParams) {
	for params := range ch {
		params := params
		e.runAsync(func() asyncApply {
			return func(ed *Editor) {
				if ed.lspServer != server {
					return // superseded by a newer server — stale, discard
				}
				ed.applyLSPDiagnosticsLocked(params)
			}
		})
	}
}

// applyLSPDiagnosticsLocked converts a real publishDiagnostics payload into
// viewmanager.StyledSpans against the buffer's current content. Caller
// must already hold e.mu (it's always invoked from an asyncApply closure,
// which the async-loop goroutine calls under lock).
func (e *Editor) applyLSPDiagnosticsLocked(params lspclient.PublishDiagnosticsParams) {
	if params.URI != e.lspURI {
		return // diagnostics for a document that's no longer the active one
	}
	source := []byte(e.table.CombinePieces())
	spans := make([]viewmanager.DiagnosticSpan, 0, len(params.Diagnostics))
	for _, d := range params.Diagnostics {
		start := lspclient.PositionToOffset(source, d.Range.Start)
		end := lspclient.PositionToOffset(source, d.Range.End)
		if end <= start {
			continue
		}
		style := types.StyleDiagnosticWarning
		if d.Severity == lspclient.SeverityError {
			style = types.StyleDiagnosticError
		}
		spans = append(spans, viewmanager.DiagnosticSpan{
			TextRange: offset.TextRange{Start: start, Length: end - start},
			Style:     style,
			Message:   d.Message,
		})
	}
	e.diagnostics = spans
	e.notifyChange(piecetable.Edit{})
}

// notifyLSPDidChangeLocked sends didChange for the active server, if any —
// called from moveCursorToLocked on every real edit, synchronously and
// inline, the same pattern already established for the tree-sitter
// highlighter's own Update call right next to it: a Notify is a single
// fire-and-forget write to a local subprocess's stdin pipe, not a network
// round trip, and doing it inline keeps didChange calls strictly ordered
// (a version N notification can never race a version N+1 one sent from a
// separate goroutine). Caller must already hold e.mu.
func (e *Editor) notifyLSPDidChangeLocked(source []byte) {
	e.lspVersion++
	if err := e.lspServer.DidChange(e.lspURI, e.lspVersion, string(source)); err != nil {
		e.setStatusMessageLocked(fmt.Sprintf("LSP didChange failed: %v", err))
	}
}

# LSP stack (lspmanager + lspclient + lspservice)

Three packages, one feature arc, split cleanly by concern:

## internal/lspmanager

A real, working language-server **installer**, consuming
`mason-registry`'s live published data (`github.com/mason-org/mason-registry`)
— the same data Neovim's Mason plugin uses — rather than a hand-rolled
manifest. `catalog.go` is the curated `types.Language` → mason-registry
package table (Go→`gopls`, TypeScript/JS/TSX→`typescript-language-server`,
HTML/CSS/SCSS/JSON→`vscode-langservers-extracted`, PHP→`intelephense`,
YAML→`yaml-language-server`, C/C++→`clangd`, Markdown→`marksman`,
Dockerfile→`docker-language-server`, Dart→detected on `PATH` only).
`install.go` checks a package's declared runtime dependency (Node, Go, or
none) lazily, only for the server actually being installed. **Zero new
third-party dependencies** — everything is stdlib (`net/http`,
`archive/zip`, `archive/tar`, `os/exec`). `index.go` persists what's
actually installed so `:LspStatus` reflects real, restart-durable state.

## internal/lspclient

A hand-rolled (not a library — the real message surface, confirmed
against real `gopls` first, turned out small enough not to need one)
JSON-RPC/LSP **client**: `transport.go` (framing), `client.go`
(request/response correlation, notification dispatch), `lifecycle.go`
(spawn, handshake, bounded `Stop()`), `position.go` (byte-offset ↔ LSP
`Position`, real UTF-16 code-unit math — `gopls` defaults to UTF-16 even
when UTF-8 is offered), `sync.go` (`didOpen`/`didChange`/`didClose`/
`hover`/`completion`/`formatting` wrappers, diagnostics parsing).
`didChange` is full-sync (whole document each time) — a deliberate v1
call, accepted explicitly ("if you have a 500MB file with a complex LSP
running, you're doing something wrong").

`CompletionItem` (T49, 2026-10-18) carries `AdditionalTextEdits` alongside
its own `TextEdit` — how `gopls` (and most servers) express auto-import:
completing an unimported package member returns the completion's own edit
plus a second edit inserting the import line elsewhere in the document.
[editor](editor.md)'s `confirmLSPCompletionLocked` applies both together in
one pass, not sequentially — see that page for why sequencing them would
have been a real correctness bug.

`SemanticTokensLegend`/`SemanticTokensOptions`/`Server.SemanticTokensFull`/
`DecodeSemanticTokens` (T48, 2026-10-18) wire up
`textDocument/semanticTokens/full` — the legend (negotiated once at
`initialize`) maps each decoded token's type index to a name; `Decode`
itself is a pure function turning the wire format's delta-encoded flat
array into absolute `(line, startChar, length, tokenType)` positions, no
server needed to test it. One real, empirically-caught gotcha: `gopls`
(v0.23.0) doesn't advertise `semanticTokensProvider` at all unless
`initialize`'s `initializationOptions` includes a gopls-specific
`{"semanticTokens": true}` — a generic LSP client wouldn't know to send
this; found by getting a real "capability not advertised" skip in testing,
not assumed from documentation. See [editor](editor.md)'s
`semantictokens.go` for how the decoded result becomes `StyleSpans`.

`SignatureHelpOptions`/`Server.SignatureHelp`/`SignatureHelpResult.
ActiveLabel` (T46, 2026-10-18) wire up `textDocument/signatureHelp` —
deliberately decoding only the active signature's whole `Label` string
(e.g. `"Println(a ...any) (n int, err error)"`), not per-parameter
`Label`/`Documentation` ranges — a real, deliberate MVP scope cut matching
how Hover already shows plain normalized text. See [editor](editor.md)'s
`Cursor.SignatureHelp` for why it deliberately doesn't get dismissed the
same way Hover does.

## internal/lspservice

The **session** type — one per buffer language, mirroring
[internal/syntax](syntax.md)'s own lazy-per-buffer `Highlighter` pattern.
Given its own methods first (so `Editor` stopped touching its fields
directly), then moved to its own package once that made it
dependency-free of `Editor`'s own state.

## How it wires into Editor

One lazily-started server per buffer language, spawned/initialized
asynchronously via `Editor.runAsync` (see [editor](editor.md)'s `async.go`
entry). `didChange` is sent synchronously inline from `moveCursorToLocked`,
right next to the tree-sitter highlighter's own `Update` call — same
"local pipe write, not a network call" reasoning both times. Diagnostics
flow into `Editor.DiagnosticSpans`, tagged against the currently-open
document's URI so a notification for a since-closed buffer is discarded.

## Depends on

Nothing in this module cross-depends between the three — `lspclient` is
the protocol layer, `lspmanager` is the installer, `lspservice` is the
per-buffer session wrapping a `lspclient.Client`.

## Depended on by

[internal/editor](editor.md) (`lsp.go`), which is the only consumer of
all three.

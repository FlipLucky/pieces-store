# The boring list

Not a roadmap, not prioritized, not scoped to "what fits a hobby project" —
just a brain-dump of the unglamorous plumbing that separates a text editor
with great syntax highlighting/LSP from something a colleague on PhpStorm
could actually live in daily. Nothing here is flashy. Most of it is the
stuff you only notice when it's *missing*: the file-watch prompt, the
crash-recovery buffer, the thing that stops two windows from silently
clobbering each other's save.

`FEATURES.md` is the narrative "what exists, how it got here, where it's
headed" view. `BACKLOG.md` is known, scoped, deliberately-deferred issues.
This is neither — it's food for thought for your own weekend triage. Skim,
cross out what you don't care about, star what you do, and we can turn
whatever survives into real backlog items together.

Grouped by theme. Order within a group isn't priority.

## File & buffer handling

- [ ] **File-changed-on-disk detection** — another process (git checkout,
      prettier, a second editor window) touches the open file; today
      nothing notices, so a save silently clobbers it.
- [ ] **Crash recovery / autosave to a swap file** — vim's `.swp`,
      VS Code's hot-exit. If the editor dies mid-edit, the work shouldn't
      be gone.
- [ ] **Confirm-before-close on unsaved changes in the GUI** — `:q` already
      refuses via the command line; there's no equivalent for closing a
      tab/window with the mouse or an OS-level close button.
- [ ] **Encoding detection & conversion** — everything today assumes UTF-8.
      A Latin-1 or UTF-16 file either mangles or fails outright.
- [ ] **Line-ending handling (CRLF/LF/CR)** — no detection, no
      normalization, no per-file memory of which one a file actually uses.
- [ ] **Mixed line-ending detection within one file** — worth flagging, not
      silently "fixing" (a real correctness trap on Windows-authored repos).
- [ ] **BOM handling** — detect and strip/preserve a byte-order mark rather
      than treating it as content.
- [ ] **Binary file detection** — opening a binary file today either
      renders garbage or wastes effort syntax-highlighting noise; should
      detect and show a "binary file" placeholder instead.
- [ ] **Read-only files / permission errors** — no distinct handling for a
      file that can't be written (chmod, filesystem read-only) versus a
      generic save failure.
- [ ] **Very large file handling** — the whole-buffer `CombinePieces()`
      calls sprinkled through LSP sync/formatting materialize the entire
      document; a real multi-GB file needs a genuinely different strategy
      somewhere in that path, not just a piece table that's fast at small
      edits.
- [ ] **Very long single line ("minified" files)** — a 2MB single line
      (bundled JS, a data file) stresses per-rune rendering/highlighting
      differently than many short lines; worth a deliberate stress test.
- [ ] **Symlink handling** — save-as-new-file-replacing-symlink vs.
      write-through-symlink is a real, silent footgun in naive editors.
- [ ] **Single-instance / file-locking awareness** — two windows (or two
      processes) opening the same file today have no idea about each
      other.
- [ ] **Auto-save on a timer / on focus-loss** — distinct from crash
      recovery: an explicit, configurable "save automatically" option.
- [ ] **Local history** — a JetBrains staple: every save (not just git
      commits) recorded and diffable/revertable, independent of version
      control.
- [ ] **Trim trailing whitespace on save** (configurable) — and more
      generally, save-time normalization hooks (final newline enforcement,
      tab/space conversion).
- [ ] **`.editorconfig` support** — the de facto standard for per-project
      indent style/width/line-ending/charset that every serious editor
      reads.

## Navigation & search

- [ ] **In-buffer find** (`/`, `?` already exist as vim primitives to build
      on, but no incremental search UI exists yet) with match highlighting
      and next/prev cycling.
- [ ] **Find & replace in buffer** — regex support, case-sensitive/
      whole-word toggles, replace-one vs. replace-all, a preview of what
      will change before committing.
- [ ] **Project-wide search & replace** — already planned via an external
      ripgrep-style tool per `CLAUDE.md`'s Stage 3, but worth its own line
      item: scoped search (current folder, exclude `.gitignore`d paths,
      file-type filters), and a *safe* replace-all with a diff-style
      preview.
- [ ] **Go to file** (fuzzy filename finder — VS Code's `Ctrl-P`, a
      near-mandatory navigation primitive once a project has more than a
      handful of files).
- [ ] **Go to line / go to symbol / go to definition** — go-to-definition
      is already its own deferred plan; symbol/line jump is a smaller,
      independent piece worth splitting out rather than waiting on the
      full multi-buffer story.
- [ ] **Workspace-wide symbol search** (LSP `workspace/symbol` — "find this
      function anywhere in the project," not just the open file).
- [ ] **Find references / find usages** (LSP `textDocument/references`).
- [ ] **Document outline / breadcrumbs** — a structural view of the current
      file (functions, classes, headings for markdown) for fast navigation
      without scrolling.
- [ ] **Jumplist** (`Ctrl-O`/`Ctrl-I` in vim — back/forward through recent
      cursor positions, not just undo history).
- [ ] **Marks** (`m{a-z}` + `` `{a-z} ``) — vim's named-position primitive;
      genuinely missing today, not deferred deliberately.
- [ ] **Bracket/paren matching + jump-to-match** (`%` in vim) — distinct
      from auto-*pairing*, which exists; this is *navigating* an existing
      pair.
- [ ] **Code folding** — collapse/expand blocks, classes, functions,
      import groups.
- [ ] **Minimap or scrollbar-with-markers** — a compressed overview of the
      whole file (search matches, diagnostics, git changes) at a glance.
- [ ] **Command palette** — fuzzy-searchable list of every available
      command/action, independent of remembering a keybinding for it.
- [ ] **Recent files list**.
- [ ] **Search history** (for `/`, `?`, and `:` — vim keeps these; nothing
      here persists any input history today).

## Editing conveniences (non-LSP)

- [ ] **Multiple cursors / multi-select** — a VS Code/Sublime-era
      expectation now, even outside vim's own multi-cursor plugins.
- [ ] **Block/column visual selection** — vim's `Ctrl-V` mode; Visual mode
      as a whole is a known, tracked gap (`CLAUDE.md`), but block selection
      specifically deserves its own callout since it's a distinct
      interaction model, not just "Visual mode, but rectangular."
- [ ] **Registers** (named + numbered yank/delete registers) — vim's
      clipboard model; nothing exists yet beyond (presumably) the OS
      clipboard, if even that.
- [ ] **System clipboard integration** — copy/paste that actually round-
      trips with other applications, not just internal buffer state.
- [ ] **Macros** (`q` record / `@` replay) — a core vim power-user feature,
      not built at all yet.
- [ ] **Move line up/down, duplicate line, join lines, sort lines**.
- [ ] **Comment/uncomment toggle** (line and block, language-aware via
      tree-sitter node types or a simple per-language comment-syntax
      table).
- [ ] **Case conversion** (upper/lower/title/snake_case/camelCase) on a
      selection — a small but constantly-used convenience.
- [ ] **Smart Home/End** — toggle between column 0 and first
      non-whitespace character, matching most editors' `Home` behavior.
- [ ] **Surround** (add/change/remove a wrapping pair of
      quotes/brackets/tags around a selection or text object) — vim-
      surround is practically expected at this point.
- [ ] **Snippets** — user-defined and language-provided expansion
      templates (`for<Tab>` → a full for-loop skeleton).
- [ ] **Counted operators actually implemented** — already flagged in
      `BACKLOG.md` (`3diw` behaves like `diw` today); grouping it here too
      since it's a real, user-visible editing gap, not just an API nit.

## Language intelligence (beyond what's already shipped)

- [ ] **Rename symbol** (LSP `textDocument/rename`, workspace-wide,
      multi-file) — probably the single most-used "IDE, not editor"
      feature after go-to-definition.
- [ ] **Code actions / quick fixes** — already explicitly deferred as
      "Part 4" per `CLAUDE.md`; listed here for completeness of the set,
      not as a new idea.
- [ ] **Signature help** — parameter hints as you type inside a function
      call (LSP `textDocument/signatureHelp`).
- [ ] **Inlay hints** — inferred types / parameter names shown inline
      (LSP `textDocument/inlayHint`), a very "modern IDE" affordance.
- [ ] **Semantic highlighting** — LSP semantic tokens layered on top of
      tree-sitter's syntactic highlighting (distinguishes, e.g., a
      parameter from a local variable from a global, which pure grammar-
      based highlighting can't).
- [ ] **Organize imports / auto-import on completion** — completion exists;
      it doesn't yet add the corresponding import statement.
- [ ] **Call hierarchy / type hierarchy** (LSP `callHierarchy`/
      `typeHierarchy`) — "who calls this," "what implements this
      interface."
- [ ] **Diagnostics panel** — an aggregated, sortable list of every
      error/warning in the project, not just inline squiggles you have to
      scroll to find.
- [ ] **Multi-root / multi-server awareness** — today one server runs per
      language, globally; a real multi-buffer world needs to think about
      whether that still holds once two projects with different `gopls`
      configs are open at once.
- [ ] **LSP workspace configuration** (`workspace/configuration`,
      `didChangeConfiguration`) — servers like `gopls` support real
      per-project settings; nothing today sends any.

## Vim-completeness

(Distinct from "language intelligence" above — this is closing gaps
against vim itself, the stated baseline.)

- [ ] Visual mode (line-wise, block-wise) — tracked already, listed here
      for completeness of the theme.
- [ ] Registers, marks, macros, jumplist — all listed above too; grouped
      here as "this is what 'nvim as baseline' actually implies."
- [ ] **`.` (repeat last change)**.
- [ ] **Ex command completion/history** beyond file-path completion (which
      already exists) — command *name* completion, and a `:` history you
      can cycle through.
- [ ] **A real config file** (`.editorrc`/similar) — nothing today is
      user-configurable outside recompiling; even a minimal key-value
      config (leader key, tab width, theme) is a big usability jump.

## Version control integration

- [ ] **Git status gutter** — added/modified/deleted line markers next to
      the line numbers.
- [ ] **Inline git blame**.
- [ ] **Diff viewer** (and ideally a merge-conflict resolution UI —
      `<<<<<<<`/`=======`/`>>>>>>>` markers rendered as a real 3-way view).
- [ ] **Stage/commit/push from inside the editor** — doesn't need to be
      full `lazygit`-level, but *something* beyond shelling out manually.
- [ ] **`.gitignore`-aware file tree and search** — don't show/search
      `node_modules`, `vendor`, build output by default.

## Project & workspace management

- [ ] **A file tree / project explorer panel**.
- [ ] **File create/rename/delete/move from the UI**, with references
      updated automatically where an LSP `willRenameFiles` hook exists.
- [ ] **Tabs for open buffers** + **splits/panes** (horizontal & vertical)
      — the actual multi-buffer UI once the Stage 3 buffer model lands.
- [ ] **Session/workspace restore** — reopen the last set of files, cursor
      positions, scroll positions, and fold state on relaunch.
- [ ] **Multi-root workspaces** — more than one top-level project folder
      open at once (monorepo-adjacent workflows).
- [ ] **Per-project vs. global settings** — a `.pieces-store/` (or
      similar) project-local config layered over user-global preferences.
- [ ] **Workspace trust / safe mode** — VS Code's "do you trust this
      folder" prompt before running any project-defined tasks/config;
      relevant the moment task-runner config or a plugin API exists,
      since both are effectively arbitrary code execution on file-open.

## Terminal, build & task running

- [ ] **An embedded terminal panel**.
- [ ] **Configurable task runner** — "build," "run," "test" as
      project-defined commands (a `Taskfile`/`package.json`-scripts-style
      convention), surfaced in the command palette.
- [ ] **Output/log panel** distinct from the terminal — build output,
      language-server logs, the editor's own diagnostic log.
- [ ] **Problem matchers** — parse a build tool's error output into
      clickable, jump-to-location diagnostics (VS Code's "problem
      matcher" concept).

## Debugging

- [ ] Already called out in `CLAUDE.md` as "a genuinely separate domain,
      likely later than Stage 3" — restated here because it's the single
      biggest remaining category, not because it's newly discovered:
      breakpoints, step in/over/out, variable inspection, watch
      expressions, call stack view, conditional breakpoints, remote/
      attach-to-process debugging. Realistically means adopting the Debug
      Adapter Protocol (DAP), the debugging equivalent of LSP, rather than
      hand-rolling per-language debugger integration.

## Settings, configuration & customization

- [ ] **A real settings UI or config file** — see "a real config file"
      above; restated here as its own category since it touches
      everything else on this list (keybindings, theme, per-language
      indent rules all need somewhere to live).
- [ ] **Keybinding customization/remapping**.
- [ ] **Theme system** — today's Tokyo Night palette is hardcoded per
      frontend; a real theme system means swappable palettes, at minimum
      light/dark, ideally user-defined.
- [ ] **Font configuration** (family, size, ligatures) — GUI-specific, but
      real.
- [ ] **Per-language settings** (tab width, indent style, format-on-save
      toggle) — today's tab-based indent is global and hardcoded.

## Extensibility

- [ ] **A real plugin/scripting API** — explicitly out of scope for now
      per `CLAUDE.md`, listed here only because "boring but needed for a
      full IDE" genuinely includes it eventually; not suggesting building
      it soon.
- [ ] **User-defined snippets** (distinct from language-provided ones
      above).
- [ ] **Macro persistence** — save a recorded macro across sessions, not
      just for the current one.

## UI/UX plumbing

- [ ] **Mouse support** — click to move cursor, drag to select, scroll
      wheel, double-click word select, triple-click line select. Easy to
      forget this is "boring infra" rather than a feature, but a huge
      fraction of real-world editing muscle memory depends on it existing
      at all.
- [ ] **Native file open/save dialogs** (GUI-specific) rather than
      `:e`/`:w`'s command-line path entry as the *only* way in.
- [ ] **Context menus** (right-click → relevant actions).
- [ ] **Drag-and-drop file opening**.
- [ ] **OS file-type association** — double-clicking a `.go` file in a
      file manager opens this editor.
- [ ] **Multi-monitor / window-state persistence** (position, size,
      maximized state remembered across launches).
- [ ] **High-DPI / display-scaling awareness** (GUI-specific).

## Reliability & data safety

(Some overlap with "File & buffer handling" above — these are the
cross-cutting versions.)

- [ ] **Graceful handling of invalid UTF-8** — a corrupt or
      non-UTF-8-as-claimed file shouldn't be able to crash or silently
      corrupt the buffer.
- [ ] **Rune/grapheme-cluster correctness** — cursor movement and
      selection over multi-codepoint graphemes (combining accents, most
      emoji, some CJK) needs grapheme-aware stepping, not just UTF-8
      rune-aware stepping, to avoid splitting a visual character in two.
- [ ] **IME support** (CJK and other composed-input methods) — a distinct,
      real input-handling mode from plain keystroke capture; both Gio and
      terminal input need explicit support for it, it doesn't fall out of
      "handle key events" for free.
- [ ] **Slow-filesystem resilience** — a network drive (NFS/SMB) shouldn't
      be able to freeze the UI thread on open/save; needs the same
      "expensive work goes through the async bridge" discipline `lsp.go`
      already established, applied to file I/O too.
- [ ] **Bounded memory for many open buffers/undo history** — `BACKLOG.md`
      already flags unbounded undo-history growth per-buffer; a real
      multi-buffer world multiplies that by buffer count.

## Performance & scaling infrastructure

- [ ] **Background workspace indexing** — a real "IDE" (vs. editor) does
      project-wide symbol/reference indexing independent of any single
      open buffer; this is a different, heavier thing than the current
      per-buffer LSP session model.
- [ ] **Startup time work** — lazy-load grammars/LSP servers only for
      languages actually opened in a session, not eagerly.
- [ ] **Debounced/coalesced expensive operations at scale** — the
      coalescing `ChangeChan` pattern already exists for redraw signaling;
      the same discipline will matter again for things like live search-
      as-you-type against a large index.

## Accessibility & internationalization

- [ ] **Screen reader support** (GUI-specific — Gio's accessibility story
      is worth investigating directly, since immediate-mode GUIs
      historically lag retained-mode toolkits here).
- [ ] **High-contrast theme(s)**.
- [ ] **UI string localization** — everything user-facing is
      English-only/hardcoded today.
- [ ] **Right-to-left text support** — relevant the moment this is used
      for anything beyond code (Markdown/plain text in Arabic/Hebrew,
      say).

## Cross-platform & packaging

- [ ] **Installers/packages per platform** (`.dmg`/`.exe`/`.deb`/`.rpm`/
      Flatpak/AppImage) rather than "build the binary yourself."
- [ ] **Auto-update mechanism**.
- [ ] **Code signing / notarization** (macOS Gatekeeper, Windows
      SmartScreen — both will actively warn users away from an unsigned
      binary).
- [ ] **npm-installed LSP servers' Windows shim handling** — already a
      known, documented gap in `CLAUDE.md` (a `.cmd`/`.ps1` shim isn't
      directly `exec`-able); restated here since it's squarely in "boring
      cross-platform correctness" territory.

## Project infrastructure (not user-facing, but real)

- [ ] **CI** — `CLAUDE.md` states plainly that none exists yet; `go vet`/
      `go test` are run by hand today.
- [ ] **Cross-platform automated testing** (this project's own dev/test
      loop is presumably Linux-first; Mac/Windows are stated target
      platforms with, presumably, zero current test coverage on real
      hardware/CI runners).
- [ ] **Crash reporting** (opt-in) — for a real daily-driver product, not
      just a local dev loop.
- [ ] **A release/versioning process** — tags, changelogs, binary
      artifacts attached to releases.

---

**A note on scale, since you said not to filter for hobby-project
reasonableness**: several of the categories above (debugging via DAP,
background workspace indexing, a plugin API, full accessibility support)
are each easily as large as everything built so far, combined. That's not
a reason to avoid listing them — it's genuinely what "full IDE" costs
everywhere, including at JetBrains/Microsoft/the Neovim plugin ecosystem,
just paid for by much bigger teams. Worth keeping in view while
prioritizing: the *editor* core (Stage 1-2, largely done) and the *IDE
shell* around it (most of this document) are different-sized problems, and
it's fine — probably correct — to spend a long time being a great editor
before being a complete IDE shell.

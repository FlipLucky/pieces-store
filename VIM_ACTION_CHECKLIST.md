# What makes vim vim

A companion to `IDE_FEATURE_CHECKLIST.md`, but scoped entirely to modal
editing — the actual verb/motion/text-object vocabulary and the surrounding
machinery (registers, marks, macros, the command line, autocommands) that
makes vim/nvim feel like *vim* rather than "an editor with hjkl." Same
purpose as the last one: weekend reading, not a prioritized plan.

Checked items are confirmed working today (cross-referenced against
`FEATURES.md`'s Present section and `BACKLOG.md`); unchecked items don't
exist yet. Some checked items are checked with a caveat — read those notes,
they usually matter.

## The grammar underneath everything

This is the one section worth reading even if you skip the rest — it's the
idea `internal/keyengine` is already explicitly built around
(`CLAUDE.md`'s "action + offset" principle), so most of what follows is
really just "how big is the noun/verb table," not "does the grammar exist."

- [x] **`[count] verb [count] noun`** — e.g. `2d3w` deletes 6 words.
      Counts multiply across verb and noun. Today: counts work on bare
      motions (`3l`, `2w`); counted *operators* (`3diw`) don't yet
      (`BACKLOG.md`).
- [x] **A verb without a noun repeats itself, line-wise** (`dd`, `yy`,
      `cc`) — `dd` exists; `yy`/`cc` don't yet (yank/change don't exist as
      verbs at all yet, see Operators below).
- [x] **The same noun resolves an offset whether reached via an operator or
      pressed standalone as a motion** — `w` alone moves the cursor; `dw`
      deletes using the identical resolver. This symmetry is a real,
      already-implemented architectural decision, not just a coincidence
      of vim's own design (see `CLAUDE.md`).
- [ ] **A capital verb variant means "to end of line"** (`D` = `d$`, `C` =
      `c$`, `Y` = `y$`) — none of these exist yet since `d`/`c`/`y` as
      standalone verbs don't either.

## Modes

- [x] Normal mode
- [x] Insert mode
- [x] Command-line mode (`:`)
- [ ] **Visual mode** — charwise (`v`). Exists as a concept in
      `internal/keyengine` but genuinely unreachable — no key enters it.
- [ ] **Visual line mode** (`V`)
- [ ] **Visual block mode** (`Ctrl-V`) — column/rectangular selection;
      distinct interaction model from charwise/linewise visual, not just a
      variant.
- [ ] **Select mode** (`gh`/`gH`/`gv` variants — Windows-style "typing
      replaces selection"; genuinely obscure, low priority even by this
      document's own generous standard)
- [ ] **Replace mode** (`R` — typing overwrites instead of inserting)
- [ ] **Virtual Replace mode** (`gR` — overwrite accounting for tab
      display width; obscure)
- [ ] **Terminal mode** (nvim-specific — an embedded terminal you can enter
      Insert-mode-like typing into; depends on the terminal-panel feature
      in `IDE_FEATURE_CHECKLIST.md` existing first)
- [ ] **Command-line window** (`q:`, `q/`, `q?` — the command-line history
      opened as an editable buffer instead of a one-line prompt)

## Motions (the noun side, non-operator use)

- [x] `h` `j` `k` `l` + arrow keys
- [x] `w` `b` `e` (word forward/backward/end)
- [ ] `W` `B` `E` (WORD variants — whitespace-delimited instead of
      punctuation-aware; genuinely distinct from `w`/`b`/`e`, not the same
      thing capitalized for style)
- [ ] `ge` `gE` (backward to end of word/WORD)
- [ ] `0` `^` `$` (start of line, first non-blank, end of line)
- [ ] `gg` `G` (start/end of file, or go to line `{count}G`)
- [ ] `{` `}` (paragraph backward/forward)
- [ ] `(` `)` (sentence backward/forward)
- [ ] `%` (jump to matching bracket — flagged already in
      `IDE_FEATURE_CHECKLIST.md`; also extensible via `matchit` to
      HTML tags/`if`-`else`-`end` keyword pairs, worth knowing that's a
      whole further layer beyond bare bracket matching)
- [ ] `f{char}` `F{char}` `t{char}` `T{char}` + `;` `,` (repeat last
      find, forward/reversed) — find/till a character on the current
      line; genuinely high-frequency in real vim usage, not a nice-to-have
- [ ] `H` `M` `L` (top/middle/bottom of visible screen)
- [ ] `zt` `zz` `zb` (scroll so cursor line is at top/middle/bottom,
      without moving the cursor within the line)
- [ ] `Ctrl-E` `Ctrl-Y` (scroll one line down/up, cursor stays put)
- [ ] `Ctrl-D` `Ctrl-U` `Ctrl-F` `Ctrl-B` (scroll half-page/full-page)
- [ ] `` `{mark} `` and `'{mark}` (jump to a mark, exact position vs.
      line-start — see Marks below)
- [ ] `n` `N` (repeat last search, same/opposite direction — see Search)
- [ ] `*` `#` (search forward/backward for word under cursor)
- [ ] `gn` `gN` (select/operate on the next search match — a genuinely
      slick way to compose search with an operator, e.g. `cgn`)

## Operators (the verb side)

- [x] `d` (delete) — as a verb combining with the text-object table below.
- [ ] `c` (change — delete + enter Insert mode) — doesn't exist as its own
      verb yet; `r` (replace one character) exists as a distinct, simpler
      thing today.
- [ ] `y` (yank/copy) — doesn't exist at all; blocked on registers not
      existing yet (nothing to yank *into*).
- [ ] `p` `P` (put/paste, after/before cursor) — same blocker as `y`.
- [ ] `>` `<` (indent/dedent by operator+motion, e.g. `>ip`)
- [ ] `=` (reindent via the language's indent rules — pieces-store already
      *has* tree-based indent depth calculation for `o`/`O`/`<Enter>`; this
      would be the same machinery exposed as a standalone operator)
- [ ] `gu` `gU` `g~` (lowercase/uppercase/toggle-case over a motion)
- [ ] `gq` `gw` (format/reflow text to `textwidth`, e.g. wrapping prose —
      genuinely relevant given pieces-store's markdown-editing origin)
- [ ] `!` (filter a range of lines through an external shell command —
      obscure but real; e.g. `!ipython -c sort` piping to enact `sort`)
- [ ] `zf` (define a fold over a motion — see Folding below)

## Text objects (the other noun table)

- [x] `iw` `aw` (inner/around word)
- [x] `ip` (inner paragraph)
- [ ] `ap` (around paragraph — including trailing blank lines; `ip`
      exists, `ap` doesn't yet per `BACKLOG.md`'s own note that `Around`
      the modifier itself is registered but not every noun combination is)
- [ ] `iW` `aW` (inner/around WORD — the WORD-vs-word distinction again)
- [ ] `is` `as` (inner/around sentence)
- [ ] `i(`/`ib` `a(`/`ab`, and the `{`/`[`/`<` bracket-pair equivalents
      (inner/around a bracket pair)
- [ ] `i"` `a"` (and `'`/`` ` `` variants) — inner/around a quoted string
- [ ] `it` `at` (inner/around an HTML/XML tag)
- [ ] **Tree-sitter-powered text objects** (function, class, argument,
      conditional, loop — nvim-treesitter-textobjects' whole premise).
      Worth calling out specially: pieces-store already parses a real
      syntax tree per buffer for highlighting/indent, so `if`/`af`
      ("inner/around function") would be building on infrastructure that
      already exists, not starting from zero the way it would in stock
      vim.
- [ ] **Custom/user-defined text objects** — an extensibility point, not a
      single feature.

## Registers

- [ ] **The unnamed register** (`"`) — every yank/delete goes here by
      default; genuinely nothing exists yet, this is the most basic case.
- [ ] **Named registers** (`"a` through `"z`) — explicit, addressable
      clipboard slots.
- [ ] **Appending to a register** (`"A` — uppercase name appends instead of
      overwriting).
- [ ] **Numbered registers** (`"1`-`"9`, a rotating history of recent
      deletes; `"0` specifically always holds the last *yank*,
      un-clobbered by subsequent deletes — a real, easy-to-miss nuance).
- [ ] **Special registers**: `"%` (current filename), `"#` (alternate
      file), `".` (last inserted text), `":` (last Ex command), `"/`
      (last search pattern), `"+`/`"*` (system clipboard — see below).
- [ ] **The black-hole register** (`"_`) — delete without touching any
      other register, so a subsequent paste isn't clobbered.
- [ ] **The expression register** (`"=`) — evaluate an expression and
      insert/paste its result; a small scripting hook baked into normal
      editing.
- [ ] **System clipboard integration** (`"+`/`"*`) — already listed in
      `IDE_FEATURE_CHECKLIST.md`; restated here specifically as *which
      register* it should be, not a separate mechanism from the register
      system above.

## Marks

- [ ] **Lowercase marks `a`-`z`** — buffer-local named positions (`ma`
      sets, `` `a `` / `'a` jumps).
- [ ] **Uppercase marks `A`-`Z`** — global, cross-file positions (jumping
      to one opens the file if it isn't already open — genuinely depends
      on the multi-buffer work in `FEATURES.md`'s Stage 3).
- [ ] **Automatic marks**: `` `. `` (last change), `` `^ `` (last insert
      exit point), `` `[ ``/`` `] `` (start/end of last change or yank),
      `` `< ``/`` `> `` (last visual selection bounds), `` `` ` `` (position
      before the last jump — lets you "jump back" once).

## Jumps & change tracking

- [ ] **The jumplist** (`Ctrl-O` back / `Ctrl-I` forward) — distinct from
      undo history: tracks *navigation*, not edits.
- [ ] **The changelist** (`g;` / `g,`) — tracks positions of recent
      *edits* specifically, separate list from the jumplist.
- [ ] **`gi`** — resume Insert mode at the exact position it was last
      exited from.

## Search

- [ ] `/{pattern}` `?{pattern}` (forward/backward search)
- [ ] `n` `N` (repeat search, same/reversed direction)
- [ ] Incremental search (`incsearch` — matches highlight live while
      typing the pattern, before pressing Enter)
- [ ] Persistent match highlighting (`hlsearch`) + a clear-highlight
      command
- [ ] Search offsets (`/pattern/e`, `/pattern/+1`, etc. — land the cursor
      relative to the match, not just at its start)
- [ ] `smartcase`/`ignorecase` interplay (case-insensitive by default,
      case-sensitive the moment an uppercase letter appears in the
      pattern)
- [ ] Regex search — vim's own regex dialect (`\v`/`\V`/`\m`/`\M` "magic"
      levels control how much of vim's regex syntax needs escaping)

## The command line / Ex commands

- [ ] **Ranges** — `:5,10`, `:.,$`, `:'a,'b` (between two marks), `:%`
      (whole file), `:.+3` — addressing is itself a small composable
      language, not just "a line number."
- [ ] **`:s///` substitute**, with flags: `g` (all matches on a line, not
      just first), `c` (confirm each), `i`/`I` (case override) — already
      explicitly the planned mechanism for find/replace per `CLAUDE.md`'s
      Roadmap, not a new idea, just unbuilt.
- [ ] **`:g/pattern/command`** (global) and **`:v/pattern/command`**
      (inverse-global) — run an Ex command on every line matching (or not
      matching) a pattern; one of vim's most powerful, least
      GUI-editor-equivalent features (e.g. `:g/TODO/d` deletes every line
      containing TODO).
- [ ] **`:normal {keys}`** — run a normal-mode keystroke sequence
      programmatically, usually combined with a range or `:g` (e.g.
      `:%normal A;` appends `;` to every line).
- [ ] **`:sort`** (with `u` for unique, `i` for case-insensitive, `n` for
      numeric, `!` for reverse, and pattern-based sort keys)
- [ ] **`:!{cmd}`** (run a shell command) and **`:r !{cmd}`** (insert its
      output) and **`:{range}!{cmd}`** (filter a range of lines through an
      external command, replacing them with its output)
- [ ] **Command abbreviation/completion** — typing `:e` and pressing Tab
      completes to full command names, not just file paths (file-path
      completion already exists per `FEATURES.md`).
- [ ] **User-defined Ex commands** (`:command`) — an extensibility
      primitive.

## Macros

- [ ] **Record** (`q{register}` ... `q`) and **play** (`@{register}`,
      `@@` to repeat the last one played, `{count}@{register}` to repeat
      N times).
- [ ] **Macros are just registers** — worth calling out as a concept, not
      a separate mechanism: a recorded macro lives in a normal named
      register and can be edited as text (`"ap` to paste it, edit it,
      yank it back) — this is *why* registers have to exist before macros
      can.

## Undo, beyond linear

- [x] Linear undo/redo (`u` / `Ctrl-r`) — real, snapshot-based, with each
      entry carrying its precise edit delta (`piecetable`'s `Undo`/`Redo`).
- [ ] **The undo tree** — vim's undo isn't actually a stack: undoing, then
      making a *different* edit, doesn't discard the original redo branch;
      `g-`/`g+` walk *time*, not just one linear stack, and both branches
      stay reachable. A real, non-trivial data-structure difference from
      what exists today, not just "add more undo levels."
- [ ] `:earlier` / `:later` (time- or count-based undo-tree navigation —
      "go back 5 minutes," not just "go back 5 edits")
- [ ] Persistent undo (`undofile` — undo history survives closing and
      reopening the file)

## Dot-repeat

- [ ] **`.`** — repeat the last *change* (not motion, not yank) exactly.
      A small feature with an outsized effect on real vim fluency: most
      "vim feels fast" anecdotes trace back to composing an edit once,
      then repeating it with `.` and `n` (search) instead of re-typing or
      reaching for a mouse.

## Visual mode specifics

(All blocked on Visual mode existing at all — grouped here as "once it's
in, here's the actual surface it implies," not a flat restating of "Visual
mode: missing.")

- [ ] `o` (swap which end of the selection the cursor is anchored to)
- [ ] `gv` (reselect the last visual selection)
- [ ] Operators act directly on the selection instead of taking a further
      noun (`d`, `y`, `c`, `>`, `<`, `gu`/`gU`, `=` all apply immediately
      once something's visually selected — the one place the verb+noun
      grammar above *doesn't* apply, by design)
- [ ] Block-mode `I` / `A` (insert/append text across every line of a
      block selection simultaneously — vim's answer to multi-cursor
      column editing, listed distinctly in
      `IDE_FEATURE_CHECKLIST.md` too)
- [ ] Block-mode `r` (replace every selected character with one typed
      character, across all lines at once)
- [ ] `Ctrl-A` / `Ctrl-X` over a block selection (increment/decrement
      numbers down a column — a genuinely famous vim party trick)

## Insert mode's own mini-language

Worth its own section — Insert mode isn't just "capture keys literally,"
even in stock vim, though pieces-store's current auto-pairing/indent work
already goes beyond the naive version:

- [x] Literal character capture (baseline)
- [x] Auto-closing brackets/quotes (already built, and already goes beyond
      what stock vim does out of the box without a plugin)
- [x] Tree-based auto-indent on `<Enter>` (already built)
- [ ] `Ctrl-W` (delete back one word)
- [ ] `Ctrl-U` (delete back to start of insertion)
- [ ] `Ctrl-R {register}` (paste a register's content mid-insert, without
      leaving Insert mode)
- [ ] `Ctrl-O` (execute exactly one Normal-mode command, then return to
      Insert mode — genuinely handy for "just this once" without a full
      mode round-trip)
- [ ] `Ctrl-T` / `Ctrl-D` (indent/dedent the current line)
- [ ] `Ctrl-N` / `Ctrl-P` — already exist, but today only for path/LSP
      completion; stock vim also has plain keyword completion from the
      buffer's own words as a fallback, worth knowing that's a distinct,
      simpler thing from LSP completion.
- [ ] `Ctrl-K {digraph}` (insert a special character by two-letter code,
      e.g. `e:` for ë)
- [ ] `Ctrl-V {code}` (insert a literal character, bypassing any
      mapping/autopairing — an escape hatch that matters more once
      autopairing exists, not less)

## Windows, tabs, buffers

- [ ] **Splits** (`Ctrl-W s`/`v`, navigation `Ctrl-W h/j/k/l`, resizing) —
      the actual interaction model behind the "splits" feature named in
      `IDE_FEATURE_CHECKLIST.md`.
- [ ] **Tabs** (`:tabnew`, `gt`/`gT`) — a tab in vim is a *layout of
      splits*, not "one file" the way browser tabs work; worth keeping
      that distinction in mind when designing this rather than copying a
      browser-tab mental model.
- [ ] **Buffers** (`:ls`, `:bnext`/`:bprev`, `:bdelete`) — the actual list
      of open files, independent of how many windows are currently
      showing them; `Ctrl-^` jumps to the alternate (previously-active)
      buffer, a very high-frequency motion in real usage.

## Folding

- [ ] `zf{motion}` (create a fold), `zo`/`zc`/`za` (open/close/toggle),
      `zR`/`zM` (open/close all folds), fold methods (manual, indent-based,
      syntax-based, expression-based) — syntax/tree-sitter-based folding
      in particular is, again, building on a parse tree pieces-store
      already maintains per buffer.

## Quickfix & location lists

- [ ] **The quickfix list** — populated by `:make`, `:grep`, or a linter,
      navigated with `:cnext`/`:cprev`/`:copen` — this is the mechanism
      the "diagnostics panel" and "problem matchers" items in
      `IDE_FEATURE_CHECKLIST.md` would actually be built on in vim's own
      terms.
- [ ] **The location list** — a window-local variant of the same idea.

## Autocommands

- [ ] **Event-driven hooks** (`BufWritePre`, `BufEnter`, `InsertLeave`,
      etc.) — arguably as much "what makes vim vim" architecturally as any
      single keybinding: most of vim's own "smart" behaviors (format-on-
      save, auto-trim-whitespace, filetype-specific settings) are
      autocommands under the hood, not hardcoded editor behavior. Directly
      relevant to `CLAUDE.md`'s own noted future plugin API — autocommands
      are effectively half of what a plugin API needs to expose.

## Mappings & discoverability

- [ ] **Custom key mappings** (`:map`/`:nnoremap`/etc., and a leader key
      convention) — already implied by "keybinding customization" in
      `IDE_FEATURE_CHECKLIST.md`; the vim-specific nuance is *recursive vs.
      non-recursive* mapping (`map` vs `noremap`) and the four+ separate
      mapping namespaces (normal/insert/visual/operator-pending each map
      independently).
- [ ] **Operator-pending mode mappings** — custom text objects and custom
      motions are technically mappings into this fourth, easy-to-forget
      mode.
- [ ] **A "which-key"-style discoverability popup** — not stock vim, but
      close to mandatory in the modern nvim ecosystem specifically because
      the mapping surface above gets large fast; worth treating as part of
      "what makes *modern* nvim nvim," not just historical vim.

## Small, easy-to-forget things

- [ ] `Ctrl-A` / `Ctrl-X` on a number under the cursor (increment/decrement
      — also listed under Visual mode for the block-column version)
- [ ] `~` (toggle case of character under cursor, then advance)
- [ ] `J` (join current line with the next, smartly handling whitespace)
      / `gJ` (join without inserting a space)
- [ ] `xp` (transpose two characters — a specific, extremely common typo
      fix; not a primitive of its own, just `x` then `p`, but worth
      knowing it's a "move," not a keybinding to design for directly)
- [ ] `:iabbrev` (abbreviation expansion — typing `teh` auto-corrects to
      `the` as you keep typing; distinct from snippets, which expand on an
      explicit trigger key rather than automatically)
- [ ] `:mksession` (save/restore window layout + open buffers + cursor
      positions as one file — the vim-native version of "session/workspace
      restore" from `IDE_FEATURE_CHECKLIST.md`)
- [ ] Diff mode (`vimdiff`, or `:diffthis` on two windows) — `do`/`dp`
      (obtain/put a diff hunk between the two sides)

## What *modern* nvim adds beyond stock vim

Worth its own closing section since "nvim as baseline" (per `CLAUDE.md`'s
own vision statement) means more than "vim with a different name" —
pieces-store is already, structurally, closer to this list than to stock
vim in places:

- [x] **A real, built-in LSP client** — nvim's biggest departure from
      stock vim; pieces-store already has this, arguably more directly
      wired than nvim's own (no Lua config layer in between).
- [x] **Tree-sitter as the highlighting/indent engine**, not regex-based
      syntax files — also already true here.
- [ ] **Floating windows** — nvim's mechanism for hover docs, completion
      menus, and signature help popups; pieces-store's own hover/
      completion popups are conceptually the same idea, independently
      built rather than reusing this exact primitive (which makes sense —
      there's no Lua-plugin ecosystem here to inherit the primitive from).
- [ ] **Extmarks** — nvim's mechanism for attaching metadata to a buffer
      position that survives edits around it (used for diagnostics,
      git-gutter marks, breakpoints). Conceptually close to what
      `viewmanager.DiagnosticSpan` already does for diagnostics
      specifically; a general version of this idea is what most of the
      "gutter marker" style features in `IDE_FEATURE_CHECKLIST.md` (git
      status, bookmarks, breakpoints) would end up sharing under the hood.
- [ ] **A built-in plugin manager** (`vim.pack`, as of recent nvim
      versions) — only relevant once a plugin API exists at all.
- [ ] **`vim.ui.select`/`vim.ui.input`** — a standard hook plugins use to
      request user input/choices through whatever UI picker is installed,
      rather than each plugin rolling its own prompt; the underlying idea
      ("a UI request goes through one seam, not N different plugin-specific
      prompts") is worth keeping in mind if a plugin API is ever designed
      here.

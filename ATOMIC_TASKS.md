# Atomic Tasks

A prioritized, pick-up-anytime queue — unlike `BACKLOG.md` (a sidebar list,
not ordered) and `VIM_ACTION_CHECKLIST.md`/`IDE_FEATURE_CHECKLIST.md` (full,
deliberately unprioritized inventories for weekend reading), this file is
the actual next few things to do, each sized to finish in one sitting.
Each item has a stable ID — say "do T4" rather than re-describing it.
Where an item leans on a named project principle, it links into the wiki's
Concepts section (`docs/concepts/`) instead of re-explaining that
principle inline. Check items
off, reorder freely, delete finished sections, add new atomic items here
as they come up instead of letting prose backlogs grow instead.

**Legend** — size: `S` under 30 min, `M` 30–90 min, `L` 2h+ or multi-session.
Lane: 🔧 mechanical/solo-good, 🧠 in-reach/yours-to-drive, 🤖 good to hand off,
❓ either, pick by mood/time that day.

## 0. Clear the current stall first

Two half-finished things are sitting in the working tree right now — this
is exactly the "scaffold, then cold storage" pattern, caught early. Closing
these out removes the "where was I" friction before anything else.

- [x] **T2 — Decide `internal/keyengine/insert_mode.go`'s fate.** Resolved
      (2026-10-10, see `DESIGN_KEY_PIPELINE.md`): the stub's placement
      doesn't fit — Insert mode has no verb/mod/noun grammar to validate,
      so it was never going to finish as a `keyengine` type. Delete the
      stub; what replaces it is the real work in **T18** below, not a
      patch to this file.
- [x] **T3 — Decide the `clipboardRegister` field's fate the same way.**
      Resolved 2026-10-12 as a side effect of T10–T12: now constructed for
      real in `NewEditor`/`NewEditorFromFile`.

## 1. The key-handling pipeline redesign — done ✅

Settled 2026-10-10 as a real design, built 2026-10-11 (T18–T21, "module
1"). Final shape: `Editor.HandleKey` routes through **Validator →
Processor → Executor** (the names below solidified from the original
Validate/Interpret/Apply framing as the design conversation continued) —
see [Validator → Processor → Executor](docs/concepts/validator-processor-executor.md)
and [Decoupled by shape, not by sender](docs/concepts/decoupled-by-shape.md)
for the lasting summary; `DESIGN_KEY_PIPELINE.md` is retired to
history-only status. Kept below for the record, not as a pending list.

- [x] **T18** — Delete `internal/keyengine/insert_mode.go`'s stub. Done
      2026-10-11; `go build`/`vet`/`test` all clean after. Incidentally
      found and fixed a real, unrelated build break while verifying: the
      first checkpoint commit (`b3121bc`) left `os`/`time` imported but
      unused in `editor.go` after removing the old `SIMULATE` ticker —
      `go build ./...` had been broken since then. (S, 🔧)
- [x] **T19** — `internal/interpreter` (Processor, Normal mode) and
      `internal/executor` (Executor) built and wired in. Done 2026-10-11;
      `go build`/`vet`/`test` all clean after, including both real
      tagged `gui-local`/`tui-local` builds. Ported **every** Normal-mode
      verb in one pass (Move, Delete, Undo, Redo, Replace, EnterInsert,
      EnterCommand, OpenBelow/Above, Hover) — no dual-path left on the old
      `execute()`, which is deleted outright along with
      `executeMove`/`executeDelete`/`resolveMotion`/`resolveRange` (the
      last two relocated into `interpreter`, not rewritten). `Action`/
      `KeySequence` landed as interfaces per `DESIGN_KEY_PIPELINE.md`'s
      "Decoupled by shape, not by sender" — `interpreter`'s concrete
      Action types satisfy `executor.Action` structurally with zero
      import between the two packages; `types.ActionKind` is the one
      shared vocabulary item. Executor's `Editor` capability interface is
      satisfied by a small adapter (`lockedEditorView` in dispatch.go)
      whose methods call Editor's own existing `*Locked` bodies directly
      — reuses already-correct, already-tested locking logic
      (`undoLocked`/`redoLocked`/`setModeLocked`/`requestHoverLocked`,
      newly split out the same way `moveCursorUpLocked` already was)
      instead of Executor reimplementing it, and avoids the exact
      reentrant-mutex class of bug caught once already this session for
      `:Format`/`StartFormat`. New public `Editor.Dispatch(action)` is the
      entry point for a future producer with nothing to resolve first
      (an LSP-driven edit, a plugin call) — keypress-driven Actions go
      through `handleNormalModeKey`'s own locked block instead, since
      resolution (reading `e.table`/cursor) and application need to stay
      one atomic section, not two. `OpenLine`'s mode switch got folded
      into the same locked section as its text insert as an incidental
      atomicity improvement (previously two separate critical sections).
      All pre-existing dispatch/hjkl/undo-redo/replace/open-line tests
      pass unchanged — confirms behavioral fidelity, not just compilation.
- [x] **T20** — Insert mode's Validator+Processor built as one function,
      `interpreter.ProcessInsert` (`internal/interpreter/insert.go`). Done
      2026-10-11; `go build`/`vet`/`test` clean, both tagged builds clean,
      every pre-existing bracket/quote/backspace/indent/arrow test passes
      unchanged. Validator and Processor are folded into one function —
      Insert mode's grammar really is close to a no-op (confirmed, not
      just predicted): there's no accumulation state worth a separate
      stage. The surrounding-rune context interface the open question
      asked for is `RuneContext{Before, After rune}` — a value snapshot,
      same pattern as Normal mode's `Position` for j/k, not a live
      interface onto Editor. Real reuse, not just parallel structure:
      Insert's cursor motion and backspace-with-pair-collapse resolve
      into the *same* `MoveAction`/`DeleteAction` types T19 already built
      — Executor's handlers for those didn't need to change at all. Two
      now-dead methods caused directly by this port were deleted rather
      than left as unused API: `DismissLSPCompletion` (redundant with
      `SetMode`'s existing LSPCompletion-clearing cleanup, verified
      before cutting) and `LSPCompletionActive` (only caller was the code
      just replaced; frontends already read `cursor.LSPCompletion.Active`
      directly). `OpenLine`'s locked-adapter pattern repeated for two new
      methods (`InsertTextAtCursor`, `InsertNewlineWithIndent`) — same
      "Executor computes the indent itself, Processor never touches the
      highlighter" reasoning. `maybeTriggerLSPCompletion` (kicks off an
      async network request) deliberately stays outside the Action/
      Executor pipeline — a genuine side effect, same scope boundary
      `DESIGN_KEY_PIPELINE.md` already draws around Command mode's `:w`/
      `:q`.
- [x] **T21** — Rewire `Editor.HandleKey` to the three-stage shape and
      route `Apply`/dispatch by Action kind. Done 2026-10-11 as a side
      effect of doing T19 in full (no dual-path left to rewire later):
      `handleNormalModeKey` already calls Validator (`keyengine`) →
      Processor (`interpreter.ProcessNormal`) → Executor
      (`executor.Dispatch`) for every verb. What's left for Command mode
      stays explicitly out of scope per `DESIGN_KEY_PIPELINE.md`'s own
      scope boundary.

## 2. Structural refactor — remainder — done ✅

Done 2026-10-17. All mechanical reconnect work, same discipline as the four
pieces already done (`langdetect`, `autopairs`, `exmode`, `lspservice` — see
`BACKLOG.md` for what "done" looked like each time) and the same discipline
as every module before it this session: real current code read first,
extract, reconnect every call site, then `go build`/`vet`/`gofmt`/`test`
clean (plus both tagged `gui-local`/`tui-local` builds) before moving to the
next one. Every item here is the same underlying move: pulling logic out of
`Editor` into the type that should own it — see
[decoupling rule / conveyor-belt splitter](docs/concepts/decoupling-splitter.md)
and [the orchestrator doesn't play an instrument](docs/concepts/orchestrator-instrument.md).

- [x] **T4** — `Cursor.SetMode` now resets its own `CommandBuffer`/
      `Completion`/`LSPCompletion` fields on a mode transition, instead of
      `Editor.setModeLocked` reaching in field by field from outside.
      `Editor`-level cleanup the transition also needs
      (`completionLineHead`, dismissing hover, notifying frontends) stays
      in `setModeLocked` — hover's dismissal isn't exclusively a mode-
      transition concern (cursor movement triggers the same dismiss), so
      it wasn't folded into `Cursor.SetMode` too.
- [x] **T5** — New `CommandBuffer` type (`Append`/`Backspace`/`Clear`/
      `Set`/`String`), replacing the raw `string` field. `Set` was a
      needed addition beyond the originally-named four methods — path
      completion rewrites the buffer's entire content wholesale, not via
      append/backspace. Being a named string type under the hood, `%s`
      formatting and direct string-literal assignment/comparison (as
      existing tests already did) both kept working with zero call-site
      changes beyond the methods themselves.
- [x] **T6** — `CompletionState`/`LSPCompletionState` each got their own
      `Cycle(direction)` method (the wraparound math, kept as two separate
      methods since the types hold different candidate slices —
      `Candidates` vs. `Items`), replacing `Editor.CycleCompletion`/
      `cycleLSPCompletionLocked` computing it externally against the
      fields.
- [x] **T7** — `internal/syntax.Highlighter.IndentStringAt` now does the
      tab-formatting (`IndentAt`'s depth → a literal tab string);
      `Editor.indentStringAtLocked` is a pure passthrough.
- [x] **T8** — File-related `Editor` methods (`NewEditorFromFile`,
      `OpenFile`, `SaveFile`, `SaveFileAs`, `GetFilePath`, `Language()`)
      moved into new `internal/editor/file.go`, pure move, no logic
      change — `setupHighlighterLocked` and `:e`'s handling in
      `executeCommandLocked` deliberately stayed in `editor.go` (not
      explicitly in scope, still called fine from the new file since it's
      the same package).
- [x] **T9** — `MoveCursorLeft/Right/Up/Down` + their `*Locked` bodies
      moved into `cursor.go`, pure move, no logic/receiver change (they
      stay `*Editor` methods — they need `e.table`/`e.mu`/
      `e.moveCursorToLocked`, which only `Editor` has; this is a file-
      organization move, not a receiver-type change).

## 3. Registers & real yank/paste — done ✅

Done 2026-10-12. `internal/clipboard-register` (the user's own design) is
now wired into real keybindings — `go build`/`vet`/`test` all clean,
both tagged builds clean, new `internal/editor/clipboard_test.go` covers
yank/delete/change/paste/no-register-touch-on-replace end to end through
real `HandleKey` sequences.

- [x] **T10** — `y` wired as a new verb (`yw`/`yiw`/`yy`), storing via
      `clipboardRegister.StoreRegisterValue(value, Yank)`. New
      `interpreter.YankAction` — stores text, moves cursor to the range's
      start, never touches the buffer.
- [x] **T11** — `p`/`P` wired as new direct verbs (`Paste`/`PasteBefore`
      in `keyengine`). The feared `p`-as-`Paragraph`-noun collision never
      materialized: directs and nouns are only ever consulted in mutually
      exclusive parser states (no verb pending vs. a verb/modifier already
      pending), so `p` is both the standalone paste key *and* `dip`'s
      noun, with zero conflict. **Linewise paste, done 2026-10-11**:
      `clipboardregister.Register` now stores `Entry{Value, Linewise}`
      instead of a bare string (`StoreRegisterValue`/`GetValueFrom*`/
      `Paste` all widened accordingly — the user's own package, extended
      on their explicit request); `DeleteAction`/`YankAction` set
      `IsLinewise` from `Noun == LineMotion` (the same doubled-verb signal
      that already drove `dd`/`yy` themselves — "the processor determines
      it from the key sequence," per the user's own framing); `Paste`/
      `PasteBefore` in `ProcessNormal` branch on `clip.Linewise` and insert
      the stored text as a genuine new line (via `offset.RangeLine`) above
      or below the current line, instead of splicing charwise into it. One
      real edge case found and fixed along the way: pasting *after* the
      buffer's last line (which has no trailing `"\n"` of its own) needs a
      leading `"\n"` inserted ahead of the register value as a separator,
      or the two lines glue together — see `ProcessNormal`'s `Paste` case
      and `clipboard_test.go`'s `TestDeleteThenPasteRoundTrips` (exercises
      exactly this case) and `TestDeleteLineAndPasteBeforeMiddleOfBuffer`
      (the ordinary, non-edge case for `P`).
- [x] **T12** — `d` (and the new `c`) both store into the Delete register
      before mutating, so `dd` then `p` round-trips, now linewise-correct
      (see T11). `r` (Replace) deliberately stays out of the register
      pipeline entirely — a real, tested guarantee
      (`TestReplaceDoesNotTouchRegister`), not just an omission.
- **Bonus, not originally scoped**: `c` (Change) — `ciw`/`caw`/`cc` now
  work too, reusing `Delete`'s own modifier grammar (`Change` was already
  a declared-but-unregistered `keyengine` verb) and the existing
  `DeleteAction`/`EnterInsertAction` types unchanged. First real case of
  `ProcessNormal`/`ProcessInsert` returning more than one `Action` — their
  signature widened from `Action` to `[]Action`, applied by Executor in
  order, exactly the shape the wiki's
  [Validator → Processor → Executor](docs/concepts/validator-processor-executor.md)
  entry already described as intended, not yet exercised.

## 4. Find/replace (`:s///`) — done ✅

Done 2026-10-17, chained in directly after the structural refactor
(Section 2) per the user's own instruction, and widened on their request
to include `%s` (whole-buffer scope) from the start rather than deferring
all ranges as originally planned — the rest of the original staging
(`g` flag first, `c`/`i`/arbitrary `N,M` ranges deferred) held.

- [x] **T13** — `exmode.Command` gained `Raw` (the untokenized remainder
      after the command name — a substitute's pattern/replacement may
      contain spaces, so `Args`'s whitespace-split shape doesn't fit) and
      `WholeBuffer` (true on a leading `%`). `Parse`'s name-scanner now
      stops at the first non-letter/digit/`!` character, not just
      whitespace, so `:s/foo/bar/` parses `Name="s"` with no space needed
      — every other command always has one. New
      `ParseSubstitute(raw) (SubstituteCommand, bool)` parses the
      delimited body: whichever character immediately follows `s` is the
      delimiter (matching real vim, not hardcoded to `/`), `\<delim>`
      escapes a literal delimiter inside a segment.
- [x] **T14** — `executeCommandLocked`'s `"s", "substitute"` case calls
      new `executeSubstituteLocked` (`internal/editor/substitute.go`).
      Scope (current line vs. every line, from `WholeBuffer`) is resolved
      by splitting the buffer into lines and computing each line's
      absolute byte offset once; matches become `substituteEdit{Start,
      End, NewText}` entries applied end-to-start via
      `applySubstituteEditsLocked` — the same "sort/apply backward so an
      earlier edit's offset survives" shape `applyTextEditsLocked`
      (`:Format`) already established, kept as a separate method since
      the edit shapes genuinely differ (byte offsets here vs. LSP
      `Position`s there, which would need a pointless round-trip
      conversion to reuse the existing one). Like `:Format`'s multi-edit
      apply, N matches is N separate undo steps, not one — an existing,
      already-accepted tradeoff in this codebase, not a new one.
- [x] **T15** — `g` flag supported (`strings.Contains(flags, "g")`);
      without it, only the first match per in-scope line is replaced,
      matching real vim's own default. **A real, deliberate divergence
      from vim's own regex dialect**: patterns/replacements use Go's
      `regexp` package (RE2 syntax) — no backreferences or lookaround —
      documented in `substitute.go`'s package doc comment and
      `docs/packages/small-packages.md` rather than silently accepted (no
      new third-party dependency, consistent with this codebase's existing
      stdlib-first pattern — see `lspmanager`'s own "zero new third-party
      dependencies" precedent). One real capability that came along free:
      Go's `ReplaceAllString` already interprets `$1`/`$name` capture-group
      references in the replacement (not vim's own `\1` syntax) — see
      `TestSubstituteSupportsCaptureGroups`. `ErrPatternNotFound` (real
      vim's own E486) and `ErrInvalidSubstitute` (malformed command shape)
      are distinct, surfaced the same way `:q`'s refusal already is. See
      `internal/editor/substitute_test.go` for the full matrix (current-
      line/whole-buffer × with/without `g`, pattern-not-found, malformed
      command, invalid regex, capture groups, a custom delimiter).

## 5. Small, self-contained bug fixes (good solo confidence-builders)

- [x] **T16** — Fixed 2026-10-17: `piecetable.GetRuneAt`/`FindPieceAt`
      bypassing their own mutex. New unexported `pieceAt(offset) (int,
      int)` is the lock-free lookup (the plain, behavior-named identifier
      the earlier scoping note called for — not `findPieceAtLocked`);
      `FindPieceAt` is now a thin, self-locking (`RLock`) wrapper around
      it; `GetRuneAt` also self-locks and calls `pieceAt` directly (never
      the newly-locking `FindPieceAt`) and inlines its own length check
      rather than calling the equally self-locking `Len()` — `sync.
      RWMutex` doesn't support recursive `RLock`, so an exported,
      self-locking method must never call another one on the same
      `*Table` from inside its own locked body. **`Insert` needed the
      exact same fix in the other direction**: it already held `s.mu.
      Lock()` and called `s.FindPieceAt(offset)` — once `FindPieceAt`
      itself started taking `s.mu.RLock()`, that call would have
      deadlocked on first use (a write lock already held by the same
      goroutine blocks its own later `RLock`) — caught by tracing the
      actual call graph before shipping, the same discipline already
      documented for `:Format`/`StartFormat` and the key-handling
      pipeline. Fixed by having `Insert` call `pieceAt` directly too. The
      package-wide invariant ("every exported method is self-locking; no
      unexported helper ever locks, and none should be called except
      already-locked") is now stated once in `table.go`'s package doc
      comment, not repeated per-function, per the naming/locking
      convention this task itself originally proposed. **Verified as a
      real, not just theoretical, race**: new
      `TestConcurrentInsertAndGetRuneAt` runs `Insert`/`GetRuneAt`
      concurrently under `go test -race` — confirmed it actually fails
      with a real `WARNING: DATA RACE` against the pre-fix code (reverted
      temporarily to check, not just assumed), and passes clean against
      the fix. Whole module also verified clean under `go test -race
      ./...`, not just this one package.
- [x] **T17** — Counted delete/change/yank (`3diw`, `2dd`, ...). Done
      2026-10-11: `resolveRange` (`internal/interpreter/normal.go`) now
      takes `count` and chains `resolveRangeOnce` that many times, each
      application starting where the last one ended — matches real vim's
      own documented counted-`iw` behavior (a whitespace run between words
      counts as its own chained unit, same as a word does). A count
      running past the document's end naturally saturates rather than
      erroring, since `RangeLine`/`RangeInnerWord`/etc. already clamp to
      the document boundary on their own — no explicit bounds-check
      needed. Covers Delete/Change/Yank all at once (not just Delete as
      originally scoped) since all three share this one resolver — see
      `TestCountedDeleteInnerWord`/`TestCountedDD`/
      `TestCountedDeleteStopsAtDocumentEnd` in
      `internal/editor/dispatch_test.go`. **Verified against real vim
      docs, not just asserted**: `3diw` really does alternate word/
      whitespace as separate counted units in vim too — confirmed via
      search before claiming it, since this got pushed back on. **Distinct
      from `3dw`** (operator + bare motion, no modifier) — a genuinely
      different vim command this codebase has never implemented at all
      (confirmed: `dw` is currently a silent no-op). See new **T17b**.
- [x] **T18b** — `f`/`F` wired up for real. Done 2026-10-11: new
      `offset.MotionFindCharBackward` (F's mirror of the existing
      `MotionFindCharForward`), new `keyengine.FindCharForward`/
      `FindCharBackward` verbs registered as directs, new
      `interpreter.BeginFindCharAction`/`ResolveFindChar` (same two-
      keystroke shape as `r`'s `BeginReplaceAction`/`ReplaceRuneAction` —
      f/F's second keystroke bypasses `ProcessNormal`'s normal path
      entirely, same as r's does), new `Editor.pendingFindChar` state
      mirroring `pendingReplace`. Resolution (`interpreter.ResolveFindChar`,
      a real computation against doc/cursor state) stays out of
      `dispatch.go`'s own body, unlike r's second keystroke which needs no
      resolution at all — see `docs/concepts/orchestrator-instrument.md`.
      Deliberately **not** composable with an operator yet (`df{char}`,
      `dt{char}`) — that needs a new modifier/noun shape in `keyengine`,
      a genuinely separate piece of work, not bundled in here. Also
      doesn't support a count prefix (`3fa`) yet, since the count would
      need to be carried across the two-keystroke boundary into the
      pending state — skipped for this pass, not forgotten.
- [x] **T17b / T22** — "Short delete": `d`/`c`/`y` + a bare motion
      (`dw`/`db`/`de`/`dh`/`dl`, and `cw`/`yw`/... alongside them), no
      modifier. Done 2026-10-11, renamed T22 as the user scoped it
      uniformly across all three verbs rather than Delete alone. New
      `interpreter.resolveMotionRange` reuses `resolveMotion` (the exact
      single-step resolver plain `Move` already calls) rather than
      duplicating motion logic, folded into `resolveRange` as its
      `modifier == None && noun != LineMotion` branch. `3dw` sweeps three
      whole word-motions (not word/whitespace alternation the way counted
      `diw` does — confirmed against real vim docs, two genuinely
      different commands). `keyengine.CreateNormalModeNouns` gained
      `b`/`e`/`h`/`l` entries (previously only reachable as Move-only
      directs, so `db`/`de`/`dh`/`dl` fizzled with no resolved command at
      all).
      **Two real bugs caught by testing against actual vim semantics
      before asserting them, not just assumed**: (1) `offset.
      MotionWordBackward` never moved at all when the cursor already sat
      on a word's first character (`spanStart` resolved to the same
      position) — real vim's `b` jumps to the *previous* word in that
      case; fixed by stepping back one rune first, mirroring
      `MotionWordForward`'s own "exit the current word first" shape. (2)
      `e` is an *inclusive* motion in real vim (the landing character is
      part of the swept range) while `w`/`b`/`h` are exclusive — handled
      once, generically, via `isInclusiveMotion` in
      `resolveMotionRange` rather than special-cased per verb; `l` is
      already correct by construction (one rune swept, nothing to adjust).
      See `TestShortDeleteWord(Backward|End)`/`TestShortChangeWord`/
      `TestShortYankWord`/`TestCountedShortDeleteWord` in
      `internal/editor/dispatch_test.go`.
- [x] **T23** — `x` wired as a direct: `{Verb: Delete, Noun: CharForward}`
      in `CreateNormalModeDirects`, reusing T22's short-delete path
      end to end (no new logic needed — `x` is just `dl` reachable in one
      keystroke). `3x` deletes three characters via the same count-chained
      loop. See `TestX`/`TestCountedX`.
- [x] **T24** — Named registers (lowercase only, per the user: "i didnt
      even know vim had an uppercase named register. lowercase is
      enough"). `"{letter}` prefix parsing in `keyengine.CommandContext.
      ProcessCommand` (a one-keystroke literal capture, same shape as
      Replace/FindChar's second keystroke, works whether the register
      comes before or after a count — `"a3yw` and `3"ayw` both work).
      `keyengine.ExecutableCommand.Register rune` threads through;
      `clipboard-register`'s `namedRegister` widened from `map[rune]string`
      to `map[rune]Entry` (single-value, no history, unlike the category
      registers), `StoreRegisterValue` takes a `named rune` parameter.
      `interpreter.DeleteAction`/`YankAction` gained `Named rune` +
      `NamedRegister() rune`; `executor`'s new `hasNamedRegister`
      capability threads it into `DeleteRangeWithRegister`/`YankRange`,
      both widened with a `named rune` parameter. `"ap` reads from the
      named register specifically (`dispatch.go`'s paste-snapshot branches
      on `cmd.Register != 0` before falling back to the unnamed register's
      `Paste()`). See `TestNamedRegisterYankAndPaste`/
      `TestNamedRegisterDelete`.
- [x] **T25** — `:register`/`:registers` command + UI. Done 2026-10-11, as
      a **docked bottom panel** per the user's explicit choice ("i like
      the nvim way: a bottom buffer/modal for it") over the cursor-
      anchored hover/LSP-completion popup mechanism — reuses the exact
      docking pattern already backing the `:e`/`:w` completion popup in
      both frontends (Gio's zero-size-when-inactive `layout.Rigid`;
      tview's `flex.ResizeItem(view, 0, 0)` collapse). New
      `Cursor.RegisterOverview RegisterOverviewState{Active, Lines
      []string}` — not a real `Mode` (same reasoning as `HoverState`):
      dismissed by whatever keystroke arrives next, consumed purely as a
      dismiss signal by a new branch at the very top of
      `handleNormalModeKey` (before `pendingReplace`), rather than fed to
      `keyengine`. New `internal/editor/registers.go`:
      `openRegisterOverviewLocked` (wired into `executeCommandLocked`'s
      `"register"/"registers"` case) builds the overview per the user's
      own proposed layout — unnamed first (if set), then up to 3 of each
      of yank/delete/visual (skipped when empty), then every named
      register in letter order; `formatRegisterLine` centralizes
      display formatting (truncation, newline-collapsing) in `Editor`
      itself rather than duplicating it per frontend — a deliberate
      departure from the diagnostic-virtual-text precedent, justified
      since this is pure string logic with zero toolkit dependency, unlike
      per-frontend color mappings. New `ClipboardRegister.TopEntries`/
      `NamedEntries`/`GetValueFromNamedRegister` read methods. New
      `renderRegisterOverlay` (gui-base) / `registerView` (tui-base). See
      `TestRegisterOverview`.

## 6. Live `:s///` match highlighting — "inccommand" (2026-10-17)

Raised directly off the back of T13-T15's `:s///`/`:%s///` work ("i want
to highlight the code in the editor when i use :s and %s, like nvim").
Scoped via an explicit choice between nvim's two related "highlight while
using :s" behaviors — the live-preview-while-typing one first, the `c`
flag's own per-match confirm-prompt highlighting as a deliberate follow-up
rather than both at once. **T30 bumped ahead of section 7's T27/T28**
per the user's own real workflow ("i often change a word's part with
%s/gc, but sometimes it's part of a word I don't want to change, or of an
import — c is already a habit; %s feels incomplete without it").

- [x] **T29 — Live preview highlighting while typing `:s`/`:%s`.** Done
      2026-10-17, then corrected same day after real use surfaced a scoping
      bug. `executeSubstituteLocked`'s line-scanning logic (current-line-
      only by default, every line on `%`, first match per line unless `g`)
      was factored out into a shared `findSubstituteMatchesLocked` so
      execution and preview can't drift apart — one
      `substituteMatchLoc{Start,End,Text}` scan feeds both. New
      `Editor.SubstitutePreviewRanges() []offset.TextRange`, silent
      (returns nil, never errors) whenever the buffer isn't shaped like a
      substitute at all, or the pattern is empty/doesn't compile as RE2.
      New `viewmanager.SubstitutePreviewContains`/
      `SubstitutePreviewRangesForRange` reuse `offset.TextRange` directly
      rather than a new span type. Deliberately still scoped to
      highlighting the matches themselves, not previewing the replaced
      text in place (nvim's full `inccommand=nosplit` behavior) — see the
      Longer-horizon entry below for why that's a separate, bigger task.
      **The bug, found via real testing**: the first version required a
      *complete* `pattern/replacement/` before showing anything, so
      `:s/hel` and `:s/hello` highlighted nothing — wrong; real nvim's
      incsearch narrows the highlight live as the pattern itself is typed,
      before the closing delimiter even exists. Fixed by splitting
      `exmode.ParseSubstitute`'s strict segment logic out into a new,
      lenient `exmode.SplitSubstituteSegments` (same splitting, no minimum-
      segment-count requirement) that `ParseSubstitute` itself now calls —
      `SubstitutePreviewRangesLocked` uses the lenient form directly, so
      `s/fo` already highlights every "fo" and narrows/updates live as more
      is typed; the "g" flag only takes effect once its own segment has
      actually been typed, matching how execution reads it. Rendering
      differs structurally per frontend: `gui-base` draws every glyph
      itself, so a match gets a `withSubstitutePreviewHighlight` full-cell
      background fill layered underneath whatever the cell already draws
      (no toolkit limitation, same non-destructive layering
      `withDiagnosticUnderline` already established). `tui-base` had to
      grow a real capability it didn't have before: `openTag`'s background
      was previously always hardcoded to the editor's own bg color, so a
      match now gets its own background slot in the tag (`#515c7e`, Tokyo
      Night's own visual-selection shade, kept the same in both frontends)
      while keeping its syntax-color foreground. Verified end-to-end at
      the `Editor` level (progressive narrowing, g-flag timing,
      current-line/`%` scope, silent on invalid/incomplete input) and at
      the rendering level (`tui-base`'s `styledLineText` tests cover
      highlight-over-plain-text and highlight-over-keyword-color). (M, 🤖)
- [x] **T30 — The `c` (confirm) flag.** Done 2026-10-17. Walks each match
      one at a time, asking y/n/a/q/l before deciding. Reuses T29's
      `findSubstituteMatchesLocked` for the match list and
      `SubstitutePreviewRanges` itself for highlighting — that function now
      checks `Cursor.SubstituteConfirm.Active` first and returns just the
      current candidate's range, so **zero frontend rendering changes**
      were needed for the highlight (both frontends already called
      `SubstitutePreviewRanges` every render). New `SubstituteConfirmState
      {Active, Prompt}` on `Cursor` (same shape as `HoverState`/
      `RegisterOverview` — not a real `Mode`); the actual match list/
      regexp/replacement-template bookkeeping lives as unexported `Editor`
      fields (`substituteConfirm*`), mirroring `pendingReplace`/
      `pendingFindChar`'s own split. A real, necessary subtlety caught
      before writing any code (not after): `executeCommandLocked` resets
      `Mode` to Normal *after* the command-dispatch switch returns, so by
      the time `executeSubstituteLocked` starts a confirm session,
      `Editor.Mode` is about to be Normal again — meaning
      `handleNormalModeKey`, not `handleCommandModeKey`, is what's live for
      the next keystroke, which is exactly where the new
      `y`/`n`/`a`/`q`/`l` interception needed to go (ahead of `pendingReplace`,
      mirroring its own precedent) — confirmed by reading the actual
      control flow, not assumed. "a" (replace this and all remaining) is a
      tight loop reusing the same single-match apply+advance the "y" case
      uses; "l" applies one and ends; `q`/`<Esc>` end untouched; anything
      else is ignored outright, matching real vim's own prompt behavior.
      Each applied match shifts every *remaining* match's offsets by the
      real length delta, since the whole list was located against the
      pre-edit text. Verified end-to-end through `ExecuteCommand`/
      `HandleKey`, not just unit-level resolver tests — see
      `internal/editor/substitute_test.go`'s `TestSubstituteConfirm*`
      (14 cases), including a dedicated regression guard
      (`TestSubstituteConfirmYIsInterceptedNotYanked`) proving `y` is
      answered as "yes, replace," never falls through to the real
      Normal-mode yank verb it would otherwise collide with. Both tagged
      `gui-local`/`tui-local` builds verified clean.

## 7. UX polish pass — "works" vs. "feels nice" (2026-10-17)

Prompted directly by the user's own lived experience hitting the gap
("i need to check the bottom bar for my mode, which felt odd") plus two
more small-change/big-UX-result asks in the same message. Final confirmed
sequence (2026-10-17, after T30 was bumped ahead and T31 was added): T26
(done) → T30 (section 6, above) → **T28 → T27 → T31** (this section, in
this order).

- [x] **T26 — Mode-based cursor shape in `tui-base`.** Done 2026-10-17.
      `gui-base` already drew a block (Normal/Visual/Command) vs. pipe
      (Insert) cursor by hand-drawing the glyph itself (confirmed via a
      verification fork before starting — CLAUDE.md's prose had drifted,
      claiming both frontends had this); `tui-base` only ever drew a
      solid inverted-color block, in every mode, by stamping a
      `cursorTag`-wrapped rune into `styledLineText`'s output — no `mode`
      parameter existed anywhere in that function. Rather than extend
      that simulation with some bar-shaped approximation, switched to
      tcell's real native cursor: `tcell.Screen.SetCursorStyle(CursorStyle,
      ...Color)` (confirmed present and actually wired to the terminal
      backend in the installed `tcell v2.8.1`, not just declared on an
      interface — `tScreen.SetCursor` has a real implementation).
      `tview.Application.SetBeforeDrawFunc`/`SetAfterDrawFunc` hand the
      live `tcell.Screen` straight to a callback, so `Boot` registers an
      `AfterDrawFunc` that calls `screen.ShowCursor(x, y)` +
      `screen.SetCursorStyle(...)` every frame, using the same
      `editorView.GetInnerRect()` + cursor-row/col math already proven
      correct by `renderHoverPopup`'s exact-cell positioning. `cursorTag`
      and the cursor-cell branches in `styledLineText` are deleted
      outright (no dual-path): the function's signature shrank from 6
      params to 4 (`isCursorLine`/`cursorCol` no longer needed once the
      cursor isn't drawn as text), and `style_test.go`'s cursor-specific
      cases were removed rather than ported, since there's nothing left
      in this function for them to test — cursor positioning is now a
      `Boot`-level concern, not a `styledLineText` one. Verified:
      `go build`/`vet`/`gofmt`/`test ./...` clean, plus both tagged
      `gui-local`/`tui-local` builds. (M, 🤖)
- [ ] **T28 — Line numbers, with a relative/absolute toggle.** New gutter-
      rendering logic in both `gui-base` and `tui-base` independently (no
      shared rendering code between the two frontends today, so this is
      real duplicated work, not a one-package change) plus a toggle
      mechanism — likely a `:set relativenumber`/`:set number`-style
      command through the existing `exmode`/`executeCommandLocked` path.
      Open design question not yet resolved: where the toggle's own state
      lives (a new `Editor`/`Cursor` field most likely, since both
      frontends need to read it every render). (L, 🧠 — real design
      surface, a good one to scope together rather than hand off whole)
- [ ] **T27 — Command-name autocomplete in Command mode.** `:Lsp<Tab>` →
      cycles `LspInstall`/`LspUninstall`/`LspStatus`/etc., the same way
      `:e`/`:w` already autocomplete paths. Reuses the existing
      `CompletionState`/popup UI mechanism end to end (both frontends
      already render it) — the new part is a static command-name
      candidate source, triggered from `completionCandidates` when its
      current `idx := strings.LastIndexByte(buffer, ' '); if idx < 0`
      guard would otherwise come back empty (i.e. the user is still
      typing the command name itself, before any argument). Widened
      2026-10-17: also wire arrow-up/down and `<C-p>`/`<C-n>` to browse
      *command history* while in Command mode with an empty/unmodified
      buffer (real vim's own `:`-history recall) — **but there's no
      command-history mechanism anywhere in this codebase today**
      (confirmed via grep, not assumed), so that half is split out into
      **T32** below and deferred rather than silently expanding this
      task's scope. (M, ❓)
- [ ] **T31 — `/pattern<CR>` search, with `<C-n>`/`<C-p>` to cycle
      matches.** The missing half of "find and replace" — only replace
      exists today. The live-highlight mechanism from T29
      (`SubstitutePreviewContains`/`SubstitutePreviewRangesForRange` +
      each frontend's highlight rendering) already works and is confirmed
      good by real use ("tested it, works great!") — reuse it directly for
      search-match highlighting rather than building a second highlight
      path. Real differences from `:s///`'s scan, though, not a copy-
      paste: search scope is the *whole buffer* (not current-line-by-
      default like substitute), matches wrap around from the cursor
      position, and there's no replacement/edit at all — just navigation.
      Needs: a new `/`-prefixed entry point (the `/` key in Normal mode,
      parsed similarly to `:` but not through `exmode` — a search pattern
      isn't a command), a `Cursor`-level search state (pattern + the
      current match index, so `<C-n>`/`<C-p>` know what they're cycling)
      that `Enter` commits into, and cursor-jump-to-match on confirm/cycle.
      Deliberately scoped per the user's own explicit instruction to reuse
      `<C-n>`/`<C-p>` for next/prev match — the same two keys already mean
      "cycle" everywhere else in this codebase (path completion, LSP
      completion) — rather than real vim's own dedicated `n`/`N`, keeping
      one consistent "these two keys cycle whatever's active" convention
      instead of adding a second one. (M, ❓)

## 7b. Deferred sub-feature

- [ ] **T32 — Command-mode history.** A real command-history store (most
      recently executed `:`/`/` commands, newest-accessible-first) —
      doesn't exist anywhere in this codebase yet (confirmed via grep).
      Split out from T27 rather than silently folded in, since T27's own
      scope (command-*name* completion) is unrelated to recalling past
      full commands. Needed by T27's arrow-up/down/`<C-p>`/`<C-n>`
      history-browsing half, and plausibly reusable by T31's `/` search
      too (real vim shares one history ring per prompt type — `:` and `/`
      each get their own). Where it lives is an open design question: most
      likely a bounded ring on `Editor` (mirroring `Table.History`'s own
      "no cap yet, revisit if it matters" precedent from
      `BACKLOG.md`), populated on every real `ExecuteCommand`/confirmed
      search. (M, ❓)

## 8. Bulk Normal-mode keybinding wins (audit 2026-10-17)

The user's own framing: "look up all nvim keybindings in normal mode, and
see how much can already be built... if we can make a bulk of all the
keybindings that can be implemented with reusing what we already have, we
make a major UX improvement, without adding anything 'new'." A dedicated
fork audit re-verified `VIM_ACTION_CHECKLIST.md`'s claims against the real
current `keyengine`/`interpreter`/`offset`/`executor` code (the checklist
itself turned out badly stale — `y`/`p`/`P`, named registers, short-delete,
`x`, counted operators all show as unchecked there despite being done) and
sorted every remaining candidate into: already done (stale checkbox only),
trivial reuse, small-new-primitive, or genuinely-excluded (Visual mode,
macros, marks, undo tree, dot-repeat, viewport-relative motions, and
anything depending on T31's search existing first — these stay tracked in
`VIM_ACTION_CHECKLIST.md` itself, not duplicated here). Queued (exact
ordering against T27/T28/T30/T31/T32 above deliberately left open — "decide
order later"), grouped by shape rather than as one giant task or 20 tiny
ones:

- [ ] **T33 — Bare line-position & motion-variant keys: `0` `$` `^` `t` `T`
      `W` `B` `E`.** All table-entry-level: `0`/`$` export already-written-
      but-unexported `lineStart`/`lineEnd` (`internal/offset/motions.go`)
      as new noun identities; `^` composes `lineStart` + `FindOffset` with
      a `Not(IsWhitespace)` predicate (already exported) clamped to
      `lineEnd`; `t`/`T` are `f`/`F`'s existing `MotionFindCharForward`/
      `Backward` shape with a one-step-back adjustment; `W`/`B`/`E` are
      `MotionWordForward`/`Backward`/`End` with `IsWordChar` swapped for
      `Not(IsWhitespace)` — near-1:1 parametrization of functions that
      already exist, 3 keybindings almost free. (L — many small pieces
      bundled, 🔧 mechanical/solo-good)
- [ ] **T34 — Edit-operator variants: `X` `a` `s` `S` `Y` `D` `C` `I`
      `A`.** All Verb/EnterInsert composition, no new `ActionKind`: `X` =
      `resolveMotionRange`'s existing backward-landing branch + a direct
      entry; `a` = Move(CharForward) + EnterInsert; `s`/`S` = `cl`/`cc`
      exactly (Change already composes Delete+EnterInsert); `Y` = `yy`
      (real vim/nvim default, not `y$`); `D`/`C` reuse T33's `$`
      noun identity once it lands; `I`/`A` reuse T33's `^`/`$` + EnterInsert
      once those land (real dependency — sequence T33 before this). (L,
      🔧 mechanical/solo-good)
- [ ] **T35 — Case-toggle & line-join primitives: `~` `gu` `gU` `g~` `J`
      `gJ`.** `~` = read-rune (`GetRuneAt` already exists) + Delete(1) +
      Insert(toggled) + Move, all existing `ActionKind`s. `gu`/`gU`/`g~`
      generalize `~` over `resolveRange`'s already-generic range
      resolution (build once, reuse for all three case directions). `J`/
      `gJ` need one genuinely new `offset` function (join point + smart-
      whitespace replacement text) but reuse existing Delete+Insert to
      apply it — same shape as every other primitive here, not a new
      architectural concept. (M, 🔧/🧠 mixed — `~` is mechanical, the `g`-
      prefixed range generalization is a real small design)
- [ ] **T36 — Navigation primitives: `gg` `G` (bare) `{` `}` `;` `,`
      `gi`.** `gg`/`G` reuse existing `PositionToByteOffset`/
      `ByteOffsetToPosition` math but need special-casing in
      `resolveMove`'s generic count-repeat loop (`{count}G` means "go to
      line N," not "repeat N times" — same kind of exception
      `resolveLineStep` already carves out for `j`/`k`). Bare `{`/`}`
      reuse `RangeInnerParagraph`'s existing `prevLineSpan`/`nextLineSpan`/
      `isBlankSpan` blank-detection directly, just as a Move instead of a
      Range. `;`/`,` (repeat last `f`/`F`/`t`/`T`) need one new `Cursor`
      field (last target rune + direction) — same shape as the existing
      `pendingReplace`/`pendingFindChar` fields. `gi` (resume insert at
      last exit position) needs one new `Cursor` field, trivial once
      tracked. (M, ❓)
- [ ] **T37 — Bracket/quote text objects: `i(`/`a(`, `i{`/`a{`, `i"`/`a"`
      (+ their other bracket/quote equivalents).** Same resolver *shape*
      as `RangeInnerWord`/`RangeAroundWord`, but each needs real bracket-
      matching or quote-scanning logic (not just a predicate swap the way
      `W`/`B`/`E` were) — a genuinely new primitive per text-object family,
      though all the same small kind of thing `offset` already does. (M
      per family — size the bracket family and the quote family
      separately when picked up, don't treat as one task)
- [ ] **T38 — Indent/dedent/reindent operators: `>` `<` `=`.** Reuses
      `resolveRange` for the target range and `IndentStringAt` for `='s
      depth, but needs a new multi-line batch-apply method — real
      precedent already exists for this exact shape (`:Format`'s
      `applyTextEditsLocked`, `:s///`'s `applySubstituteEditsLocked`), so
      this is "write the third instance of an established pattern," not
      a new one. (M, ❓)

## 9. `:h`/`:help` + a scrollable keybinding overview (2026-10-17)

Raised directly off the T33-T38 audit above — more keybindings landing in
bulk makes "what's actually bound" worth surfacing *now*, not deferred,
per the user's own framing ("better maintenance... should be added now,
instead of in the future"). Scoped deliberately small for a first pass: a
scrollable list, one line per keybinding, a ~5-word description each — no
real help-topic prose yet (that's explicitly future work layered onto the
same `:h` entry point later, e.g. `:h daw` for a full explanation).

- [ ] **T39 — `:h`/`:help` command + scrollable keybinding-overview UI.**
      The real design decision, worth getting right once rather than
      iterating: **generate the overview from the `keyengine` tables
      themselves** (adding a short `Description` field to each table
      entry/`DirectAction`), not a hand-maintained parallel list — the
      exact staleness failure this task's own audit just found in
      `VIM_ACTION_CHECKLIST.md` is precisely what a second, separately-
      maintained list would reproduce. New UI shape, not a reuse of
      `RegisterOverview`'s mechanism: that one is "any key dismisses,"
      but a scrollable list needs real scroll-key handling (`j`/`k`/
      arrows/`<C-d>`/`<C-u>`) before an explicit close key (`q`/`<Esc>`) —
      closer to a small, real mini-mode than a passive overlay. Likely a
      full-viewport overlay in both frontends rather than a small docked
      panel, since the list is long: `tui-base` can reuse `tview.Pages`
      (already used for the hover popup) with a real scrollable
      `tview.TextView`; `gui-base` can use Gio's `layout.List`, which
      natively supports scrolling — built for exactly this, unlike
      anything currently in `gui-base`'s own rendering. (M/L, 🧠 — real
      interaction + data-source design, worth scoping deliberately)

## Longer horizon — browse, don't start yet

- **Visual mode** — real design work (operators act directly on a
  selection, no following noun — a genuinely different shape from
  Normal's verb+noun grammar). Flagged previously as your carve-out to
  design/drive, not a default delegate. Its `StoreSelection` Target+Action
  (see `DESIGN_KEY_PIPELINE.md`) is now at least sketched, not designed.
- **Live `:s///` replacement-text preview** ("inccommand=nosplit" —
  showing the word actually turn into its replacement in the buffer as
  you type, not just a highlight on the original). Deliberately not high
  priority — "the highlight is already amazing" — and deliberately not
  folded into T29: today's rendering model everywhere in this codebase
  assumes on-screen text is literally the buffer's own bytes (styles/
  diagnostics are colors keyed by real byte offset); this needs a second,
  purely visual code path that splices replacement text into a line for
  display only, then re-derives that line's column math so the cursor and
  every other per-rune overlay still land correctly against a line that's
  temporarily a different length than the real buffer. Real, not
  impossible — roughly line-numbers-sized (T28) — but a genuine new
  rendering mechanism, not a quick follow-on to T29's highlight. Mutating
  the real buffer live and undoing it per keystroke was considered and
  ruled out: it would spam undo history, tree-sitter re-parses, and LSP
  `didChange` for edits that aren't real yet.
- **Regex support for `:s///`'s dialect itself** — raised while discussing
  the above, framed as "somewhere down the line," low priority. Needs
  disambiguating when picked up: `:s///` already uses real regex (Go's
  `regexp`/RE2 — see `BACKLOG.md`'s `:s///`-dialect entry and
  `docs/packages/small-packages.md`'s `exmode` entry), so this may mean
  closer vim-regex-dialect parity (backreferences, lookaround, `\d`/`\w`-
  style shorthands, `\1` instead of Go's `$1`) rather than "add regex" from
  scratch — confirm which before scoping real work here.
- Search, macros, marks, splits/multi-buffer, folding, autocommands — see
  `VIM_ACTION_CHECKLIST.md` and `IDE_FEATURE_CHECKLIST.md` for the full
  unprioritized inventory; `FEATURES.md`'s Stage 3+ for sequencing.

---

**How to use this**: pick from the top unless something below jumps out.
If you've got 20–30 minutes and your head's elsewhere, grab anything
tagged 🔧 — no design context to reload. If you want the "mine to build"
feeling, 🧠 items are picked specifically for that. Anything else, say
"do T<n>" and I'll pick up from there.

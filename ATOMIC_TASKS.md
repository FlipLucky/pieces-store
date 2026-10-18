# Atomic Tasks

A prioritized, pick-up-anytime queue — unlike `BACKLOG.md` (a sidebar list,
not ordered) and `FEATURES.md`'s Future section (the staged, deliberately
unprioritized roadmap — the stable plan of record this queue pulls from),
this file is the actual next few things to do, each sized to finish in one
sitting.
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
(done) → T30 (section 6, above) → T28 (done, below) → T27 (done, below) →
**T31** (this section, next).

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
- [x] **T28 — Line numbers, with a relative/absolute toggle.** Done
      2026-10-17. Modeled on real vim's own two *independent* options,
      `number`/`relativenumber` — not a single three-way toggle — a design
      choice backed by a real signal, not just vim-parity-for-its-own-sake:
      a background audit of the user's own nvim dotfiles found both set
      together, confirming the "hybrid" gutter (current line shows its
      absolute number, every other line shows relative distance) is a
      genuine daily habit, not a hypothetical. New `internal/viewmanager/
      gutter.go` holds the pure, dependency-free rendering math:
      `GutterWidth(highestVisibleRow)` sizes the column off the *currently
      visible* highest row (not a full-document line-count scan), matching
      this codebase's existing viewport-bounded-cost principle
      (`ViewportSlice`/`FindPieceAt`'s own deferred-scan precedent) — the
      tradeoff is the gutter can grow a digit as a user scrolls deep into a
      huge file, accepted deliberately rather than paying for an upfront
      full scan. `GutterText(row, cursorRow, showNumber, relative, width)`
      returns one row's padded cell, empty when both options are off so a
      caller skips the column entirely. One real bug caught by its own test
      before shipping (not by the user): the current-line left-aligned
      special case fired whenever `row == cursorRow` regardless of
      `relative`, so plain `:set number` (no relativenumber) incorrectly
      left-aligned just the current line instead of right-aligning every
      row uniformly — fixed by gating that branch on `relative &&
      row == cursorRow`. Toggle state lives as two unexported `Editor`
      fields (`showLineNumbers`/`relativeLineNumbers`, see `editor.go`) set
      via a new minimal `:set` command (`internal/editor/settings.go`'s
      `executeSetLocked` — `number`/`nu`, `nonumber`/`nonu`,
      `relativenumber`/`rnu`, `norelativenumber`/`nornu`) reusing
      `exmode.Parse`'s existing `Args` tokenization with zero `exmode`
      changes needed; `Editor.LineNumberSettings()` exposes the on/off
      state read-only. Wired into both frontends independently (no shared
      rendering code between them, confirmed real duplicated work, not a
      one-package change): `tui-base` prepends each line's gutter text as a
      muted-color tag inside `renderContent`'s per-line loop; `gui-base`
      prepends a gutter `material.Label` via a new Horizontal
      `layout.Flex` wrapper in `renderLine` (falling back to the old
      single-widget fast path when the gutter and diagnostic virtual-text
      are both absent, so a plain line pays no new Flex overhead). Critical
      follow-on fix in both frontends, caught by tracing the column math
      rather than discovered live: the gutter shifts where column 0 of the
      cursor's own text actually starts on screen, so `tui-base`'s
      `cursorScreenCol` (feeding T26's native-cursor positioning) and both
      frontends' hover/completion-popup `x` calculation
      (`renderHoverPopup`) each gained `+ gutterWidth` (character-count in
      `tui-base`, `gutterWidth * charWidth` pixels in `gui-base`) — without
      this, turning line numbers on would have silently misplaced the
      terminal cursor and the LSP popup out from under the real text.
      Verified via `internal/viewmanager/gutter_test.go` (5 cases) and
      `internal/editor/settings_test.go` (6 cases covering toggle-on/off,
      the independent-combination "hybrid" case, and unknown-option/no-
      argument errors); `go build`/`vet`/`gofmt`/`test ./...` all clean,
      plus both tagged `gui-local`/`tui-local` builds. (L, 🧠)
- [x] **T27 — Command-name autocomplete in Command mode.** Done
      2026-10-17. `:Lsp<Tab>` cycles `LspInstall`/`LspStatus`/
      `LspUninstall`; `<C-n>`/`<C-p>` browse the same list without
      descending — reuses the existing `CompletionState`/popup UI
      mechanism end to end, zero frontend changes (both already render
      it). The only new part was a static command-name candidate source:
      `complete.go`'s `completionCandidates` previously returned `("",
      nil)` whenever `strings.LastIndexByte(buffer, ' ') < 0` (still
      typing the command name itself, no argument started yet) — that
      branch now calls a new `commandNameCandidates(prefix)` against a
      hand-maintained `commandNames` list (one literal per
      `executeCommandLocked` case, its longest/clearest spelling — kept in
      sync by hand, not derived via reflection, same reasoning `exmode`
      already assumes of its own callers). Case-sensitive prefix match,
      same as the command names themselves (`LspInstall` vs. `install`);
      an empty buffer matches every command, matching real vim's own bare
      `:<Tab>` behavior. `TriggerCompletion`'s existing directory-descend
      check (`strings.HasSuffix(selected, "/")`) needed no changes — a
      command name never ends in `/`, so confirming one correctly just
      ends the cycle. One real, deliberate behavior change this surfaced:
      `TestTriggerCompletionNoSpaceIsNoop` (`:q<Tab>` used to be a no-op)
      no longer holds now that "no space yet" means "completing a command
      name" — replaced with
      `TestTriggerCompletionCompletesCommandName`/`CommandNameNoMatchIsNoop`/
      `EmptyBufferListsAllCommands` rather than left stale. Widened
      2026-10-17 to also note arrow-up/down and `<C-p>`/`<C-n>` browsing
      *command history* on an empty/unmodified buffer (real vim's own
      `:`-history recall) — **but there's no command-history mechanism
      anywhere in this codebase today** (confirmed via grep, not assumed),
      so that half stays split out into **T32** below, deferred rather
      than silently expanding this task's scope. Verified via
      `internal/editor/complete_test.go`; `go build`/`vet`/`gofmt`/
      `test ./...` clean, plus both tagged `gui-local`/`tui-local` builds.
      (M, ❓)
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

## 7c. Worklog / quick-capture + mode-based markdown rendering (born from an org-mode brainstorm, 2026-10-17)

Not "port org-mode" — the user doesn't want org-mode's own plugin or its
programmability (capture-template DSL, agenda/TODO-state queries across
files), just the one capability it happens to satisfy: frictionless
logging/notes, sometimes project-scoped, sometimes not, without breaking
flow. Markdown is "good enough, maybe json" — deliberately not a new data
model. Placed right after T31/T32 (T31 stays next, already agreed) and
ahead of section 8/9's bulk wins — "fairly high," per the user's own
framing, but not jumping the already-committed next step. An actual
calendar/agenda view (real org-mode territory) is explicitly deferred to a
later stage, not scoped here.

- [ ] **T40 — Checkbox toggle (`- [ ]` ↔ `- [x]`).** Small, mechanical: a
      new Normal-mode action flipping the checkbox marker on the current
      line, reusing existing line-range/text-replace primitives — no new
      mechanism. Open question: which key — nothing in the existing
      `keyengine` tables is free and obviously right for this, and this
      codebase has no `<leader>`-style user-remappable prefix concept the
      way nvim does, so the binding needs a deliberate pick, not a
      default. (S, 🔧)
- [ ] **T41 — Quick-capture / worklog command (`:Log`).** Composes
      existing primitives rather than inventing a subsystem: open (or
      create) a target markdown file, append a timestamped heading, jump
      to end, drop into Insert — all through machinery that already
      exists (`:e`'s file-load path, `piecetable.Insert`, mode switch).
      Open design question: one global target (a new
      `platform.NotesDir()`-style convention, mirroring the existing
      `CacheDir`/`ConfigDir` precedent) vs. also a project-local variant
      (`:Log` vs. `:Log local` → `./WORKLOG.md`) — "sometimes project-
      based, sometimes not" suggests both eventually; whether both ship
      together or global-first is the user's call.
      **Real gap surfaced while scoping this, independent of T41 itself**:
      `executeCommandLocked`'s `"e", "edit"` case unconditionally
      overwrites `e.table` with the newly-opened file — no dirty-check at
      all, unlike `:q`'s explicit `ErrUnsavedChanges` refusal. T41 would
      walk straight into this (capture a quick note mid-edit → silently
      discards unsaved work in the real file) — worth fixing on its own
      before or alongside T41, not papering over it. (M, 🧠)
- [ ] **T42 — Mode-based markdown rendering (raw in Insert, rendered in
      Normal/Visual/Command).** The big one — a real new rendering
      mechanism, not a quick extension of `StyledSpan`. Also resolves
      Stage 2's still-open "is preview a narrow special-cased pane or the
      first real multi-pane" question from a third direction nobody had
      on the table: neither — mode-conditional rendering of the *same*
      buffer, no separate pane at all. Three real sub-decisions once this
      starts (not resolved here, flagged for when it's picked up):
      (1) confirm markdown's treesitter highlight-query capture names
      actually cover what's needed (heading level, emphasis, links) —
      `internal/syntax` already parses markdown for indent, but its
      *coloring* coverage is unverified; (2) a new span type that can
      *hide* raw bytes and substitute display-only text, not just color
      them — same "new type, don't fold into the general one" precedent
      as `DiagnosticSpan`; (3) the exact column-math problem T28 just
      solved reappears here — concealing raw markup changes a line's
      on-screen width vs. its byte length, so cursor positioning and
      hover-popup placement in both frontends need the same class of fix
      `gutterWidth` just required, except harder (the gutter was one
      constant per-line offset; conceal width varies per span). Expect
      this to split into its own multi-task arc once started, the way
      `:s///` (T13-T15) and the pipeline (T18-T21) did. (L, 🧠 — yours to
      drive, real design surface)

Deferred, not a task (per the user's own "that's a feature for later"):
since this codebase owns its own treesitter integration rather than
borrowing someone else's grammar bundle, a worklog entry's structured
metadata (JSON frontmatter) and prose body (markdown) could theoretically
render through one combined view. Just a forward-pointer — no scoping yet.

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
`FEATURES.md`'s Future section now, since `VIM_ACTION_CHECKLIST.md` itself
was merged there and deleted 2026-10-17). Queued (exact
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
      `VIM_ACTION_CHECKLIST.md` (since merged into `FEATURES.md` and
      deleted) is precisely what a second, separately-maintained list
      would reproduce. New UI shape, not a reuse of
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

## 10. Stage 2 completion — treesitter/LSP extensions (2026-10-18)

Closes out `FEATURES.md`'s Stage 2 list (everything there except **T42**,
which stays its own separate arc — markdown rendering needs a genuinely new
rendering mechanism, not an extension of infra that already works, the
real dividing line used to scope this batch). Worked in this order —
quick wins first, the item needing real investigation last:

- [x] **T51 — Format-on-save, behind a `:set` toggle.** Done 2026-10-18.
      Not always-on — "not everyone wants it": a new `formatOnSave bool`
      alongside T28's `number`/`relativenumber` precedent
      (`internal/editor/settings.go`'s `:set formatonsave`/
      `:set noformatonsave`, `FormatOnSave()` accessor). The real design
      problem this task actually turned on, caught before writing any code
      rather than discovered as a bug later: formatting is an *asynchronous*
      LSP round trip (`startFormatLocked` kicks off a request and returns
      immediately; the edits apply later via `runAsync`'s callback), so
      naively calling "format, then save" in sequence inside `:w` would
      have saved the file's *pre-formatted* content — the save would run
      before the async edits ever arrived. New `executeWriteLocked`
      (`internal/editor/lspfeatures.go`, next to `startFormatLocked`) is
      the real fix: when the toggle is off, or no server is running/capable
      of formatting, it's a plain synchronous save (identical to before —
      a missing/incapable server never blocks the save itself). When the
      toggle is on and a capable server exists, the save itself moves
      *inside* the format request's async callback — `saveLocked` (a new
      small helper factoring out the existing Save/SaveAs choice) runs only
      after the formatted edits have actually been applied, reporting
      `"formatted and saved"` via `StatusMessage` (or `"saved (format
      failed: ...)"` if the format request itself errored — a failed format
      still saves the user's real edits rather than blocking the save).
      `:wq` deliberately does **not** route through this — composing
      "format, then save, then quit" correctly would mean deferring
      `isQuitRequested` into the same async callback too (today's quit
      check happens immediately after `HandleKey`/`ExecuteCommand`
      returns, so quitting before an in-flight async save completes would
      silently lose it) — flagged explicitly as a real, deliberate scope
      narrowing in a code comment rather than left as an unexplained
      inconsistency between `:w` and `:wq`. Deliberately in-memory-only,
      same as every other `:set` option today — real persistence waits on
      the "real config file" item already flagged in `FEATURES.md`'s
      Cross-cutting section as its own dependency hub. Verified two ways:
      fast unit tests for the toggle and the no-server fallback path
      (`internal/editor/settings_test.go`), and a real end-to-end test
      against actual `gopls` (`TestRealFormatOnSaveEndToEnd`,
      `LSPMANAGER_INTEGRATION=1`) proving the save genuinely waits for the
      formatted content rather than racing ahead of it — the one way a
      naive implementation would have silently shipped the bug this task
      was designed around. The existing real-gopls test's setup was
      factored into a shared `setupRealGoplsEditor` helper rather than
      duplicated for the second test. `go build`/`vet`/`gofmt`/
      `test ./...` all clean, plus both tagged `gui-local`/`tui-local`
      builds. (S, 🔧)
- [x] **T49 — Organize-imports / auto-import on completion.** Done
      2026-10-18. Confirmed `additionalTextEdits` was genuinely absent from
      `lspclient.CompletionItem` — not dropped downstream, never decoded at
      all. Added the field (`sync.go`) and wired it through
      `confirmLSPCompletionLocked`: the completion's own `TextEdit` plus
      every `AdditionalTextEdit` apply together in one pass (both are byte
      ranges against the same pre-edit `source`, so applying them
      separately — primary first, then additional — would have let an
      earlier-in-the-document import insertion invalidate the primary
      edit's own already-computed offset). `applyTextEditsLocked`'s apply
      loop was split out into a reusable `applyTextEditsRawLocked` (no
      change in behavior for `:Format`'s own call site) specifically so
      completion could reuse the existing sorted end-to-start apply logic
      without inheriting `applyTextEditsLocked`'s "land the cursor at its
      own pre-edit position" assumption — completion needs the cursor at
      the end of the *inserted* text instead, shifted by the net length
      delta of every additional edit landing earlier in the document (an
      import line is almost always above the completed symbol, but the
      math doesn't assume that — it sums deltas of edits whose start
      offset is ≤ the primary edit's own start). Verified at both tiers:
      two fast unit tests
      (`TestConfirmLSPCompletionAppliesAdditionalTextEdits`,
      `TestConfirmLSPCompletionWithoutAdditionalTextEditsUnchanged` in the
      new `lspfeatures_test.go`) exercise the offset math directly with a
      hand-built `CompletionItem`, and a real end-to-end test
      (`TestRealAutoImportCompletionEndToEnd`, `LSPMANAGER_INTEGRATION=1`)
      types `strings.` with no existing import against a real `gopls`,
      confirms a real `ToUpper` candidate actually carries a nonempty
      `AdditionalTextEdits` (not assumed — logged the real response), and
      confirms the buffer after applying it contains both the inserted
      `import "strings"` and `strings.ToUpper` in the right places.
      `go build`/`vet`/`gofmt`/`test ./...` all clean, plus both tagged
      `gui-local`/`tui-local` builds. (S/M, 🔧)
- [x] **T44 — Smarter `%` bracket matching.** Done 2026-10-18. `%` didn't
      exist at all before this (confirmed, not assumed) — this is the
      whole motion's first version, "smart" from the start rather than a
      dumb version upgraded later. New `offset.MatchBracket` (`bracket.go`)
      is a pure depth-counting scanner over `offset.Document` — zero
      dependency on styling, matching the package's existing zero-
      dependency contract — that takes a `skip func(offset int) bool`
      predicate so a caller can exclude offsets that shouldn't count as
      real brackets at all (a `{`/`}` inside a string or comment) without
      `offset` ever needing to know what a "string" is. Wired as a new
      `BracketMatch` noun (`keyengine`'s `%` direct action, `Move` verb —
      the same silent-motion shape `w`/`b`/`e`/`h`/`l` already use) and a
      new `interpreter.BracketContext{Skip}` snapshot parameter on
      `ProcessNormal`/`resolveMove`, threaded the same way
      `PasteClipboard` already is: Editor builds the real predicate once
      under its own lock (`bracketMatchSkipLocked`, new `editor/bracket.go`)
      from `StyleString`/`StyleComment` spans and hands it in, so
      `interpreter` itself never touches `Highlighter`/`StyledSpan`
      directly. Deliberately bounded to a ±20,000-byte window around the
      cursor, not the whole document — reusing `Highlighter.Spans`
      unbounded would reintroduce exactly the O(n)-per-keystroke class of
      bug the 2026-09-21 scrolling fix exists to prevent; a match farther
      away than that falls back to plain unstyled depth-counting, an
      honest documented boundary rather than a silent wrong answer.
      Deliberate non-goal, matching `f`/`F`'s own precedent of flagging
      scope rather than silently omitting it: real vim's `N%` (jump to N
      percent of the file) is a different feature sharing the same key,
      not implemented here. Verified at both levels: `offset/bracket_test.go`
      covers the pure scanner (simple pair, nested same/different types,
      unbalanced, non-bracket position, the skip predicate itself), and
      `editor/bracket_test.go` drives the real `HandleKey("%")` path,
      including a real tree-sitter-backed test
      (`TestBracketMatchSkipsBracketInsideGoString`) that opens an actual
      `.go` file with a `}` inside a string literal and confirms `%` lands
      on the real closing brace, not the fake one — no `LSPMANAGER_INTEGRATION`
      gating needed, since this only exercises `internal/syntax`'s local
      parse, not a spawned LSP server. `go build`/`vet`/`gofmt`/
      `test ./...` all clean, plus both tagged `gui-local`/`tui-local`
      builds. (M, 🔧)
- [x] **T48 — Semantic highlighting.** Done 2026-10-18.
      `textDocument/semanticTokens/full`, merged into `StyleSpans` — listed
      *first*, ahead of tree-sitter's own spans, so `viewmanager.StyleAt`'s
      already-documented "first match wins" rule gives semantic tokens
      priority at any byte both cover, with zero span-splitting and zero
      new rendering code in either frontend (confirmed: both already apply
      `StyleSpans` the same way they always have). Real, if minor, gotcha
      caught empirically rather than assumed: real `gopls` (v0.23.0)
      doesn't advertise `semanticTokensProvider` at all unless told to via
      a gopls-specific `initializationOptions: {"semanticTokens": true}`
      on `initialize` — confirmed by first getting a real "this gopls
      build doesn't advertise semanticTokensProvider" skip, then fixing it
      and re-running the same test to confirm real token data actually
      arrives. New `lspclient` wiring: `SemanticTokensLegend`/
      `SemanticTokensOptions` on `Capabilities`, `Server.SemanticTokensFull`,
      and a pure `DecodeSemanticTokens` expanding the wire format's
      delta-encoded flat array (`deltaLine`/`deltaStartChar`/`length`/
      `tokenType`/`modifiers` quintuples) into absolute positions — the
      decode logic itself needed no server at all to test.
      `internal/lspservice` gained a `SemanticTokensGeneration` counter,
      the same staleness-guard shape `CompletionGeneration` already
      established.

      The real design decision wasn't the protocol plumbing, it was
      **when to ask**: this is a whole-document request by design (no
      `semanticTokens/range` wiring, out of scope here), so firing it on
      every keystroke would mean one full-document request per character
      typed — directly against CLAUDE.md's "no unbounded per-keystroke LSP
      cost" principle. Solved with a simple in-flight throttle
      (`semanticTokensInFlight`): at most one outstanding request at a
      time, a burst of edits while one's in flight fires no new ones, and
      the first edit after it resolves is free to fire the next — the same
      "coalesces to the latest" shape `ChangeChan`'s own doc comment
      already describes for a different channel. Separately, and more
      importantly: an edit clears `semanticTokens` to `nil` *immediately*
      (synchronously, in `moveCursorToLocked`, not just "soon" via the
      async response) — the old spans' byte offsets are computed against
      the pre-edit document and would point at the *wrong* text once the
      buffer shifts, not just outdated text, so falling back to plain
      tree-sitter coloring in the gap is the correct behavior, not a
      compromise. New `editor/semantictokens.go` holds
      `requestSemanticTokensLocked`/`decodeSemanticStyleSpans`/
      `semanticStyleFor` (the small subset of the standard LSP token-type
      vocabulary this codebase's own `types.Style` enum can actually
      represent — everything else is dropped, not forced into a
      wrong-looking color). Verified at both tiers: fast unit tests for
      the decoder, the type-to-`Style` mapping (including the
      drop-unknown-types and out-of-range-index cases), the in-flight
      no-op guard, and the semantic-overrides-syntax merge contract
      (`TestStyleSpansSemanticTokensOverrideSyntaxAtSameByte`, using a real
      tree-sitter-parsed `.go` file); and a real end-to-end test
      (`TestRealSemanticTokensEndToEnd`, `LSPMANAGER_INTEGRATION=1`)
      confirming real token data arrives from real `gopls`, lands in
      `StyleSpans` correctly, and gets cleared immediately on the next
      edit. `go build`/`vet`/`gofmt`/`test ./...` all clean, plus both
      tagged `gui-local`/`tui-local` builds. (M, 🧠)
- [x] **T46 — Signature help.** Done 2026-10-18. `textDocument/
      signatureHelp`, triggered on the server's own advertised
      `signatureHelpProvider.triggerCharacters` (real `gopls`: `(`/`,`) —
      `maybeTriggerSignatureHelp` is a near-direct copy of
      `maybeTriggerLSPCompletion`'s own trigger-character detection
      shape. `Cursor.SignatureHelp` literally reuses `HoverState` (not a
      parallel type) — a genuine sibling state: one new `case` in each
      frontend's existing `renderHoverPopup` switch was the entire
      rendering change needed, confirmed zero new UI primitive required as
      scoped. New `lspclient` wiring: `SignatureHelpOptions` on
      `Capabilities`, `Server.SignatureHelp`, `SignatureHelpResult.
      ActiveLabel()` — deliberately decodes only `Signatures[].Label`
      (skips `Parameters`/per-parameter highlighting entirely), so the
      popup shows the whole active signature as plain text, the same way
      Hover already shows plain text — a real, deliberate MVP scope cut,
      not an oversight.

      One real, deliberate behavior difference from Hover, not an
      inconsistency: Hover is dismissed by any cursor movement
      (`dismissHoverLocked`), but `SignatureHelp` is **not** — it's
      Insert-mode-only state (cleared in `Cursor.SetMode` alongside
      `LSPCompletion`, the same way that's already scoped), and
      deliberately stays visible through ordinary typing within the same
      call rather than vanishing the instant the next character moves the
      cursor, which would make the feature useless (you'd see the
      signature for one frame and then nothing). The real cost of this
      choice, documented rather than silently accepted: it can go stale
      if you move on to a different, unrelated call without pressing a new
      trigger character — real editors solve this with
      `retriggerCharacters`/continuous re-querying while a signature stays
      open, deliberately out of scope here (same reasoning diagnostics'
      own known linear-scan gap gets — a real, lower-priority version of a
      solved problem, not unnoticed).

      Verified at both tiers: fast unit tests for `ActiveLabel`'s
      active-index/empty/out-of-range cases, the no-server guard, and the
      "persists through plain cursor movement but clears on leaving Insert
      mode" contract; a real end-to-end test
      (`TestRealSignatureHelpEndToEnd`, `LSPMANAGER_INTEGRATION=1`) types
      `fmt.Println(` against real `gopls`, confirms the real signature
      text arrives, confirms a cursor move does NOT dismiss it, and
      confirms leaving Insert mode does. `go build`/`vet`/`gofmt`/
      `test ./...` all clean, plus both tagged `gui-local`/`tui-local`
      builds. (M, 🧠)
- [ ] **T43 — Treesitter-powered text objects** (`if`/`af`, `ic`/`ac`,
      parameter objects). **Scope investigation done 2026-10-18 — confirmed
      the bigger-scope case, not the small one.** Read `gotreesitter`
      v0.53.0's own `grammars` package source directly (`registry.go`):
      `LangEntry` carries `HighlightQuery` and `TagsQuery` fields, each
      populated per language — but **no `TextObjectsQuery` field or
      bundled `textobjects.scm`-convention query exists anywhere in the
      module** (confirmed by reading the actual struct definition, not
      just grepping for `.scm` files, which found none at all — the
      library embeds its query strings as Go string literals, not loose
      files, so a filename search alone would have been misleading).
      `gotreesitter.NewQuery(source, lang)` — the same general-purpose
      tree-sitter query compiler `HighlightQuery`/`TagsQuery` themselves
      compile through — works against *any* query string, so this is
      mechanically possible; the gap is purely that nobody has written
      the actual `@function.outer`/`@function.inner`/`@parameter.inner`-
      convention queries for any of this project's curated languages.
      Implementing this for real means pieces-store authoring and
      maintaining its own per-language text-object query set (at minimum
      Go, likely TS/JS next) — a genuinely bigger, ongoing-maintenance
      scope than "wire up an existing query," confirmed rather than
      assumed. **Awaiting a user decision on whether to take this on now
      given the real scope, or defer it** — not proceeding to
      implementation without that, per this task's own original gate.
      (L, 🧠 — scope confirmed bigger; awaiting go/no-go)
- [ ] **T50 — GUI popup pixel-exact positioning.** Real op-recording for a
      rune's actual screen position (capturing the layout op during
      `renderStyledRunes`' per-rune pass and reading its transform back),
      replacing today's font-metric estimate
      (`charWidthPx`/`lineHeightPx`). Self-contained Gio investigation,
      independent of everything else in this batch — can slot in whenever
      convenient. (M, 🧠)

## 10b. Deferred from this batch

- [ ] **T45 — Buffer-word completion fallback.** Deferred, not committed to
      this batch — genuine open question, not yet resolved: real vim's own
      buffer-keyword completion (`<C-n>`/`<C-p>` in Insert mode, outside
      any LSP popup) is a *separate*, manually-triggered mechanism from
      LSP's automatic trigger-character popup, not blended into it — but
      whether to match that distinction (separate trigger, own popup) or
      blend buffer words into `LSPCompletionState`'s results as fallback
      candidates is a real design fork with no clear answer yet. Revisit
      once there's real daily-use signal — specifically, editing a file
      with no LSP server attached, the actual scenario this feature exists
      for — rather than deciding it in the abstract now. (M, ❓)
- [ ] **T47 — Inlay hints.** Deferred, tied to **T42**: hint text isn't
      part of the buffer, so rendering it inline mid-line hits the exact
      same column-math re-derivation problem T42 already needs to solve
      for concealed markdown (a line's on-screen width diverging from its
      byte length, breaking cursor/hover-popup column math unless
      re-derived). Rather than invent a second rendering mechanism for the
      same class of problem, T47 should piggyback on whatever T42 builds,
      not be attempted independently. (M, 🧠 — blocked on T42's mechanism)

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
  `FEATURES.md`'s Future section for the full staged inventory and
  sequencing (absorbed `VIM_ACTION_CHECKLIST.md`/`IDE_FEATURE_CHECKLIST.md`
  in full, 2026-10-17; both are deleted now).

---

**How to use this**: pick from the top unless something below jumps out.
If you've got 20–30 minutes and your head's elsewhere, grab anything
tagged 🔧 — no design context to reload. If you want the "mine to build"
feeling, 🧠 items are picked specifically for that. Anything else, say
"do T<n>" and I'll pick up from there.

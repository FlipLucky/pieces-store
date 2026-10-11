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

## 2. Structural refactor — remainder

All mechanical reconnect work, same discipline as the four pieces already
done (`langdetect`, `autopairs`, `exmode`, `lspservice` — see `BACKLOG.md`
for what "done" looked like each time). Every item here is the same
underlying move: pulling logic out of `Editor` into the type that should
own it — see [decoupling rule / conveyor-belt splitter](docs/concepts/decoupling-splitter.md)
and [the orchestrator doesn't play an instrument](docs/concepts/orchestrator-instrument.md).

- [ ] **T4** — Move `SetMode`'s mode-transition cleanup onto
      `Cursor.SetMode` instead of `Editor` reaching into `Cursor`'s fields
      from outside. (M, 🔧)
- [ ] **T5** — Give `CommandBuffer` its own type (`Append`/`Backspace`/
      `Clear`/`String`) instead of a raw string field poked at by `Editor`
      methods. **Note:** this is also a soft prerequisite for Command
      mode's eventual Validator stage, if/when `:s///` (Section 4 below)
      needs one — see [Validator → Processor → Executor](docs/concepts/validator-processor-executor.md) —
      don't do it twice. (M, 🔧)
- [ ] **T6** — Give `CompletionState`/`LSPCompletionState` their own
      `Cycle(direction)` method instead of `Editor` doing the wraparound
      math externally. (S, 🔧)
- [ ] **T7** — Move `IndentStringAt`'s tab-formatting into
      `internal/syntax.Highlighter.IndentStringAt`, so `Editor`'s version
      becomes a pure passthrough. (S, 🔧)
- [ ] **T8** — Split file-related `Editor` methods (`NewEditorFromFile`,
      `OpenFile`, `SaveFile`, `SaveFileAs`, `GetFilePath`, `Language()`)
      into `internal/editor/file.go`. Pure move, no logic change. (S, 🔧)
- [ ] **T9** — Move `MoveCursorLeft/Right/Up/Down` (+ `*Locked` bodies) out
      of `editor.go` into `cursor.go`. Pure move, no logic change. (S, 🔧)

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

## 4. Find/replace (`:s///`) — the next roadmap item per `CLAUDE.md`

Deliberately out of scope for the pipeline redesign above — `:s///` is the
one Command-mode feature that will eventually fit Target+Action, but
nothing here needs the redesign to land first.

- [ ] **T13** — Design `:s/pattern/replacement/flags` parsing as a new
      `exmode` command shape. (M, ❓ — approachable either way; regex edge
      cases are a reasonable delegate candidate if time's short)
- [ ] **T14** — Wire execution into `executeCommandLocked`. (M, 🤖)
- [ ] **T15** — Support the `g` flag first; defer `c`/`i`/ranges — matches
      the staged approach `CLAUDE.md` already describes for this feature.
      (S, 🤖, depends on T13/T14)

## 5. Small, self-contained bug fixes (good solo confidence-builders)

- [ ] **T16** — Fix `piecetable.GetRuneAt`/`FindPieceAt` bypassing their
      own mutex — a real, self-contained concurrency bug, good for
      sharpening Go skills without touching unfamiliar territory. (M, 🧠)
- [ ] **T17** — Implement counted delete (`3diw`) in `interpreter`'s
      Delete-verb handling (`ProcessNormal`/`resolveRange`,
      `internal/interpreter/normal.go` — moved here from `dispatch.go`'s
      now-deleted `executeDelete` during T19). Mirrors the count-loop
      pattern `resolveMove` (same file) already uses for Move. (S, 🔧)
- [ ] **T18b** — `offset.MotionFindCharForward` (`f{char}`'s motion math)
      is fully implemented and unit-tested but was never wired into
      `keyengine` — no `f`/`F` verb or direct exists in `normal_mode.go`,
      found via a dead-code scan 2026-10-11. Looks like a primitive built
      ahead of its keybinding, not a mistake. Decide: wire up `f`/`F` as a
      real keybinding (the resolver's already done — this is mostly a
      `normal_mode.go` table entry plus a `resolveMotion` case in
      `interpreter/normal.go`), or delete it if `f`/`F` isn't wanted yet.
      Not decided here — your call. (S, ❓)

## Longer horizon — browse, don't start yet

- **Visual mode** — real design work (operators act directly on a
  selection, no following noun — a genuinely different shape from
  Normal's verb+noun grammar). Flagged previously as your carve-out to
  design/drive, not a default delegate. Its `StoreSelection` Target+Action
  (see `DESIGN_KEY_PIPELINE.md`) is now at least sketched, not designed.
- Search, macros, marks, splits/multi-buffer, folding, autocommands — see
  `VIM_ACTION_CHECKLIST.md` and `IDE_FEATURE_CHECKLIST.md` for the full
  unprioritized inventory; `FEATURES.md`'s Stage 3+ for sequencing.

---

**How to use this**: pick from the top unless something below jumps out.
If you've got 20–30 minutes and your head's elsewhere, grab anything
tagged 🔧 — no design context to reload. If you want the "mine to build"
feeling, 🧠 items are picked specifically for that. Anything else, say
"do T<n>" and I'll pick up from there.

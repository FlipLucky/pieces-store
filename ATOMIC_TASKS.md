# Atomic Tasks

A prioritized, pick-up-anytime queue — unlike `BACKLOG.md` (a sidebar list,
not ordered) and `VIM_ACTION_CHECKLIST.md`/`IDE_FEATURE_CHECKLIST.md` (full,
deliberately unprioritized inventories for weekend reading), this file is
the actual next few things to do, each sized to finish in one sitting.
Each item has a stable ID — say "do T4" rather than re-describing it.
Where an item leans on a named project principle, it links into
`CONCEPTS.md` instead of re-explaining that principle inline. Check items
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
- [ ] **T3 — Decide the `clipboardRegister` field's fate the same way.**
      It's declared on `Editor` but never constructed (always nil) or read
      from any keybinding — same half-started shape. Either start wiring
      real yank/paste now (T10–T12 below) or remove the field until that's
      actually being built, so the struct doesn't advertise a capability
      that silently does nothing. (S, 🧠)

## 1. The key-handling pipeline redesign — yours to drive

Settled 2026-10-10 as a real design, not just a T2 patch — full reasoning
in `DESIGN_KEY_PIPELINE.md`, promote to `CONCEPTS.md` once built. Short
version: `Editor.HandleKey` becomes **Validate → Interpret → Apply**, three
mode/`Target`-keyed handoffs, each one a
[conveyor-belt splitter](CONCEPTS.md#decoupling-rule--conveyor-belt-splitter).
This is bigger and more architectural than the rest of this file — treat it
as its own multi-session arc, not a quick pickup.

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
- [ ] **T20** — Design + build Insert mode's Validate + Interpret, per
      `DESIGN_KEY_PIPELINE.md`'s open questions (the surrounding-rune
      context interface in particular needs naming). (L, 🧠)
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
own it — see [decoupling rule / conveyor-belt splitter](CONCEPTS.md#decoupling-rule--conveyor-belt-splitter)
and [the orchestrator doesn't play an instrument](CONCEPTS.md#the-orchestrator-doesnt-play-an-instrument).

- [ ] **T4** — Move `SetMode`'s mode-transition cleanup onto
      `Cursor.SetMode` instead of `Editor` reaching into `Cursor`'s fields
      from outside. (M, 🔧)
- [ ] **T5** — Give `CommandBuffer` its own type (`Append`/`Backspace`/
      `Clear`/`String`) instead of a raw string field poked at by `Editor`
      methods. **Note:** this is also a soft prerequisite for Command
      mode's eventual Validate stage in the pipeline redesign above — see
      `DESIGN_KEY_PIPELINE.md`'s open questions — don't do it twice. (M, 🔧)
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

## 3. Registers & real yank/paste — yours to drive

`internal/clipboard-register` is your own design (built end-to-end last
time without me writing the implementation) — wiring it into real
keybindings is a natural continuation of that, not a new delegate-or-not
question. Each of these is a new instance of
[action+offset](CONCEPTS.md#actionoffset): a verb applying an action
through an existing resolver. **Note:** once T19–T21 above land, these
verbs are the first real content flowing through the new Interpret/Apply
pipeline rather than the old direct-dispatch path — worth sequencing after
the pipeline redesign, not before, so this isn't built twice.

- [ ] **T10** — Wire `y` as a new verb reusing the existing noun resolvers
      (`yw`, `yiw`, `yy` for the whole line), storing the result via
      `clipboardRegister.StoreRegisterValue(value, Yank)`. (M, 🧠)
- [ ] **T11** — Wire `p`/`P` as direct actions pasting
      `clipboardRegister.Paste()` after/before the cursor. **Note:** `p` is
      already taken in `keyengine`'s noun table for `Paragraph` (`dip`) —
      standalone paste needs its own key identity, not a collision with
      that. (M, 🧠)
- [ ] **T12** — Make `d` also store into the register (`Delete` type) so
      `dd` then `p` round-trips the way real vim does. (S, 🔧 once T10/T11
      exist)

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
- [ ] **T17** — Implement counted delete (`3diw`) in `executeDelete` —
      mirrors the count-loop pattern `executeMove` (same file,
      `dispatch.go`) already uses. (S, 🔧)

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

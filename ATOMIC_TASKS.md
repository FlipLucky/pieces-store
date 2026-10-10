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

- [ ] **T1 — Commit the pending diff.** `CLAUDE.md`'s "[orchestrator
      doesn't play an instrument](CONCEPTS.md#the-orchestrator-doesnt-play-an-instrument)"
      section and `editor.go`'s clipboard-register field + the
      `SIMULATE`-ticker removal are already done, just uncommitted. (S, 🔧)
- [ ] **T2 — Decide `internal/keyengine/insert_mode.go`'s fate.** It's an
      orphaned stub — `InsertKeyEngine{}` and an unused `createMapping`
      mapping only `<Esc>` — referenced nowhere. Reads like the start of
      pulling Insert-mode key handling out of `dispatch.go`'s
      `handleInsertModeKey` into its own file/type, mirroring how Normal
      mode got `internal/keyengine`. Either finish that extraction or
      delete the stub (and note the idea in `BACKLOG.md` instead) — just
      don't leave it as silent dead code. (S, 🧠 — your call on direction)
- [ ] **T3 — Decide the `clipboardRegister` field's fate the same way.**
      It's declared on `Editor` but never constructed (always nil) or read
      from any keybinding — same half-started shape. Either start wiring
      real yank/paste now (T6–T8 below) or remove the field until that's
      actually being built, so the struct doesn't advertise a capability
      that silently does nothing. (S, 🧠)

## 1. Structural refactor — remainder

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
      methods. (M, 🔧)
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

## 2. Registers & real yank/paste — yours to drive

`internal/clipboard-register` is your own design (built end-to-end last
time without me writing the implementation) — wiring it into real
keybindings is a natural continuation of that, not a new delegate-or-not
question. Each of these is a new instance of
[action+offset](CONCEPTS.md#actionoffset): a verb applying an action
through an existing resolver.

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

## 3. Find/replace (`:s///`) — the next roadmap item per `CLAUDE.md`

- [ ] **T13** — Design `:s/pattern/replacement/flags` parsing as a new
      `exmode` command shape. (M, ❓ — approachable either way; regex edge
      cases are a reasonable delegate candidate if time's short)
- [ ] **T14** — Wire execution into `executeCommandLocked`. (M, 🤖)
- [ ] **T15** — Support the `g` flag first; defer `c`/`i`/ranges — matches
      the staged approach `CLAUDE.md` already describes for this feature.
      (S, 🤖, depends on T13/T14)

## 4. Small, self-contained bug fixes (good solo confidence-builders)

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
  design/drive, not a default delegate.
- Search, macros, marks, splits/multi-buffer, folding, autocommands — see
  `VIM_ACTION_CHECKLIST.md` and `IDE_FEATURE_CHECKLIST.md` for the full
  unprioritized inventory; `FEATURES.md`'s Stage 3+ for sequencing.

---

**How to use this**: pick from the top unless something below jumps out.
If you've got 20–30 minutes and your head's elsewhere, grab anything
tagged 🔧 — no design context to reload. If you want the "mine to build"
feeling, 🧠 items are picked specifically for that. Anything else, say
"do T<n>" and I'll pick up from there.

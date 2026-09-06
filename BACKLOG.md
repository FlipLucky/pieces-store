# Backlog

A sidebar priority list — known, deliberately-deferred issues that don't
block the current phase. Pick these up opportunistically (when touching the
same context) or reactively (when the issue actually manifests as a real
problem), not proactively. See `CLAUDE.md`'s Roadmap section for the phase
plan this list sits alongside.

This is not the same as a blocker: if something here is actually stopping
the current phase's work, it belongs in `CLAUDE.md`'s Known Issues, not
here.

## Deferred performance (viewport-scoping changes the calculus here)

- **`piecetable.FindPieceAt` is an O(P) linear scan** (`internal/piecetable/reader.go`)
  with no index/tree over pieces. Deferred: once viewport slicing exists,
  this scales with piece count from editing activity, not file size — for a
  single-file MVP session, piece count should stay small enough that this
  doesn't matter in practice. Revisit if a long, heavily-edited session
  makes it measurably slow.
- **`Table.History` grows unbounded** — every `Insert`/`Delete` pushes a full
  snapshot forever, no cap or eviction. Revisit once real session lengths
  are known; a simple depth cap or coalescing strategy would fix it.
- **`internal/viewmanager/virtual_grid.go`'s `GetScreenPosition` is a no-op**
  — no scroll offsets, line wrapping, tab expansion, or gutter width yet.
  Needed for real files with tabs/long lines; not needed for a minimal
  plain-text MVP.

## Correctness/robustness (latent — not currently triggered, but real)

- **`piecetable.GetRuneAt`/`FindPieceAt` bypass their own mutex** on the
  actual read path — a genuine data race if ever called concurrently from
  outside `Editor`'s coarser lock. Not triggered today (single-goroutine
  usage per `Editor`), but a broken promise on a type that advertises
  thread safety via `sync.RWMutex`.
- **`GetRange` panics on out-of-range/reversed input**; `Insert` with a
  negative offset silently misplaces instead of erroring. No consistent
  failure discipline across `Insert`/`Delete`/`GetRange`/`FindPieceAt`. Will
  matter more once viewport work starts calling `GetRange` for real.
- **No rune-boundary guard on `Insert`/`Delete`/`GetRange`** — a
  non-rune-aligned byte offset would silently corrupt UTF-8 text. Not
  currently reachable (all current callers stay rune-aligned via
  `viewmanager`), but no defense-in-depth exists.

## API/interface cleanup (cheap, no urgency)

- `internal/editor/interfaces.go`'s `Document` interface is unused/dead —
  `viewmanager.Document` is the one actually consumed.
- `internal/editor/keymap` package (and its test file) should be deleted
  once `key_engine.go` fully replaces it — see `CLAUDE.md` Known Issues.
- Once `key_engine.go` is complete: `CreateModifiers()` is missing an
  `Around` entry despite being referenced in verbs' whitelists and the
  `KeyActionName` enum; `CreateNouns()` is missing `LineMotion`/`RuneMotion`
  entries despite existing as enum values.

## Needs a decision, not just a fix

- **Undo/redo model**: action-log-based (undo replays the inverse of the
  last action, redo re-applies it) vs. staying snapshot-based. Likely to
  get forced during the key-engine work anyway, since `u` needs real
  backing — see `CLAUDE.md` Roadmap.
- **`:q` has no dirty/unsaved-changes guard** — quits unconditionally, no
  `:q!` distinction possible since nothing tracks "modified." Flagged
  separately from the rest of this list because it's arguably not a
  sidebar item at all — losing unsaved work isn't acceptable behavior for
  even the most minimal text editor, so this may belong in the "backend
  needs to work" must-have bucket rather than deferred. Worth a deliberate
  call, not a default.

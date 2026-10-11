# internal/piecetable

**The crown jewel** — the actual reason this project exists. `Table` holds
`Master`/`Add` byte buffers plus a `Pieces` list; `Insert`/`Delete` mutate
it, `Coalesce` merges adjacent same-buffer pieces. Everything else in this
codebase exists to connect this to a user, not to be the interesting part
itself — see
[Dumb Front Ends / Editor as Spine](../concepts/dumb-front-ends.md).

## Undo/Redo

Snapshot-based (each unexported `historyEntry` restores a full `State`
directly, no replay) — not the bigger action-log redesign once floated.
Each entry also carries the precise `Edit{Offset, OldLength, NewLength}`
that produced it, so `Undo`/`Redo` report an exact reversible delta rather
than leaving callers to diff two snapshots to find out what changed. This
shape was forced by treesitter/LSP integration: incremental re-parsing and
`didChange` both need exactly this `Edit` shape, and diffing two full
snapshots to reconstruct it on every undo would have been real per-undo
cost on a large file.

## File I/O

`NewPieceTableFromFile`, `Save`, `SaveAs`. This package is a
mutex-protected rewrite of the older, now-deleted `internal/piecestore`
package (`Store` → `Table`).

## Known, deliberately-deferred gaps

- `GetRuneAt`/`FindPieceAt` bypass their own mutex on the actual read path
  — a real, currently-latent data race (not triggered by single-goroutine
  usage per `Editor` today).
- `FindPieceAt`'s O(P) linear scan is deliberately deferred — once
  viewport slicing exists it scales with piece count from editing
  activity, not file size.
- `Insert` with a negative offset silently misplaces instead of erroring;
  no rune-boundary guard on `Insert`/`Delete`/`GetRange`.

See `BACKLOG.md`'s Correctness/robustness section for the full, current
list — these are tracked there, not duplicated here in detail.

## Depends on

Nothing in this module (stdlib only).

## Depended on by

Nearly everything: [internal/editor](editor.md) directly;
[internal/offset](offset.md)/[internal/interpreter](interpreter.md)/
[internal/executor](executor.md) only via the small `Document`-shaped
interfaces they each declare (never importing `piecetable` directly — see
[Decoupling rule / conveyor-belt splitter](../concepts/decoupling-splitter.md)).

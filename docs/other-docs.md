# Other docs in this repo

This wiki is additive. These are the actively-edited files at the repo
root — when this wiki and any of them disagree, trust these:

- **`CLAUDE.md`** — the real source of truth for current package
  structure, design philosophy, roadmap, and a running "Current phase"
  changelog of what's shipped and when. Read this first if you want to
  know where the project actually stands today.
- **`BACKLOG.md`** — an unordered sidebar list of known, deliberately-
  deferred issues that don't block the current phase. Pick these up
  opportunistically, not proactively.
- **`ATOMIC_TASKS.md`** — an ordered, ID'd (`T1`, `T2`, ...) pick-up-
  anytime task queue, distinct from `BACKLOG.md`'s unordered list.
- **`FEATURES.md`** — the product-facing view: what the editor actually
  does today, how it got here, where it's going.
- **`DESIGN_KEY_PIPELINE.md`** — retired, history-only. The original
  design-conversation trail for the Validator→Processor→Executor
  pipeline, kept for that history; the
  [Concepts](concepts/validator-processor-executor.md) page here and
  `CLAUDE.md`'s Current phase section are the current source of truth,
  not this file.
- **`README.md`**, **`CONCEPTUAL_GUIDE.md`** — stale; predate the
  `piecestore`→`piecetable` rename and the whole `keyengine`/`offset`/
  `viewmanager` split. Don't trust these for architecture; see `CLAUDE.md`
  or this wiki's Packages section instead.

`CONCEPTS.md` used to live at the repo root too — its entries are now the
**Concepts** section of this wiki's sidebar, so that file is gone and
anything that used to link to it points here instead.

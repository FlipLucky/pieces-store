# Small / leaf packages

Each of these is small, single-purpose, and (per
[Decoupling rule / conveyor-belt splitter](../concepts/decoupling-splitter.md))
deliberately has few or no internal dependencies of its own.

## internal/types

The shared kernel: small, dependency-free enums every other package agrees
on, imported freely but never importing anything else in the module
itself.

- `Mode` (Normal/Insert/Visual/Command).
- `Style` — a small syntax-highlighting/diagnostic category enum (see
  [viewmanager](viewmanager.md)).
- `Language` + `DetectLanguage` (now actually `internal/langdetect`'s job
  — see below; `types.Language` is just the enum).
- `ActionKind` — the one shared vocabulary item
  [interpreter](interpreter.md)'s Processor and [executor](executor.md)'s
  Executor both agree on. See
  [Decoupled by shape, not by sender](../concepts/decoupled-by-shape.md).

## internal/langdetect

Extension→`types.Language` detection, split out of `internal/types` to
keep that package logic-free. Extension-only detection, deliberately not
content-sniffing or shebang-detection (Dockerfile itself is the one
special-cased filename, since it has no extension at all).

## internal/autopairs

Bracket/quote pairing decision logic, split out of `Editor` — a pure
function (`Decide`) [interpreter](interpreter.md)'s `insert.go` calls
directly. Typing `{`/`(`/`[` inserts the matching close too; typing the
close yourself skips over an already-auto-inserted one instead of
doubling up; `shouldPairQuoteHere` only pairs quotes when neither
neighboring character is a word character (so `don't` doesn't trigger an
unwanted pair on the apostrophe).

## internal/exmode

Command-mode (`:...`) parsing, split out of `Editor`, mirroring how Normal
mode already got [keyengine](keyengine.md). The planned future home for
`:s///` parsing (see `ATOMIC_TASKS.md`'s T13-T15) — deliberately almost
entirely out of scope for the Validator→Processor→Executor redesign, since
`:w`/`:q`/`:e` are genuine side effects, not Target+Action.

## Depends on

Each other, minimally: `langdetect` and `autopairs` depend only on
`types`; `exmode` depends only on `types`.

## Depended on by

[internal/editor](editor.md) (all four), [internal/interpreter](interpreter.md)
(`autopairs`, `types`).

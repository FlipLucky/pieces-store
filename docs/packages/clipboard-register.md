# internal/clipboard-register

The user's own design, later wired into real keybindings (2026-10-12).
Backs `y`/`d`/`c`'s store side and `p`/`P`'s read side.

## Shape

`ClipboardRegister` holds several named `Register`s
(`unnamedRegister, yankRegister, visualRegister, deleteRegister`) plus a
`getRegister(RegisterType) *Register` lookup — the textbook instance of
[Decoupling rule / conveyor-belt splitter](../concepts/decoupling-splitter.md):
a caller supplies a type and a value and gets a result, with zero
knowledge of which struct field actually backs it.

`Register` is `[]Entry{Value string, Linewise bool}` — **not** a bare
`[]string`. The `Linewise` flag (added 2026-10-11) is what lets a later
paste tell a whole-line `dd`/`yy`/`cc` delete/yank apart from a charwise
`diw`/`yiw`/`ciw` one, so `p`/`P` can insert the former as a genuine new
line instead of splicing it into whatever line the cursor sits on.

`StoreRegisterValue` always prepends to the unnamed register too (capped
at 9 entries) in addition to whichever category register was given —
matches real vim's "every yank/delete also updates the unnamed register"
rule.

## Named registers (T24, 2026-10-11)

`namedRegister map[rune]Entry` — single-value per letter, no history
(unlike the category registers' 9-entry cap), per the user's own call:
"i didnt even know vim had an uppercase named register. lowercase is
enough" — only lowercase letters are ever used, uppercase-append (real
vim's own convention) is deliberately not supported. `StoreRegisterValue`
takes a `named rune` parameter (0 means none selected); `GetValueFromNamedRegister`
reads one back. Selected via a `"{letter}` prefix parsed in
[keyengine](keyengine.md) (`"ayw`, `"adw`, `"ap`), threaded through
`keyengine.ExecutableCommand.Register` → `interpreter`'s
`DeleteAction`/`YankAction.Named` → `executor`'s `hasNamedRegister`
capability.

## Register overview read path (T25, 2026-10-11)

`TopEntries(registerType, n)` and `NamedEntries()` back `:register`'s
display — see [editor](editor.md)'s `registers.go`. `TopEntries`
special-cases `Unnamed` (the one `RegisterType` `getRegister` doesn't
resolve, since it's always written directly rather than routed through
the splitter).

## How it's set from the key sequence, not a new rule

[interpreter](interpreter.md)'s `ProcessNormal` sets `IsLinewise` on
`DeleteAction`/`YankAction` from `cmd.Noun == keyengine.LineMotion` — the
exact same signal that already makes `dd`/`yy` linewise in
[keyengine](keyengine.md)'s own doubled-verb mechanism. No new parsing
needed; the Processor determines it from the key sequence it already has.

## `RegisterType.Visual` actually populated (T52, 2026-10-18)

The `visualRegister` field and `RegisterType.Visual` constant existed
since before Visual mode itself did — `:registers`' display formatting
(T25) was already wired to show it, just with nothing ever stored there.
T52's `d`/`x`/`y`/`c` in Visual mode are the first real producers: they
store into `RegisterType.Visual` (and the unnamed register, same as every
delete/yank), not Normal mode's `Delete`/`Yank` categories, keeping
Visual's own history genuinely separate from `dd`'s or `yy`'s in the
`:registers` overview. `YankAction` needed a new `Register` field +
`RegisterDestination()` method to make this possible at all — see
[executor](executor.md)'s entry on the pre-existing hardcoding this fixed.

`Entry.Value string` → `Entry.Values []string` (needed for T54's block
mode, where one register entry spans N rows) was deliberately **not** done
here — T52 never produced more than one value per entry, so there was no
second real caller yet to justify the wider shape.

## `Entry.Values []string` widened for block mode (T54, 2026-10-18)

T54 is the task that actually earned the widening predicted above:
`Entry.Value string` → `Entry.Values []string`, with `len(Values) == 1` for
every existing charwise/linewise entry (unchanged behavior) and
`len(Values) == N` (one string per row) for a block-visual entry — sharing
this one field rather than bolting on a second, parallel shape.
`len(Values) > 1` is the signal `Paste`/`:registers` use to tell a block
entry apart from a charwise/linewise one, no separate flag needed.

New `StoreBlockRegisterValue(values []string, registerType, named rune)`
is `StoreRegisterValue`'s block-mode sibling: stores one multi-value
`Entry` into the unnamed register, the given category register, and (if
set) a named letter register — otherwise identical plumbing.
`registers.go`'s `formatRegisterLine` shows a block entry with a `b` kind
marker (checked before the existing `l`/`c` linewise/charwise check).

## A real edge case, found and fixed

Pasting *after* the buffer's last line (which has no trailing `"\n"` of
its own) needs a leading `"\n"` inserted ahead of the register value as a
separator, or the two lines glue together character-for-character. See
`interpreter.ProcessNormal`'s `Paste` case and
`internal/editor/clipboard_test.go`'s `TestDeleteThenPasteRoundTrips`
(exercises exactly this case).

## `r` deliberately stays out of this pipeline entirely

A real, tested guarantee (`TestReplaceDoesNotTouchRegister`), not just an
omission — real vim's `r` doesn't touch registers either.

## Depends on

Nothing in this module.

## Depended on by

[interpreter](interpreter.md) (for `RegisterType`, set on
`DeleteAction`/`YankAction`), [executor](executor.md) (same, plus calling
`StoreRegisterValue`/`Paste` through `editor.lockedEditorView`).

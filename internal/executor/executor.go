// Package executor is Executor's home in the Validator → Processor →
// Executor key-handling pipeline (see docs/concepts/validator-processor-
// executor.md and DESIGN_KEY_PIPELINE.md for the full history). Its only
// job is dispatch: given an already-resolved Action, decide which of
// Editor's capabilities to invoke. It never resolves anything itself and
// never imports internal/interpreter — Action's producer is irrelevant to
// it, only its shape is (see docs/concepts/decoupled-by-shape.md).
package executor

import (
	clipboardregister "github.com/fliplucky/pieces-store/internal/clipboard-register"
	"github.com/fliplucky/pieces-store/internal/offset"
	"github.com/fliplucky/pieces-store/internal/types"
)

// Action is an already-resolved, mode-blind unit of work — Executor's only
// input. Deliberately not shared with internal/interpreter's identically-
// shaped Action interface: each producer defines its own concrete type
// satisfying this by having the right methods, never importing this
// package to do it.
type Action interface {
	Kind() types.ActionKind
}

// Editor is the narrow slice of *editor.Editor's capabilities Dispatch
// needs — declared here, satisfied structurally by the real *editor.Editor
// (via an internal adapter that assumes its own lock is already held)
// without this package ever importing editor. Mirrors
// viewmanager.Document's precedent for the same problem. Every method here
// is already real, correct Editor logic; Dispatch's job is choosing which
// one a given Action maps to, not reimplementing what they already do.
type Editor interface {
	MoveCursorToOffset(target int)
	DeleteRange(rng offset.TextRange)
	// DeleteRangeWithRegister is DeleteRange plus storing the range's text
	// into register before deleting it (d/c's own behavior) — a separate
	// method rather than a parameter on DeleteRange, so DeleteRange's
	// existing callers (Insert mode's backspace, via a DeleteAction that
	// leaves StoreInRegister false) are untouched. linewise is threaded
	// through to the stored register entry so a later paste knows whether
	// to splice charwise or insert as a fresh line.
	DeleteRangeWithRegister(rng offset.TextRange, register clipboardregister.RegisterType, linewise bool)
	// YankRange stores the range's text into register and moves the
	// cursor to its start, without touching the buffer at all.
	YankRange(rng offset.TextRange, register clipboardregister.RegisterType, linewise bool)
	Undo() bool
	Redo() bool
	SetMode(m types.Mode)
	BeginPendingReplace()
	ReplaceRune(r rune)
	OpenLine(below bool)
	RequestHover()
	InsertTextAtCursor(text string, caretOffset int)
	InsertNewlineWithIndent()
	// Named ...LSPCompletion, not the shorter CycleCompletion/
	// ConfirmCompletion, to avoid colliding in spirit with Editor's own
	// pre-existing CycleCompletion (Command mode's :e/:w path completion
	// — an unrelated feature with its own state, cursor.Completion, not
	// cursor.LSPCompletion).
	ConfirmLSPCompletion()
	CycleLSPCompletion(direction int)
}

// Small, optional-capability interfaces for pulling an Action's payload
// back out — one per distinct shape actually needed below. Satisfied
// structurally by whatever concrete type produced the Action; this
// package never imports that type's defining package to know it exists.
type (
	hasOffset interface{ Offset() (int, bool) }
	hasRange  interface {
		Range() (offset.TextRange, bool)
	}
	hasCount     interface{ Count() int }
	hasRune      interface{ Rune() rune }
	hasBelow     interface{ Below() bool }
	hasInsertion interface {
		Insertion() (text string, caretOffset int)
	}
	hasDirection           interface{ Direction() int }
	hasRegisterDestination interface {
		RegisterDestination() (clipboardregister.RegisterType, bool)
	}
	hasLinewise interface{ Linewise() bool }
)

// Dispatch performs a, calling whichever of ed's methods its Kind maps to.
// A malformed Action (missing the payload interface its Kind implies, or
// a producer reporting ok=false) is a no-op, not a panic — the same "this
// executor doesn't know how, so it does nothing" contract
// resolveMotion/resolveRange already established.
func Dispatch(a Action, ed Editor) {
	if a == nil {
		return
	}

	switch a.Kind() {
	case types.ActionMove:
		if p, ok := a.(hasOffset); ok {
			if target, has := p.Offset(); has {
				ed.MoveCursorToOffset(target)
			}
		}
	case types.ActionDelete:
		if p, ok := a.(hasRange); ok {
			if rng, has := p.Range(); has {
				register, store := clipboardregister.RegisterType(0), false
				if rp, ok := a.(hasRegisterDestination); ok {
					register, store = rp.RegisterDestination()
				}
				if store {
					linewise := false
					if lp, ok := a.(hasLinewise); ok {
						linewise = lp.Linewise()
					}
					ed.DeleteRangeWithRegister(rng, register, linewise)
				} else {
					ed.DeleteRange(rng)
				}
			}
		}
	case types.ActionYank:
		if p, ok := a.(hasRange); ok {
			if rng, has := p.Range(); has {
				linewise := false
				if lp, ok := a.(hasLinewise); ok {
					linewise = lp.Linewise()
				}
				ed.YankRange(rng, clipboardregister.Yank, linewise)
			}
		}
	case types.ActionUndo:
		if p, ok := a.(hasCount); ok {
			for i, n := 0, atLeastOne(p.Count()); i < n; i++ {
				if !ed.Undo() {
					break
				}
			}
		}
	case types.ActionRedo:
		if p, ok := a.(hasCount); ok {
			for i, n := 0, atLeastOne(p.Count()); i < n; i++ {
				if !ed.Redo() {
					break
				}
			}
		}
	case types.ActionBeginReplace:
		ed.BeginPendingReplace()
	case types.ActionReplaceRune:
		if p, ok := a.(hasRune); ok {
			ed.ReplaceRune(p.Rune())
		}
	case types.ActionEnterInsert:
		ed.SetMode(types.ModeInsert)
	case types.ActionEnterCommand:
		ed.SetMode(types.ModeCommand)
	case types.ActionOpenLine:
		if p, ok := a.(hasBelow); ok {
			ed.OpenLine(p.Below())
		}
	case types.ActionHover:
		ed.RequestHover()
	case types.ActionInsertText:
		if p, ok := a.(hasInsertion); ok {
			text, caret := p.Insertion()
			ed.InsertTextAtCursor(text, caret)
		}
	case types.ActionNewline:
		ed.InsertNewlineWithIndent()
	case types.ActionExitInsert:
		ed.SetMode(types.ModeNormal)
	case types.ActionConfirmCompletion:
		ed.ConfirmLSPCompletion()
	case types.ActionCycleCompletion:
		if p, ok := a.(hasDirection); ok {
			ed.CycleLSPCompletion(p.Direction())
		}
	}
}

func atLeastOne(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

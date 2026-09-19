// Package keyengine parses vim-style keystroke sequences into a resolved
// ExecutableCommand{Verb, Modifier, Noun}. It knows nothing about
// piecetable, offset resolution, or Editor — keystrokes in, a command (or
// error) out. It's the "action" half of the action+offset model; the
// offset package is the other half, and the two are only connected by
// whatever executor consumes an ExecutableCommand.
package keyengine

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/fliplucky/pieces-store/internal/types"
)

type KeyActionName int

const (
	None KeyActionName = iota

	// Verbs
	Delete
	Change // declared for future use (ciw etc.) — not yet registered in any table
	Move
	Undo
	Redo
	Replace
	EnterInsert
	EnterCommand
	OpenBelow
	OpenAbove

	// Modifiers
	Inside
	Around

	// Nouns
	Word
	WordBackward
	WordEnd
	LineMotion
	Paragraph
	RuneMotion // declared for future use (x etc.) — not yet registered in any table
)

// ErrInconsistentKeymap signals a whitelist/semantic-table mismatch: a key
// was accepted as a valid continuation (KeyAction.allowedKeyActions said so)
// but no modifier or noun table actually defines it. That's a bug in the
// tables themselves, not invalid user input — callers should treat it as
// loud/fatal during development rather than swallow it like a plain fizzle.
var ErrInconsistentKeymap = errors.New("key engine inconsistency")

type KeyAction struct {
	name              KeyActionName
	allowedKeyActions map[string]KeyActionName
}

// DirectAction is a fully-resolved (Verb, Modifier, Noun) triple for a key
// that executes immediately with no further input needed — the counterpart
// to verb/modifier/noun sequences for keys that aren't composed (w, u, :,
// i, o, O, ...). The same Verb/Noun identities used by composed sequences
// are reused here (e.g. Move+Word is the same "Word" noun diw/daw use).
type DirectAction struct {
	Verb     KeyActionName
	Modifier KeyActionName
	Noun     KeyActionName
}

type ExecutableCommand struct {
	Count    int
	Verb     KeyActionName
	Modifier KeyActionName
	Noun     KeyActionName
}

type CommandContext struct {
	count               int
	countStarted        bool
	verb                KeyAction
	modifier            KeyAction
	noun                KeyAction
	normalModeVerbs     map[string]KeyAction
	normalModeModifiers map[string]KeyAction
	normalModeNouns     map[string]KeyAction
	normalModeDirects   map[string]DirectAction
	visualModeVerbs     map[string]KeyAction
	visualModeModifiers map[string]KeyAction
	visualModeNouns     map[string]KeyAction
	editorMode          types.Mode
}

func (cc *CommandContext) SetEditorMode(mode types.Mode) {
	cc.editorMode = mode
}

func CreateCommandContext() *CommandContext {
	none := KeyAction{name: None}
	return &CommandContext{
		count:               1,
		verb:                none,
		modifier:            none,
		noun:                none,
		normalModeVerbs:     CreateNormalModeVerbs(),
		normalModeModifiers: CreateNormalModeModifiers(),
		normalModeNouns:     CreateNormalModeNouns(),
		normalModeDirects:   CreateNormalModeDirects(),
		visualModeVerbs:     CreateVisualModeVerbs(),
		visualModeModifiers: CreateVisualModeModifiers(),
		visualModeNouns:     CreateVisualModeNouns(),
	}
}

// modeTables picks the verb/modifier/noun/direct tables for whatever mode
// CommandContext was last told about via SetEditorMode. Visual mode has no
// direct table yet (nothing there is implemented beyond the copied
// verb/modifier/noun scaffolding), which is fine — a nil map lookup is a
// safe, always-"not found" no-op in Go.
func (cc *CommandContext) modeTables() (verbs, modifiers, nouns map[string]KeyAction, directs map[string]DirectAction) {
	if cc.editorMode == types.ModeVisual {
		return cc.visualModeVerbs, cc.visualModeModifiers, cc.visualModeNouns, nil
	}
	return cc.normalModeVerbs, cc.normalModeModifiers, cc.normalModeNouns, cc.normalModeDirects
}

// ProcessCommand feeds one keystroke into the parser. It returns a non-nil
// *ExecutableCommand once a full sequence resolves, (nil, nil) while a
// sequence is still pending or the key didn't start/continue anything (a
// plain fizzle — not a registered sequence, expected and silent), or
// (nil, err) wrapping ErrInconsistentKeymap when a key passed a whitelist
// check but no modifier/noun table actually defines it — a real bug in the
// tables, not user input, and should not be treated the same as a fizzle.
func (cc *CommandContext) ProcessCommand(keyInput string) (*ExecutableCommand, error) {
	// 1. Digit accumulation for count prefixes. Unconditional on sequence
	// state so counts still work mid-sequence (e.g. "d3w"), not just at the
	// root (e.g. "3dw") — only special-cased for a leading "0", which is a
	// motion (not yet implemented), not a count digit, unlike every other
	// digit.
	if countInput, err := strconv.Atoi(keyInput); err == nil {
		if !(countInput == 0 && !cc.countStarted) {
			if !cc.countStarted {
				cc.count = countInput
			} else {
				cc.count = cc.count*10 + countInput
			}
			cc.countStarted = true
			return nil, nil
		}
	}

	verbs, modifiers, nouns, directs := cc.modeTables()

	if cc.verb.name == None {
		if direct, found := directs[keyInput]; found {
			return cc.finalizeDirect(direct), nil
		}
		if verb, found := verbs[keyInput]; found {
			cc.verb = verb
			return nil, nil
		}
		cc.clearCommandContext()
		return nil, nil
	}

	if cc.modifier.name == None {
		// A verb's own key, pressed again, means "linewise" (dd, and later
		// yy/cc) — checked generically by identity against the verb table,
		// not hardcoded per letter.
		if sameVerb, found := verbs[keyInput]; found && sameVerb.name == cc.verb.name {
			cc.noun = KeyAction{name: LineMotion}
			return cc.finalize(), nil
		}

		allowedAs, isWhitelisted := cc.verb.allowedKeyActions[keyInput]
		if !isWhitelisted {
			if noun, found := nouns[keyInput]; found {
				cc.noun = noun
				return cc.finalize(), nil
			}
			cc.clearCommandContext()
			return nil, nil
		}

		if modifier, found := modifiers[keyInput]; found {
			cc.modifier = modifier
			return nil, nil
		}
		if noun, found := nouns[keyInput]; found {
			cc.noun = noun
			return cc.finalize(), nil
		}

		cc.clearCommandContext()
		return nil, fmt.Errorf("%w: verb %v whitelists %q as %v, but no modifier or noun matches it", ErrInconsistentKeymap, cc.verb.name, keyInput, allowedAs)
	}

	allowedAs, isWhitelisted := cc.modifier.allowedKeyActions[keyInput]
	if !isWhitelisted {
		cc.clearCommandContext()
		return nil, nil
	}
	if noun, found := nouns[keyInput]; found {
		cc.noun = noun
		return cc.finalize(), nil
	}

	cc.clearCommandContext()
	return nil, fmt.Errorf("%w: modifier %v whitelists %q as %v, but no noun matches it", ErrInconsistentKeymap, cc.modifier.name, keyInput, allowedAs)
}

func (cc *CommandContext) finalize() *ExecutableCommand {
	ec := cc.createExecutableCommand()
	cc.clearCommandContext()
	return ec
}

func (cc *CommandContext) finalizeDirect(d DirectAction) *ExecutableCommand {
	ec := &ExecutableCommand{
		Count:    cc.count,
		Verb:     d.Verb,
		Modifier: d.Modifier,
		Noun:     d.Noun,
	}
	cc.clearCommandContext()
	return ec
}

func (cc *CommandContext) clearCommandContext() {
	none := KeyAction{name: None}
	cc.count = 1
	cc.countStarted = false
	cc.verb = none
	cc.modifier = none
	cc.noun = none
}

func (cc *CommandContext) createExecutableCommand() *ExecutableCommand {
	return &ExecutableCommand{
		Count:    cc.count,
		Verb:     cc.verb.name,
		Modifier: cc.modifier.name,
		Noun:     cc.noun.name,
	}
}

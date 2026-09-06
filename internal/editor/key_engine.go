package editor

import (
	"strconv"
)

type KeyActionName int

const (
	None KeyActionName = iota

	// Verbs
	Delete
	Change

	// Modifiers
	Inside
	Around

	// Nouns
	Word
	LineMotion
	RuneMotion
)

type KeyAction struct {
	name              KeyActionName
	allowedKeyActions map[string]KeyActionName
}

type ExecutableCommand struct {
	Count    int
	Verb     KeyActionName
	Modifier KeyActionName
	Noun     KeyActionName
}

type CommandContext struct {
	count              int
	verb               KeyAction
	modifier           KeyAction
	noun               KeyAction
	availableVerbs     map[string]KeyAction
	availableModifiers map[string]KeyAction
	availableNouns     map[string]KeyAction
}

func CreateCommandContext() *CommandContext {
	None := KeyAction{name: None}
	return &CommandContext{
		count:              1,
		verb:               None,
		modifier:           None,
		noun:               None,
		availableVerbs:     CreateVerbs(),
		availableModifiers: CreateModifiers(),
		availableNouns:     CreateNouns(),
	}
}

func (cc *CommandContext) ProcessCommand(keyInput string) *ExecutableCommand {
	// 1. Process digits for sequence multipliers
	if countInput, err := strconv.Atoi(keyInput); err == nil {
		if cc.count == 1 && countInput != 0 {
			cc.count = countInput
		} else {
			cc.count = (cc.count * 10) + countInput
		}
		return nil
	}

	if cc.verb.name == None {
		if noun, found := cc.availableNouns[keyInput]; found {
			cc.noun = noun
			return cc.finalize()
		}
		if verb, found := cc.availableVerbs[keyInput]; found {
			cc.verb = verb
			return nil
		}
		cc.clearCommandContext()
		return nil
	}

	if cc.verb.name != None && cc.modifier.name == None {
		if _, allowed := cc.verb.allowedKeyActions[keyInput]; !allowed {
			if noun, found := cc.availableNouns[keyInput]; found {
				cc.noun = noun
				return cc.finalize()
			}
			cc.clearCommandContext()
			return nil
		}

		if modifier, found := cc.availableModifiers[keyInput]; found {
			cc.modifier = modifier
			return nil
		}
	}

	if cc.modifier.name != None {
		if _, allowed := cc.modifier.allowedKeyActions[keyInput]; allowed {
			if noun, found := cc.availableNouns[keyInput]; found {
				cc.noun = noun
				return cc.finalize()
			}
		}
	}

	cc.clearCommandContext()
	return nil
}

func (cc *CommandContext) finalize() *ExecutableCommand {
	ec := cc.createExecutableCommand()
	cc.clearCommandContext()
	return ec
}

func (cc *CommandContext) clearCommandContext() {
	none := KeyAction{name: None}
	cc.count = 1
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

func CreateVerbs() map[string]KeyAction {
	return map[string]KeyAction{
		"d": {
			name:              Delete,
			allowedKeyActions: map[string]KeyActionName{"i": Inside, "a": Around},
		},
		"c": {
			name:              Change,
			allowedKeyActions: map[string]KeyActionName{"i": Inside, "a": Around},
		},
	}
}

func CreateModifiers() map[string]KeyAction {
	return map[string]KeyAction{
		"i": {
			name:              Inside,
			allowedKeyActions: map[string]KeyActionName{"w": Word},
		},
	}
}
func CreateNouns() map[string]KeyAction {
	return map[string]KeyAction{
		"w": {
			name:              Word,
			allowedKeyActions: map[string]KeyActionName{},
		},
	}
}

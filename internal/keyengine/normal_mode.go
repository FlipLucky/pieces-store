package keyengine

func CreateNormalModeVerbs() map[string]KeyAction {
	return map[string]KeyAction{
		"d": {
			name:              Delete,
			allowedKeyActions: map[string]KeyActionName{"i": Inside, "a": Around},
		},
	}
}

func CreateNormalModeModifiers() map[string]KeyAction {
	return map[string]KeyAction{
		"i": {
			name:              Inside,
			allowedKeyActions: map[string]KeyActionName{"w": Word, "p": Paragraph},
		},
		"a": {
			name:              Around,
			allowedKeyActions: map[string]KeyActionName{"w": Word},
		},
	}
}

func CreateNormalModeNouns() map[string]KeyAction {
	return map[string]KeyAction{
		"w": {name: Word, allowedKeyActions: map[string]KeyActionName{}},
		"p": {name: Paragraph, allowedKeyActions: map[string]KeyActionName{}},
	}
}

// CreateNormalModeDirects covers every key that executes immediately with
// no modifier/noun needed: silent motions (w/b/e reuse the same noun
// identities diw/daw would resolve), editor-level commands (u, <C-r>, r),
// and mode switches (:, i, o, O).
func CreateNormalModeDirects() map[string]DirectAction {
	return map[string]DirectAction{
		"w":     {Verb: Move, Noun: Word},
		"b":     {Verb: Move, Noun: WordBackward},
		"e":     {Verb: Move, Noun: WordEnd},
		"u":     {Verb: Undo},
		"<C-r>": {Verb: Redo},
		"r":     {Verb: Replace},
		":":     {Verb: EnterCommand},
		"i":     {Verb: EnterInsert},
		"o":     {Verb: OpenBelow},
		"O":     {Verb: OpenAbove},
	}
}

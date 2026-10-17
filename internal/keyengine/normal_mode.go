package keyengine

func CreateNormalModeVerbs() map[string]KeyAction {
	return map[string]KeyAction{
		"d": {
			name:              Delete,
			allowedKeyActions: map[string]KeyActionName{"i": Inside, "a": Around},
		},
		"c": {
			name:              Change,
			allowedKeyActions: map[string]KeyActionName{"i": Inside, "a": Around},
		},
		"y": {
			name:              Yank,
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

// CreateNormalModeNouns covers nouns reached directly after a verb with no
// modifier (dw, db, de, dh, dl — a "short delete," see
// interpreter.resolveMotionRange) or after a modifier (diw, dip). w/p are
// the only ones that pair with a modifier today; b/e/h/l are only ever
// reached bare, same identities Move's own silent-motion directs already
// use (see CreateNormalModeDirects).
func CreateNormalModeNouns() map[string]KeyAction {
	return map[string]KeyAction{
		"w": {name: Word, allowedKeyActions: map[string]KeyActionName{}},
		"p": {name: Paragraph, allowedKeyActions: map[string]KeyActionName{}},
		"b": {name: WordBackward, allowedKeyActions: map[string]KeyActionName{}},
		"e": {name: WordEnd, allowedKeyActions: map[string]KeyActionName{}},
		"h": {name: CharBackward, allowedKeyActions: map[string]KeyActionName{}},
		"l": {name: CharForward, allowedKeyActions: map[string]KeyActionName{}},
	}
}

// CreateNormalModeDirects covers every key that executes immediately with
// no modifier/noun needed: silent motions (w/b/e reuse the same noun
// identities diw/daw would resolve; h/j/k/l and the arrow keys are plain
// cursor motions with no operator-pending counterpart at all), editor-level
// commands (u, <C-r>, r), and mode switches (:, i, o, O). Arrow keys map to
// the exact same DirectAction as their hjkl equivalent — same command,
// just a different physical key — so Normal mode treats them identically.
func CreateNormalModeDirects() map[string]DirectAction {
	return map[string]DirectAction{
		"w":       {Verb: Move, Noun: Word},
		"b":       {Verb: Move, Noun: WordBackward},
		"e":       {Verb: Move, Noun: WordEnd},
		"h":       {Verb: Move, Noun: CharBackward},
		"<Left>":  {Verb: Move, Noun: CharBackward},
		"l":       {Verb: Move, Noun: CharForward},
		"<Right>": {Verb: Move, Noun: CharForward},
		"k":       {Verb: Move, Noun: LineUp},
		"<Up>":    {Verb: Move, Noun: LineUp},
		"j":       {Verb: Move, Noun: LineDown},
		"<Down>":  {Verb: Move, Noun: LineDown},
		"u":       {Verb: Undo},
		"<C-r>":   {Verb: Redo},
		"r":       {Verb: Replace},
		":":       {Verb: EnterCommand},
		"i":       {Verb: EnterInsert},
		"o":       {Verb: OpenBelow},
		"O":       {Verb: OpenAbove},
		"K":       {Verb: Hover},
		"p":       {Verb: Paste},
		"P":       {Verb: PasteBefore},
		"f":       {Verb: FindCharForward},
		"F":       {Verb: FindCharBackward},
		"x":       {Verb: Delete, Noun: CharForward},
	}
}

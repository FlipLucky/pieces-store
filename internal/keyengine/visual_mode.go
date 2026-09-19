package keyengine

// Deliberately left as the original copy-paste scaffolding, not fleshed
// out: no Visual-mode keybinding is in the agreed 14-case list, nothing
// can reach Visual mode yet (no key enters it), and real Visual-mode
// operator semantics differ from Normal's (an operator acts on the current
// selection directly, no following noun needed) — a straight copy of
// Normal's tables wouldn't be correct Visual behavior anyway. Revisit when
// Visual mode is actually in scope.

func CreateVisualModeVerbs() map[string]KeyAction {
	return map[string]KeyAction{
		"d": {
			name:              Delete,
			allowedKeyActions: map[string]KeyActionName{"i": Inside, "a": Around},
		},
	}
}

func CreateVisualModeModifiers() map[string]KeyAction {
	return map[string]KeyAction{
		"i": {
			name:              Inside,
			allowedKeyActions: map[string]KeyActionName{"w": Word},
		},
	}
}

func CreateVisualModeNouns() map[string]KeyAction {
	return map[string]KeyAction{
		"w": {name: Word, allowedKeyActions: map[string]KeyActionName{}},
	}
}

package keymap

// DefaultGraphBuilder constructs standard graphs for Vim modes

func BuildDefaultGraphs() map[int]map[string]*KeyNode {
	// Mode Constants (matching editor.Mode ints: 0=Normal, 1=Insert, 2=Visual, 3=Command)
	const (
		ModeNormal  = 0
		ModeInsert  = 1
		ModeVisual  = 2
		ModeCommand = 3
	)

	// --- 1. Define Shared Nouns ---
	wordNode := &KeyNode{
		Name: "w",
		Type: TypeNoun,
		Execute: func(ed EditorInterface, ctx CommandContext) {
			for i := 0; i < ctx.Count; i++ {
				switch {
				case ctx.Verb == "d" && ctx.Modifier == "i":
					// diw (delete inner word) - placeholder logic: delete text
					ed.DeleteText()
				case ctx.Verb == "d":
					// dw (delete word)
					ed.DeleteText()
				}
			}
		},
	}

	lineNode := &KeyNode{
		Name: "d",
		Type: TypeNoun,
		Execute: func(ed EditorInterface, ctx CommandContext) {
			for i := 0; i < ctx.Count; i++ {
				// dd (delete full line)
				ed.DeleteText()
			}
		},
	}

	// --- 2. Define Shared Modifiers ---
	innerModifier := &KeyNode{
		Name: "i",
		Type: TypeModifier,
		AllowedTransitions: map[string]*KeyNode{
			"w": wordNode,
		},
	}

	// --- 3. Define Verbs ---
	deleteVerb := &KeyNode{
		Name: "d",
		Type: TypeVerb,
		AllowedTransitions: map[string]*KeyNode{
			"d": lineNode,      // dd
			"w": wordNode,      // dw
			"i": innerModifier, // diw
		},
	}

	// --- 4. Normal Mode Graph ---
	normalGraph := map[string]*KeyNode{
		// Navigation
		"h": {Name: "h", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			for i := 0; i < ctx.Count; i++ {
				ed.MoveCursorLeft()
			}
		}},
		"j": {Name: "j", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			for i := 0; i < ctx.Count; i++ {
				ed.MoveCursorDown()
			}
		}},
		"k": {Name: "k", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			for i := 0; i < ctx.Count; i++ {
				ed.MoveCursorUp()
			}
		}},
		"l": {Name: "l", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			for i := 0; i < ctx.Count; i++ {
				ed.MoveCursorRight()
			}
		}},

		// Mode switching
		"i": {Name: "i", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			ed.SetModeInt(ModeInsert)
		}},
		":": {Name: ":", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			ed.SetModeInt(ModeCommand)
		}},

		// Modifications
		"x": {Name: "x", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			for i := 0; i < ctx.Count; i++ {
				ed.DeleteText()
			}
		}},
		"u": {Name: "u", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			for i := 0; i < ctx.Count; i++ {
				ed.Undo()
			}
		}},

		// Verbs
		"d": deleteVerb,
	}

	// --- 5. Insert Mode Graph ---
	insertGraph := map[string]*KeyNode{
		"<Esc>": {Name: "<Esc>", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			ed.SetModeInt(ModeNormal)
		}},
		"<BS>": {Name: "<BS>", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			ed.DeleteText()
		}},
		"<Enter>": {Name: "<Enter>", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			ed.InsertText([]byte("\n"))
		}},
	}

	// --- 6. Command Mode Graph ---
	commandGraph := map[string]*KeyNode{
		"<Esc>": {Name: "<Esc>", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			ed.SetModeInt(ModeNormal)
		}},
		"<BS>": {Name: "<BS>", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			ed.BackspaceCommandBuffer()
		}},
		"<Enter>": {Name: "<Enter>", Type: TypeDirect, Execute: func(ed EditorInterface, ctx CommandContext) {
			cmd := ed.GetCommandBuffer()
			_ = ed.ExecuteCommand(cmd)
		}},
	}

	return map[int]map[string]*KeyNode{
		ModeNormal:  normalGraph,
		ModeInsert:  insertGraph,
		ModeCommand: commandGraph,
	}
}

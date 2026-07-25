package keymap

import (
	"strconv"
)

type ActionType int

const (
	TypeVerb ActionType = iota
	TypeModifier
	TypeNoun
	TypeDirect
)

// CommandContext holds the accumulated state of a key sequence
type CommandContext struct {
	Count    int
	Verb     string
	Modifier string
	Noun     string
	Raw      string
}

// ActionFunc represents the execution logic for a key sequence
type ActionFunc func(ed EditorInterface, ctx CommandContext)

// EditorInterface abstracts the editor actions needed by key commands
type EditorInterface interface {
	MoveCursorLeft()
	MoveCursorRight()
	MoveCursorUp()
	MoveCursorDown()
	InsertText(data []byte)
	DeleteText()
	Undo() bool
	SetModeInt(m int)
	GetModeInt() int
	AppendCommandBuffer(r rune)
	BackspaceCommandBuffer()
	ClearCommandBuffer()
	ExecuteCommand(cmd string) error
	GetCommandBuffer() string
}

// KeyNode represents a node in the verb-modifier-noun graph
type KeyNode struct {
	Name               string
	Type               ActionType
	Execute            ActionFunc
	AllowedTransitions map[string]*KeyNode
}

// Router processes incoming keypresses through the graph state machine
type Router struct {
	ctx           CommandContext
	currentNode   *KeyNode
	activeGraph   map[string]*KeyNode
	ModeGraphs    map[int]map[string]*KeyNode
	PendingBuffer string
}

func NewRouter() *Router {
	return &Router{
		ModeGraphs: make(map[int]map[string]*KeyNode),
	}
}

func (r *Router) RegisterModeGraph(mode int, graph map[string]*KeyNode) {
	r.ModeGraphs[mode] = graph
}

// Reset clears accumulated context and returns router to root state
func (r *Router) Reset() {
	r.ctx = CommandContext{}
	r.currentNode = nil
	r.activeGraph = nil
	r.PendingBuffer = ""
}

// HandleKey processes a single keystroke string (e.g. "h", "j", "d", "w", "<Esc>", "<Enter>", "<BS>")
func (r *Router) HandleKey(ed EditorInterface, key string) bool {
	currentMode := ed.GetModeInt()
	modeGraph, exists := r.ModeGraphs[currentMode]
	if !exists {
		r.Reset()
		return false
	}

	// 1. Determine active transition map
	var transitions map[string]*KeyNode
	if r.currentNode != nil && r.currentNode.AllowedTransitions != nil {
		transitions = r.currentNode.AllowedTransitions
	} else {
		transitions = modeGraph
	}

	// 2. Check for numeric count prefix if at root in Normal mode (e.g., 3dw or 5j)
	if r.currentNode == nil && (key >= "1" && key <= "9" || (key == "0" && r.ctx.Count > 0)) {
		val, _ := strconv.Atoi(key)
		if r.ctx.Count == 0 {
			r.ctx.Count = val
		} else {
			r.ctx.Count = r.ctx.Count*10 + val
		}
		r.PendingBuffer += key
		return true
	}

	// Default count to 1 if not specified
	if r.ctx.Count == 0 {
		r.ctx.Count = 1
	}

	// 3. Lookup key in transitions
	node, found := transitions[key]
	if !found {
		// Handle mode-specific fallbacks (e.g. typing text in Insert mode or typing commands in Command mode)
		r.Reset()
		const (
			ModeInsert  = 1
			ModeCommand = 3
		)
		if currentMode == ModeInsert {
			ed.InsertText([]byte(key))
			return true
		} else if currentMode == ModeCommand {
			for _, r := range key {
				ed.AppendCommandBuffer(r)
			}
			return true
		}
		return false
	}

	// 4. Record node info into CommandContext
	r.PendingBuffer += key
	r.ctx.Raw = r.PendingBuffer

	switch node.Type {
	case TypeVerb:
		r.ctx.Verb = node.Name
	case TypeModifier:
		r.ctx.Modifier = node.Name
	case TypeNoun, TypeDirect:
		r.ctx.Noun = node.Name
	}

	// 5. If terminal node with execution logic, run action and reset
	if node.Execute != nil {
		node.Execute(ed, r.ctx)
		r.Reset()
		return true
	}

	// 6. Incomplete sequence (e.g. pressed 'd' or 'di'), hold state for next key
	r.currentNode = node
	return true
}

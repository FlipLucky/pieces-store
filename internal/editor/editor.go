package editor

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fliplucky/pieces-store/internal/editor/keymap"
	"github.com/fliplucky/pieces-store/internal/piecestore"
)

type Editor struct {
	mu              sync.RWMutex
	store           *piecestore.Store
	runeCalculator  *RuneCalculator
	virtualGrid     *VirtualGrid
	cursor          *Cursor
	keymapRouter    *keymap.Router
	isQuitRequested bool
	change          chan struct{}
}

func initRouter() *keymap.Router {
	router := keymap.NewRouter()
	graphs := keymap.BuildDefaultGraphs()
	for mode, graph := range graphs {
		router.RegisterModeGraph(mode, graph)
	}
	return router
}

func NewEditor(initialText string) *Editor {
	ed := &Editor{
		store:          piecestore.NewPieceStore([]byte(initialText)),
		runeCalculator: NewRuneCalculator(),
		virtualGrid:    NewVirtualGrid(),
		cursor:         NewCursor(),
		keymapRouter:   initRouter(),
		change:         make(chan struct{}, 1),
	}

	// Ticker simulation - only starts if SIMULATE=true is set in the environment
	if os.Getenv("SIMULATE") == "true" {
		go func() {
			chars := []byte(" Typing simulation active! Now powered by the real Piece Store under the hood.")
			idx := 0
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for range ticker.C {
				ed.mu.Lock()
				if idx >= len(chars) {
					// Reset to initial text when simulation reaches the end
					ed.store = piecestore.NewPieceStore([]byte(initialText))
					ed.cursor.Update(0, 0, 0)
					idx = 0
				} else {
					// Calculate total length and insert the character at the end of the buffer
					currentLen := ed.store.Len()
					ed.store.Insert(currentLen, []byte{chars[idx]})
					idx++

					// Update cursor position to the end of the text
					newOffset := ed.store.Len()
					gridPos := ed.runeCalculator.ByteOffsetToPosition(ed.store, newOffset)
					screenPos := ed.virtualGrid.GetScreenPosition(gridPos)
					ed.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
				}
				ed.mu.Unlock()
				ed.notifyChange()
			}
		}()
	}

	return ed
}

func NewEditorFromFile(filePath string) (*Editor, error) {
	store, err := piecestore.NewPieceStoreFromFile(filePath)
	if err != nil {
		return nil, err
	}
	return &Editor{
		store:          store,
		runeCalculator: NewRuneCalculator(),
		virtualGrid:    NewVirtualGrid(),
		cursor:         NewCursor(),
		keymapRouter:   initRouter(),
		change:         make(chan struct{}, 1),
	}, nil
}

func (e *Editor) GetText() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.store.CombinePieces()
}

func (e *Editor) GetFilePath() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.store.FilePath
}

func (e *Editor) SaveFile() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.store.Save()
}

func (e *Editor) SaveFileAs(filePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.store.SaveAs(filePath)
}

func (e *Editor) OpenFile(filePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	newStore, err := piecestore.NewPieceStoreFromFile(filePath)
	if err != nil {
		return err
	}
	e.store = newStore
	e.cursor.Update(0, 0, 0)
	e.notifyChange()
	return nil
}

func (e *Editor) GetCursor() Cursor {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return *e.cursor
}

func (e *Editor) ChangeChan() <-chan struct{} {
	return e.change
}

func (e *Editor) notifyChange() {
	select {
	case e.change <- struct{}{}:
	default:
	}
}

// --- Navigation / Vim Movement Engine ---

func (e *Editor) MoveCursorLeft() {
	e.mu.Lock()
	defer e.mu.Unlock()

	newOffset := e.runeCalculator.MoveLeft(e.store, e.cursor.ByteOffset)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.store, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)

	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange()
}

func (e *Editor) MoveCursorRight() {
	e.mu.Lock()
	defer e.mu.Unlock()

	newOffset := e.runeCalculator.MoveRight(e.store, e.cursor.ByteOffset)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.store, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)

	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange()
}

func (e *Editor) MoveCursorUp() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cursor.Row <= 0 {
		return
	}

	targetPos := Position{Row: e.cursor.Row - 1, Col: e.cursor.Col}
	newOffset := e.runeCalculator.PositionToByteOffset(e.store, targetPos)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.store, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)

	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange()
}

func (e *Editor) MoveCursorDown() {
	e.mu.Lock()
	defer e.mu.Unlock()

	targetPos := Position{Row: e.cursor.Row + 1, Col: e.cursor.Col}
	newOffset := e.runeCalculator.PositionToByteOffset(e.store, targetPos)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.store, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)

	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange()
}

// --- Text Modification Engine ---

func (e *Editor) InsertText(data []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(data) == 0 {
		return
	}

	e.store.Insert(e.cursor.ByteOffset, data)

	// Recalculate and update cursor position after insertion
	newOffset := e.cursor.ByteOffset + len(data)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.store, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)

	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange()
}

func (e *Editor) DeleteText() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cursor.ByteOffset <= 0 {
		return
	}

	newOffset := e.runeCalculator.MoveLeft(e.store, e.cursor.ByteOffset)
	lengthToDelete := e.cursor.ByteOffset - newOffset

	e.store.Delete(newOffset, lengthToDelete)

	// Recalculate cursor position after deletion
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.store, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)

	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange()
}

func (e *Editor) SetMode(m Mode) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cursor.SetMode(m)
	if m != ModeCommand {
		e.cursor.CommandBuffer = ""
	}
	e.notifyChange()
}

// --- Command Mode & Buffer Operations ---

var (
	ErrQuit           = errors.New("quit requested")
	ErrUnknownCommand = errors.New("unknown command")
)

func (e *Editor) AppendCommandBuffer(r rune) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cursor.CommandBuffer += string(r)
	e.notifyChange()
}

func (e *Editor) BackspaceCommandBuffer() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.cursor.CommandBuffer) > 0 {
		runes := []rune(e.cursor.CommandBuffer)
		e.cursor.CommandBuffer = string(runes[:len(runes)-1])
	}
	e.notifyChange()
}

func (e *Editor) ClearCommandBuffer() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cursor.CommandBuffer = ""
	e.notifyChange()
}

func (e *Editor) ExecuteCommand(rawCmd string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	trimmed := strings.TrimSpace(rawCmd)
	if strings.HasPrefix(trimmed, ":") {
		trimmed = trimmed[1:]
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		e.cursor.SetMode(ModeNormal)
		e.cursor.CommandBuffer = ""
		return nil
	}

	command := parts[0]
	var err error

	switch command {
	case "w", "write":
		if len(parts) > 1 {
			err = e.store.SaveAs(parts[1])
		} else {
			err = e.store.Save()
		}
	case "e", "edit":
		if len(parts) > 1 {
			newStore, openErr := piecestore.NewPieceStoreFromFile(parts[1])
			if openErr != nil {
				err = openErr
			} else {
				e.store = newStore
				e.cursor.Update(0, 0, 0)
			}
		} else {
			err = errors.New("no file specified for :e")
		}
	case "q", "quit":
		err = ErrQuit
		e.isQuitRequested = true
	case "wq":
		if saveErr := e.store.Save(); saveErr != nil {
			err = saveErr
		} else {
			err = ErrQuit
			e.isQuitRequested = true
		}
	default:
		err = ErrUnknownCommand
	}

	e.cursor.SetMode(ModeNormal)
	e.cursor.CommandBuffer = ""
	e.notifyChange()
	return err
}

func (e *Editor) IsQuitRequested() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isQuitRequested
}

func (e *Editor) Undo() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	success := e.store.Undo()
	if success {
		newOffset := e.cursor.ByteOffset
		if newOffset > e.store.Len() {
			newOffset = e.store.Len()
		}
		gridPos := e.runeCalculator.ByteOffsetToPosition(e.store, newOffset)
		screenPos := e.virtualGrid.GetScreenPosition(gridPos)

		e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
		e.notifyChange()
	}
	return success
}

func (e *Editor) SetModeInt(m int) {
	e.SetMode(Mode(m))
}

func (e *Editor) GetModeInt() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return int(e.cursor.Mode)
}

func (e *Editor) GetCommandBuffer() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cursor.CommandBuffer
}

func (e *Editor) HandleKey(key string) bool {
	return e.keymapRouter.HandleKey(e, key)
}

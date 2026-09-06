package editor

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fliplucky/pieces-store/internal/editor/keymap"
	"github.com/fliplucky/pieces-store/internal/piecetable"
	"github.com/fliplucky/pieces-store/internal/viewmanager"
)

type Editor struct {
	mu              sync.RWMutex
	table           *piecetable.Table
	runeCalculator  *viewmanager.RuneCalculator
	virtualGrid     *viewmanager.VirtualGrid
	cursor          *viewmanager.Cursor
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
		table:          piecetable.NewPieceTable([]byte(initialText)),
		runeCalculator: viewmanager.NewRuneCalculator(),
		virtualGrid:    viewmanager.NewVirtualGrid(),
		cursor:         viewmanager.NewCursor(),
		keymapRouter:   initRouter(),
		change:         make(chan struct{}, 1),
	}

	// Ticker simulation - only starts if SIMULATE=true is set in the environment
	if os.Getenv("SIMULATE") == "true" {
		go func() {
			chars := []byte(" Typing simulation active! Now powered by the real Piece Table under the hood.")
			idx := 0
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for range ticker.C {
				ed.mu.Lock()
				if idx >= len(chars) {
					// Reset to initial text when simulation reaches the end
					ed.table = piecetable.NewPieceTable([]byte(initialText))
					ed.cursor.Update(0, 0, 0)
					idx = 0
				} else {
					// Calculate total length and insert the character at the end of the buffer
					currentLen := ed.table.Len()
					ed.table.Insert(currentLen, []byte{chars[idx]})
					idx++

					// Update cursor position to the end of the text
					newOffset := ed.table.Len()
					gridPos := ed.runeCalculator.ByteOffsetToPosition(ed.table, newOffset)
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
	table, err := piecetable.NewPieceTableFromFile(filePath)
	if err != nil {
		return nil, err
	}
	return &Editor{
		table:          table,
		runeCalculator: viewmanager.NewRuneCalculator(),
		virtualGrid:    viewmanager.NewVirtualGrid(),
		cursor:         viewmanager.NewCursor(),
		keymapRouter:   initRouter(),
		change:         make(chan struct{}, 1),
	}, nil
}

func (e *Editor) GetText() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.table.CombinePieces()
}

func (e *Editor) GetFilePath() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.table.FilePath
}

func (e *Editor) SaveFile() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.table.Save()
}

func (e *Editor) SaveFileAs(filePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.table.SaveAs(filePath)
}

func (e *Editor) OpenFile(filePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	newTable, err := piecetable.NewPieceTableFromFile(filePath)
	if err != nil {
		return err
	}
	e.table = newTable
	e.cursor.Update(0, 0, 0)
	e.notifyChange()
	return nil
}

func (e *Editor) GetCursor() viewmanager.Cursor {
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

	newOffset := e.runeCalculator.MoveLeft(e.table, e.cursor.ByteOffset)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.table, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)

	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange()
}

func (e *Editor) MoveCursorRight() {
	e.mu.Lock()
	defer e.mu.Unlock()

	newOffset := e.runeCalculator.MoveRight(e.table, e.cursor.ByteOffset)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.table, newOffset)
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

	targetPos := viewmanager.Position{Row: e.cursor.Row - 1, Col: e.cursor.Col}
	newOffset := e.runeCalculator.PositionToByteOffset(e.table, targetPos)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.table, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)

	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange()
}

func (e *Editor) MoveCursorDown() {
	e.mu.Lock()
	defer e.mu.Unlock()

	targetPos := viewmanager.Position{Row: e.cursor.Row + 1, Col: e.cursor.Col}
	newOffset := e.runeCalculator.PositionToByteOffset(e.table, targetPos)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.table, newOffset)
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

	e.table.Insert(e.cursor.ByteOffset, data)

	// Recalculate and update cursor position after insertion
	newOffset := e.cursor.ByteOffset + len(data)
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.table, newOffset)
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

	newOffset := e.runeCalculator.MoveLeft(e.table, e.cursor.ByteOffset)
	lengthToDelete := e.cursor.ByteOffset - newOffset

	e.table.Delete(newOffset, lengthToDelete)

	// Recalculate cursor position after deletion
	gridPos := e.runeCalculator.ByteOffsetToPosition(e.table, newOffset)
	screenPos := e.virtualGrid.GetScreenPosition(gridPos)

	e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
	e.notifyChange()
}

func (e *Editor) SetMode(m viewmanager.Mode) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cursor.SetMode(m)
	if m != viewmanager.ModeCommand {
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
		e.cursor.SetMode(viewmanager.ModeNormal)
		e.cursor.CommandBuffer = ""
		return nil
	}

	command := parts[0]
	var err error

	switch command {
	case "w", "write":
		if len(parts) > 1 {
			err = e.table.SaveAs(parts[1])
		} else {
			err = e.table.Save()
		}
	case "e", "edit":
		if len(parts) > 1 {
			newTable, openErr := piecetable.NewPieceTableFromFile(parts[1])
			if openErr != nil {
				err = openErr
			} else {
				e.table = newTable
				e.cursor.Update(0, 0, 0)
			}
		} else {
			err = errors.New("no file specified for :e")
		}
	case "q", "quit":
		err = ErrQuit
		e.isQuitRequested = true
	case "wq":
		if saveErr := e.table.Save(); saveErr != nil {
			err = saveErr
		} else {
			err = ErrQuit
			e.isQuitRequested = true
		}
	default:
		err = ErrUnknownCommand
	}

	e.cursor.SetMode(viewmanager.ModeNormal)
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

	success := e.table.Undo()
	if success {
		newOffset := e.cursor.ByteOffset
		if newOffset > e.table.Len() {
			newOffset = e.table.Len()
		}
		gridPos := e.runeCalculator.ByteOffsetToPosition(e.table, newOffset)
		screenPos := e.virtualGrid.GetScreenPosition(gridPos)

		e.cursor.Update(newOffset, screenPos.Row, screenPos.Col)
		e.notifyChange()
	}
	return success
}

func (e *Editor) SetModeInt(m int) {
	e.SetMode(viewmanager.Mode(m))
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

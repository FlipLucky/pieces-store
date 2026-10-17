package editor

import (
	clipboardregister "github.com/fliplucky/pieces-store/internal/clipboard-register"
	"github.com/fliplucky/pieces-store/internal/keyengine"
	"github.com/fliplucky/pieces-store/internal/langdetect"
	"github.com/fliplucky/pieces-store/internal/piecetable"
	"github.com/fliplucky/pieces-store/internal/types"
	"github.com/fliplucky/pieces-store/internal/viewmanager"
)

// File-related Editor methods — construction from disk, opening a
// different file into an already-running Editor, saving, and language
// detection. Split out of editor.go 2026-10-17 (T8), a pure move with no
// logic change.

func NewEditorFromFile(filePath string) (*Editor, error) {
	table, err := piecetable.NewPieceTableFromFile(filePath)
	if err != nil {
		return nil, err
	}
	e := &Editor{
		table:             table,
		virtualGrid:       viewmanager.NewVirtualGrid(),
		cursor:            NewCursor(),
		commandContext:    keyengine.CreateCommandContext(),
		clipboardRegister: &clipboardregister.ClipboardRegister{},
		change:            make(chan ChangeEvent, 1),
	}
	e.startAsyncLoop()
	e.setupHighlighterLocked()
	e.ensureLSPForCurrentBufferLocked()
	return e, nil
}

func (e *Editor) GetFilePath() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.table.FilePath
}

// Language returns the buffer's detected language, based on its file
// path's extension (langdetect.Detect) — LanguagePlainText for an
// unsaved buffer or an unrecognized extension. Computed fresh each call
// rather than cached, so it's always consistent with the current
// FilePath with no staleness to manage across OpenFile/:e/SaveAs.
func (e *Editor) Language() types.Language {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return langdetect.Detect(e.table.FilePath)
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
	e.setupHighlighterLocked()
	e.ensureLSPForCurrentBufferLocked()
	e.notifyChange(piecetable.Edit{}) // whole-document replacement, not an incremental delta
	return nil
}

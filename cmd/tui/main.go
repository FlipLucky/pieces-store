package main

import (
	"log"
	"os"

	"github.com/fliplucky/daeplex/internal/editor"
	tuibase "github.com/fliplucky/daeplex/internal/tui-base"
	"github.com/fliplucky/daeplex/internal/workspace"
)

func main() {
	var ed *editor.Editor
	var err error

	if len(os.Args) > 1 {
		ed, err = editor.NewEditorFromFile(os.Args[1])
		if err != nil {
			log.Fatalf("failed to open file %s: %v", os.Args[1], err)
		}
	} else {
		ed = editor.NewEditor("")
	}

	if err := tuibase.Boot(workspace.New(ed)); err != nil {
		log.Fatalf("failed to boot TUI: %v", err)
	}
}

package main

import (
	"log"
	"os"

	"github.com/fliplucky/pieces-store/internal/editor"
	guibase "github.com/fliplucky/pieces-store/internal/gui-base"
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
		ed = editor.NewEditor("Hello World")
	}

	app := guibase.CreateApp(ed)
	app.Run()
}

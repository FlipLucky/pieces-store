package main

import (
	guibase "github.com/fliplucky/pieces-store/internal/gui-base"
	"github.com/fliplucky/pieces-store/internal/editor"
)

func main() {
	ed := editor.NewEditor("Hello World")

	app := guibase.CreateApp(ed)
	app.Run()
}

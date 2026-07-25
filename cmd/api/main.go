package main

import (
	"fmt"

	// guibase "github.com/fliplucky/pieces-store/internal/gui-base"
	"github.com/fliplucky/pieces-store/internal/piecestore"
)

func main() {
	store := piecestore.NewPieceStore([]byte("Hello World"))
	store.Insert(5, []byte(" Amazing"))
	fmt.Println(store.GetText())
	store.PrintDebug()
	store.Delete(2, 15)
	fmt.Println(store.GetText())
	store.PrintDebug()
}

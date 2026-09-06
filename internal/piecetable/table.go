// Package piecetable is the heart of the editor, being the source of the text.
// To mutate, use insert and delete
// Use CombinePieces to get the full text, while get range is to get a slice for a vieuwport
// FindPieceAt is to locate which piece contains the offset, while GetRuneAt is to retrieve a specific character from the table.
package piecetable

import (
	"sync"
)

type Table struct {
	mu       sync.RWMutex
	FilePath string
	Master   []byte
	Add      []byte
	Pieces   []Piece
	History  []State
}

type Piece struct {
	BufferType BufferType
	Start      int
	Length     int
}

type State struct {
	Pieces []Piece
	AddLen int
}

type BufferType string

const (
	Master BufferType = "MASTER"
	Add    BufferType = "ADD"
)

func NewPieceTable(data []byte) *Table {
	return &Table{
		Master:  data,
		Add:     []byte{},
		Pieces:  []Piece{{BufferType: Master, Start: 0, Length: len(data)}},
		History: []State{},
	}
}

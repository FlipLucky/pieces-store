package piecestore

import "fmt"

func (s *Store) PrintDebug() {
	for i, p := range s.Pieces {
		fmt.Printf("Piece %d: Type=%s, Start=%d, Len=%d\n", i, p.BufferType, p.Start, p.Length)
	}
}

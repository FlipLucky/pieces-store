package piecestore

import "sync"

type Store struct {
	mu      sync.RWMutex
	Master  []byte
	Add     []byte
	Pieces  []Piece
	History []State
}

type Piece struct {
	BufferType BufferType
	Start      int
	Length     int
}

type State struct {
	Pieces []Piece
	Add    []byte
}

type BufferType string

const (
	Master BufferType = "MASTER"
	Add    BufferType = "ADD"
)

func (s *Store) FindPieceAt(offset int) (int, int) {
	currentOffset := 0
	for i, p := range s.Pieces {
		if offset >= currentOffset && offset < currentOffset+p.Length {
			return i, offset - currentOffset
		}
		currentOffset += p.Length
	}
	// Handle "Append at end" case
	return len(s.Pieces), 0
}

func (s *Store) Insert(offset int, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1. Save history (Good job on this)
	s.History = append(
		s.History,
		State{
			Pieces: append([]Piece{}, s.Pieces...),
			Add:    append([]byte{}, s.Add...)})

	// 2. Add to AddBuffer
	addStart := len(s.Add)
	s.Add = append(s.Add, data...)

	// 3. Find the split point
	pieceIdx, offsetInPiece := s.FindPieceAt(offset)

	// 4. Create the new pieces
	target := s.Pieces[pieceIdx]

	// Split the target into two: left of cursor, right of cursor
	left := Piece{BufferType: target.BufferType, Start: target.Start, Length: offsetInPiece}
	right := Piece{BufferType: target.BufferType, Start: target.Start + offsetInPiece, Length: target.Length - offsetInPiece}
	middle := Piece{BufferType: Add, Start: addStart, Length: len(data)}

	// 5. Replace the old piece with the new ones
	// Using slice concatenation
	newPieces := append([]Piece{}, s.Pieces[:pieceIdx]...)  // Everything before
	newPieces = append(newPieces, left, middle, right)      // Insert the 3-piece sandwich
	newPieces = append(newPieces, s.Pieces[pieceIdx+1:]...) // Everything after

	s.Pieces = newPieces
}

func NewPieceStore(data []byte) *Store {
	return &Store{
		Master:  data,
		Add:     []byte{},
		Pieces:  []Piece{{BufferType: Master, Start: 0, Length: len(data)}},
		History: []State{},
	}
}

func (s *Store) GetText() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []byte
	for _, p := range s.Pieces {
		if p.BufferType == Master {
			result = append(result, s.Master[p.Start:p.Start+p.Length]...)
		} else if p.BufferType == Add {
			result = append(result, s.Add[p.Start:p.Start+p.Length]...)
		}
	}
	return string(result)
}

func (s *Store) Delete(start, length int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var newPieces []Piece
	end := start + length

	current := 0
	for _, p := range s.Pieces {
		pieceEnd := current + p.Length

		if pieceEnd <= start {
			// CASE 1: Piece is entirely before the delete range
			newPieces = append(newPieces, p)
		} else if current >= end {
			// CASE 2: Piece is entirely after the delete range
			newPieces = append(newPieces, p)
		} else if current < start {
			// CASE 3: Piece contains the START of the deletion
			newPieces = append(newPieces, Piece{BufferType: p.BufferType, Start: p.Start, Length: start - current})
		} else if pieceEnd > end {
			// CASE 4: Piece contains the END of the deletion
			remaining := pieceEnd - end
			newPieces = append(newPieces, Piece{BufferType: p.BufferType, Start: p.Start + (end - current), Length: remaining})
		}

		current += p.Length
	}
	s.Pieces = newPieces
}

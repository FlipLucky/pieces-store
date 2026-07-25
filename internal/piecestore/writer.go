package piecestore

func (s *Store) Insert(offset int, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(data) == 0 {
		return
	}
	// 1. Save history
	s.History = append(
		s.History,
		State{
			Pieces: append([]Piece{}, s.Pieces...),
			AddLen: len(s.Add),
		})

	// 2. Add to AddBuffer
	addStart := len(s.Add)
	s.Add = append(s.Add, data...)

	// 3. Find the split point
	pieceIdx, offsetInPiece := s.FindPieceAt(offset)

	// 4. Create the middle piece
	middle := Piece{BufferType: Add, Start: addStart, Length: len(data)}

	// 5. Replace the old piece with the new ones
	var newPieces []Piece
	if pieceIdx == len(s.Pieces) {
		newPieces = append([]Piece{}, s.Pieces...)
		newPieces = append(newPieces, middle)
	} else {
		newPieces = append([]Piece{}, s.Pieces[:pieceIdx]...)
		target := s.Pieces[pieceIdx]
		if offsetInPiece == 0 {
			newPieces = append(newPieces, middle, target)
		} else {
			left := Piece{BufferType: target.BufferType, Start: target.Start, Length: offsetInPiece}
			right := Piece{BufferType: target.BufferType, Start: target.Start + offsetInPiece, Length: target.Length - offsetInPiece}
			newPieces = append(newPieces, left, middle, right)
		}
		newPieces = append(newPieces, s.Pieces[pieceIdx+1:]...)
	}

	s.Pieces = newPieces
	s.coalesceUnlocked()
}

func (s *Store) Delete(start, length int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if length <= 0 {
		return
	}

	// Save history
	s.History = append(
		s.History,
		State{
			Pieces: append([]Piece{}, s.Pieces...),
			AddLen: len(s.Add),
		})

	var newPieces []Piece
	end := start + length

	current := 0
	for _, p := range s.Pieces {
		pieceEnd := current + p.Length

		if pieceEnd <= start || current >= end {
			// Piece is entirely outside the delete range
			newPieces = append(newPieces, p)
		} else {
			// Piece overlaps with the delete range
			if current < start {
				// Keep left part
				newPieces = append(newPieces, Piece{
					BufferType: p.BufferType,
					Start:      p.Start,
					Length:     start - current,
				})
			}
			if pieceEnd > end {
				// Keep right part
				newPieces = append(newPieces, Piece{
					BufferType: p.BufferType,
					Start:      p.Start + (end - current),
					Length:     pieceEnd - end,
				})
			}
		}

		current += p.Length
	}
	s.Pieces = newPieces
	s.coalesceUnlocked()
}

func (s *Store) Coalesce() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.coalesceUnlocked()
}

func (s *Store) coalesceUnlocked() {
	if len(s.Pieces) <= 1 {
		return
	}

	coalesced := make([]Piece, 0, len(s.Pieces))
	current := s.Pieces[0]

	for i := 1; i < len(s.Pieces); i++ {
		next := s.Pieces[i]
		if current.BufferType == next.BufferType && current.Start+current.Length == next.Start {
			current.Length += next.Length
		} else {
			coalesced = append(coalesced, current)
			current = next
		}
	}
	coalesced = append(coalesced, current)
	s.Pieces = coalesced
}

func (s *Store) Undo() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.History) == 0 {
		return false
	}
	lastState := s.History[len(s.History)-1]
	s.History = s.History[:len(s.History)-1]

	s.Pieces = lastState.Pieces
	s.Add = s.Add[:lastState.AddLen]
	return true
}

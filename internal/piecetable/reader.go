package piecetable

import "unicode/utf8"

func (s *Table) FindPieceAt(offset int) (int, int) {
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

func (s *Table) GetRuneAt(offset int) (rune, int) {
	if offset < 0 || offset >= s.Len() {
		return utf8.RuneError, 0
	}
	pieceIdx, offsetInPiece := s.FindPieceAt(offset)
	if pieceIdx >= len(s.Pieces) {
		return utf8.RuneError, 0
	}
	piece := s.Pieces[pieceIdx]
	var source []byte
	if piece.BufferType == Master {
		source = s.Master
	} else {
		source = s.Add
	}

	start := piece.Start + offsetInPiece
	end := piece.Start + piece.Length
	return utf8.DecodeRune(source[start:end])
}

func (s *Table) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := 0
	for _, p := range s.Pieces {
		total += p.Length
	}
	return total
}

func (s *Table) CombinePieces() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []byte
	for _, p := range s.Pieces {
		switch p.BufferType {
		case Master:
			result = append(result, s.Master[p.Start:p.Start+p.Length]...)
		case Add:
			result = append(result, s.Add[p.Start:p.Start+p.Length]...)
		}
	}
	return string(result)
}

func (s *Table) GetText() string {
	return s.CombinePieces()
}

func (s *Table) GetRange(start, end int) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []byte
	currentOffset := 0
	for _, p := range s.Pieces {

		relativeStart := max(0, start-currentOffset)
		relativeEnd := min(p.Length, end-currentOffset)

		bufferStart := p.Start + relativeStart
		bufferEnd := p.Start + relativeEnd
		if p.BufferType == Master {
			result = append(result, s.Master[bufferStart:bufferEnd]...)
		} else {
			result = append(result, s.Add[bufferStart:bufferEnd]...)
		}
		currentOffset += p.Length

		if currentOffset >= end {
			break
		}
	}
	return result

}

package piecestore

import (
	"errors"
	"os"
)

var ErrNoFilePath = errors.New("no file path specified")

func NewPieceStoreFromFile(filePath string) (*Store, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	store := NewPieceStore(data)
	store.FilePath = filePath
	return store, nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	filePath := s.FilePath
	s.mu.RUnlock()

	if filePath == "" {
		return ErrNoFilePath
	}
	return s.SaveAs(filePath)
}

func (s *Store) SaveAs(filePath string) error {
	content := s.CombinePieces()
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.FilePath = filePath
	s.mu.Unlock()
	return nil
}

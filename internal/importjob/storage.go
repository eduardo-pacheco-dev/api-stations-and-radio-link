package importjob

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrFileTooLarge indicates that an uploaded spreadsheet exceeds the limit.
var ErrFileTooLarge = errors.New("import file is too large")

// Storage persists uploaded spreadsheets on disk.
type Storage struct {
	dir      string
	maxBytes int64
}

// NewStorage creates a storage directory.
func NewStorage(dir string, maxBytes int64) (*Storage, error) {
	if dir == "" {
		return nil, fmt.Errorf("import directory must not be empty")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("import max bytes must be positive")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create import directory: %w", err)
	}

	return &Storage{dir: dir, maxBytes: maxBytes}, nil
}

// Save copies an upload to disk and returns its path.
func (s *Storage) Save(id string, reader io.Reader) (string, error) {
	path := filepath.Join(s.dir, id+".xlsx")

	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create import file: %w", err)
	}

	written, copyErr := io.Copy(file, io.LimitReader(reader, s.maxBytes+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("save import file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close import file: %w", closeErr)
	}
	if written > s.maxBytes {
		_ = os.Remove(path)
		return "", ErrFileTooLarge
	}

	return path, nil
}

// Remove deletes a stored import file.
func (s *Storage) Remove(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}

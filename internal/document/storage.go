package document

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrFileTooLarge indicates that an uploaded document exceeds the limit.
var ErrFileTooLarge = errors.New("document file is too large")

// Storage persists document files on disk.
type Storage struct {
	dir      string
	maxBytes int64
}

// NewStorage creates a document storage directory.
func NewStorage(dir string, maxBytes int64) (*Storage, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("document directory must not be empty")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("document max bytes must be positive")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create document directory: %w", err)
	}

	return &Storage{dir: dir, maxBytes: maxBytes}, nil
}

// Save copies an uploaded file to disk and returns its path and size.
func (s *Storage) Save(id, originalName string, reader io.Reader) (string, int64, error) {
	extension := safeExtension(originalName)
	path := filepath.Join(s.dir, id+extension)

	file, err := os.Create(path)
	if err != nil {
		return "", 0, fmt.Errorf("create document file: %w", err)
	}

	written, copyErr := io.Copy(file, io.LimitReader(reader, s.maxBytes+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return "", 0, fmt.Errorf("save document file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", 0, fmt.Errorf("close document file: %w", closeErr)
	}
	if written > s.maxBytes {
		_ = os.Remove(path)
		return "", 0, ErrFileTooLarge
	}

	return path, written, nil
}

// Open opens a stored document file.
func (s *Storage) Open(path string) (*os.File, error) {
	return os.Open(path)
}

// Remove deletes a stored document file.
func (s *Storage) Remove(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}

func safeExtension(name string) string {
	extension := strings.ToLower(filepath.Ext(name))
	if len(extension) > 10 {
		return ".bin"
	}
	for _, char := range extension {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' {
			continue
		}

		return ".bin"
	}

	return extension
}

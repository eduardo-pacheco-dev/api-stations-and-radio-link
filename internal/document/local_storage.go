package document

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage persists document files on the local filesystem.
type LocalStorage struct {
	dir      string
	maxBytes int64
}

// NewLocalStorage creates a local document storage directory.
func NewLocalStorage(dir string, maxBytes int64) (*LocalStorage, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("document directory must not be empty")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("document max bytes must be positive")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create document directory: %w", err)
	}

	return &LocalStorage{dir: dir, maxBytes: maxBytes}, nil
}

// Save copies an uploaded file to disk and returns its path and size.
func (s *LocalStorage) Save(_ context.Context, id, originalName, _ string, reader io.Reader) (string, int64, error) {
	path := filepath.Join(s.dir, id+SafeExtension(originalName))

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
func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(key)
}

// Remove deletes a stored document file.
func (s *LocalStorage) Remove(_ context.Context, key string) error {
	if key != "" {
		return os.Remove(key)
	}

	return nil
}

// SafeExtension returns a sanitized file extension.
func SafeExtension(name string) string {
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

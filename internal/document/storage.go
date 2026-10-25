package document

import (
	"context"
	"errors"
	"io"
)

// ErrFileTooLarge indicates that an uploaded document exceeds the limit.
var ErrFileTooLarge = errors.New("document file is too large")

// Storage is the file storage port used by documents.
type Storage interface {
	Save(ctx context.Context, id, originalName, contentType string, reader io.Reader) (key string, size int64, err error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Remove(ctx context.Context, key string) error
}

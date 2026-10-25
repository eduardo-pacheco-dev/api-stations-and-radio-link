// Package documentstorage selects the configured document storage adapter.
package documentstorage

import (
	"fmt"
	"strings"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/r2storage"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/config"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/document"
)

// New creates the document storage adapter selected by DOCUMENT_STORAGE.
func New(cfg config.Document) (document.Storage, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Storage)) {
	case "", "local":
		return document.NewLocalStorage(cfg.Dir, cfg.MaxBytes)
	case "r2":
		return r2storage.New(r2storage.Config{
			Endpoint:        cfg.R2Endpoint,
			AccessKeyID:     cfg.R2AccessKeyID,
			SecretAccessKey: cfg.R2SecretKey,
			Bucket:          cfg.R2Bucket,
			Region:          cfg.R2Region,
			Prefix:          cfg.R2Prefix,
			MaxBytes:        cfg.MaxBytes,
		})
	default:
		return nil, fmt.Errorf("unsupported DOCUMENT_STORAGE %q", cfg.Storage)
	}
}

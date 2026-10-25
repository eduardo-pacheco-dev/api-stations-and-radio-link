package documentstorage_test

import (
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/documentstorage"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/config"
)

func TestNew_Local(t *testing.T) {
	t.Parallel()

	storage, err := documentstorage.New(config.Document{
		Storage:  "local",
		Dir:      t.TempDir(),
		MaxBytes: 1024,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if storage == nil {
		t.Fatal("New() returned nil storage")
	}
}

func TestNew_Unsupported(t *testing.T) {
	t.Parallel()

	_, err := documentstorage.New(config.Document{Storage: "s3"})
	if err == nil {
		t.Fatal("New() error = nil, want unsupported storage")
	}
}

func TestNew_R2RequiresConfiguration(t *testing.T) {
	t.Parallel()

	_, err := documentstorage.New(config.Document{Storage: "r2", MaxBytes: 1024})
	if err == nil {
		t.Fatal("New() error = nil, want missing R2 configuration")
	}
}

package document_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/document"
)

type fakeRepository struct {
	documents   map[string]document.Document
	ownerExists bool
	createErr   error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{documents: make(map[string]document.Document), ownerExists: true}
}

func (r *fakeRepository) Create(_ context.Context, record *document.Document) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.documents[record.ID] = *record
	return nil
}

func (r *fakeRepository) FindByID(_ context.Context, id string) (document.Document, error) {
	record, ok := r.documents[id]
	if !ok {
		return document.Document{}, document.ErrNotFound
	}
	return record, nil
}

func (r *fakeRepository) List(_ context.Context, _ document.ListFilter) ([]document.Document, int64, error) {
	items := make([]document.Document, 0, len(r.documents))
	for _, record := range r.documents {
		items = append(items, record)
	}
	return items, int64(len(items)), nil
}

func (r *fakeRepository) Update(_ context.Context, record *document.Document) error {
	r.documents[record.ID] = *record
	return nil
}

func (r *fakeRepository) Delete(_ context.Context, id string) error {
	delete(r.documents, id)
	return nil
}

func (r *fakeRepository) OwnerExists(context.Context, document.OwnerType, string) (bool, error) {
	return r.ownerExists, nil
}

func newService(t *testing.T, repo *fakeRepository) (*document.Service, *document.Storage) {
	t.Helper()

	storage, err := document.NewStorage(t.TempDir(), 1024)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	return document.NewService(repo, storage), storage
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	service, _ := newService(t, repo)

	record, err := service.Create(t.Context(), document.CreateInput{
		OwnerType:   document.OwnerTypeStation,
		OwnerID:     "TSHG01",
		Title:       "Licença",
		Category:    "licenca",
		Description: "Licença de operação",
		FileName:    "licenca.pdf",
		ContentType: "application/pdf",
	}, strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if record.ID == "" || record.OwnerID != "TSHG01" || record.SizeBytes != 5 {
		t.Errorf("record = %+v, want owner TSHG01 and size 5", record)
	}
	if _, err := os.Stat(record.FilePath); err != nil {
		t.Errorf("stored file: %v", err)
	}
	if len(repo.documents) != 1 {
		t.Errorf("repository documents = %d, want 1", len(repo.documents))
	}
}

func TestService_Create_OwnerNotFound(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	repo.ownerExists = false
	service, _ := newService(t, repo)

	_, err := service.Create(t.Context(), document.CreateInput{
		OwnerType: document.OwnerTypeRadioLink,
		OwnerID:   "RL-0001",
		Title:     "Documento",
		FileName:  "doc.pdf",
	}, strings.NewReader("hello"))

	if !errors.Is(err, document.ErrOwnerNotFound) {
		t.Fatalf("Create() error = %v, want ErrOwnerNotFound", err)
	}
}

func TestService_Create_Validation(t *testing.T) {
	t.Parallel()

	service, _ := newService(t, newFakeRepository())

	_, err := service.Create(t.Context(), document.CreateInput{
		OwnerType: document.OwnerTypeStation,
		OwnerID:   "TSHG01",
		FileName:  "doc.pdf",
	}, strings.NewReader("hello"))

	if !errors.Is(err, document.ErrValidation) {
		t.Fatalf("Create() error = %v, want ErrValidation", err)
	}
}

func TestService_UpdateAndDelete(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	service, _ := newService(t, repo)

	record, err := service.Create(t.Context(), document.CreateInput{
		OwnerType: document.OwnerTypeStation,
		OwnerID:   "TSHG01",
		Title:     "Original",
		FileName:  "doc.pdf",
	}, strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := service.Update(t.Context(), record.ID, document.UpdateInput{
		Title:       "Atualizado",
		Category:    "cat",
		Description: "desc",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != "Atualizado" {
		t.Errorf("title = %q, want Atualizado", updated.Title)
	}

	if err := service.Delete(t.Context(), record.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Stat(record.FilePath); !os.IsNotExist(err) {
		t.Errorf("file still exists after delete: %v", err)
	}
}

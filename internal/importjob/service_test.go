package importjob_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/importjob"
)

type fakeRepository struct {
	created *importjob.Job
	updated *importjob.Job
}

func (r *fakeRepository) Create(_ context.Context, job *importjob.Job) error {
	r.created = job
	job.CreatedAt = job.CreatedAt.Add(0)
	return nil
}

func (r *fakeRepository) Update(_ context.Context, job *importjob.Job) error {
	r.updated = job
	return nil
}

func (r *fakeRepository) FindByID(context.Context, string) (importjob.Job, error) {
	return importjob.Job{}, importjob.ErrNotFound
}

func (r *fakeRepository) List(context.Context, importjob.Filter) ([]importjob.Job, int64, error) {
	return nil, 0, nil
}

func (r *fakeRepository) Pending(context.Context) ([]importjob.Job, error) {
	return nil, nil
}

func (r *fakeRepository) FailRunning(context.Context, string) error {
	return nil
}

func TestStorage_RejectsLargeFile(t *testing.T) {
	t.Parallel()

	storage, err := importjob.NewStorage(t.TempDir(), 3)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	if _, err := storage.Save("job-1", strings.NewReader("1234")); err != importjob.ErrFileTooLarge {
		t.Fatalf("Save() error = %v, want ErrFileTooLarge", err)
	}
}

func TestService_CreateValidation(t *testing.T) {
	t.Parallel()

	storage, err := importjob.NewStorage(t.TempDir(), 1024)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	queue := make(chan string, 1)
	service := importjob.NewService(&fakeRepository{}, storage, queue, nil)

	if _, err := service.Create(t.Context(), "invalid", "insert", false, "x.xlsx", strings.NewReader("x")); err == nil {
		t.Fatal("Create() error = nil, want invalid entity")
	}
	if _, err := service.Create(t.Context(), importjob.EntityStations, "replace", false, "x.xlsx", strings.NewReader("x")); err == nil {
		t.Fatal("Create() error = nil, want invalid mode")
	}
}

func TestService_CreateEnqueues(t *testing.T) {
	t.Parallel()

	storage, err := importjob.NewStorage(t.TempDir(), 1024)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	queue := make(chan string, 1)
	repo := &fakeRepository{}
	service := importjob.NewService(repo, storage, queue, nil)

	job, err := service.Create(t.Context(), importjob.EntityStations, "upsert", true, "stations.xlsx", strings.NewReader("content"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if job.Status != importjob.StatusPending || job.Entity != importjob.EntityStations {
		t.Errorf("job = %+v, want pending station job", job)
	}
	if job.ID == "" || job.FilePath == "" {
		t.Errorf("job = %+v, want id and file path", job)
	}
	if _, err := os.Stat(job.FilePath); err != nil {
		t.Errorf("stored file: %v", err)
	}
	select {
	case id := <-queue:
		if id != job.ID {
			t.Errorf("queued id = %q, want %q", id, job.ID)
		}
	default:
		t.Fatal("job was not enqueued")
	}
}

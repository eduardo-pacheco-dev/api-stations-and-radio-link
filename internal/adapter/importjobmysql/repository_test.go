package importjobmysql_test

import (
	"testing"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/importjobmysql"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/importjob"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/testutil/sqlitedb"
)

func newJob(id string, status importjob.Status) importjob.Job {
	return importjob.Job{
		ID:       id,
		Entity:   importjob.EntityStations,
		Status:   status,
		Mode:     "upsert",
		FileName: "stations.xlsx",
		FilePath: "/tmp/stations.xlsx",
		Errors:   []importjob.Error{},
	}
}

func TestRepository_CreateFindUpdate(t *testing.T) {
	t.Parallel()

	repo := importjobmysql.NewRepository(sqlitedb.Open(t))
	job := newJob("job-1", importjob.StatusPending)

	if err := repo.Create(t.Context(), &job); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if job.CreatedAt.IsZero() {
		t.Error("created_at was not populated")
	}

	found, err := repo.FindByID(t.Context(), "job-1")
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Entity != importjob.EntityStations || found.Status != importjob.StatusPending {
		t.Errorf("found = %+v, want stations/pending", found)
	}

	found.Status = importjob.StatusRunning
	found.ProcessedRows = 500
	found.FailedRows = 2
	found.Errors = []importjob.Error{{Row: 3, Key: "TSHG01", Message: "invalid"}}
	if err := repo.Update(t.Context(), &found); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, err := repo.FindByID(t.Context(), "job-1")
	if err != nil {
		t.Fatalf("FindByID() after update error = %v", err)
	}
	if updated.Status != importjob.StatusRunning || updated.ProcessedRows != 500 {
		t.Errorf("updated = %+v, want running/500", updated)
	}
	if len(updated.Errors) != 1 || updated.Errors[0].Key != "TSHG01" {
		t.Errorf("errors = %+v, want one TSHG01 error", updated.Errors)
	}

	progress, err := repo.FindProgress(t.Context(), "job-1")
	if err != nil {
		t.Fatalf("FindProgress() error = %v", err)
	}
	if progress.Status != importjob.StatusRunning || progress.ProcessedRows != 500 || progress.FailedRows != 2 {
		t.Errorf("progress = %+v, want running/500/2", progress)
	}
}

func TestRepository_ListPendingAndFailRunning(t *testing.T) {
	t.Parallel()

	repo := importjobmysql.NewRepository(sqlitedb.Open(t))

	pending := newJob("pending-1", importjob.StatusPending)
	running := newJob("running-1", importjob.StatusRunning)
	failed := newJob("failed-1", importjob.StatusFailed)

	for _, job := range []*importjob.Job{&pending, &running, &failed} {
		if err := repo.Create(t.Context(), job); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	pendingJobs, err := repo.Pending(t.Context())
	if err != nil {
		t.Fatalf("Pending() error = %v", err)
	}
	if len(pendingJobs) != 1 || pendingJobs[0].ID != "pending-1" {
		t.Errorf("pending = %+v, want only pending-1", pendingJobs)
	}

	if err := repo.FailRunning(t.Context(), "interrupted"); err != nil {
		t.Fatalf("FailRunning() error = %v", err)
	}
	recovered, err := repo.FindByID(t.Context(), "running-1")
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if recovered.Status != importjob.StatusFailed || recovered.ErrorMessage != "interrupted" {
		t.Errorf("recovered = %+v, want failed/interrupted", recovered)
	}

	jobs, total, err := repo.List(t.Context(), importjob.Filter{Page: 1, PageSize: 10, Entity: "stations"}.Normalize())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 3 || len(jobs) != 3 {
		t.Errorf("list = total %d, items %d, want 3/3", total, len(jobs))
	}
}

// Package importjobmysql implements the import job repository with GORM.
package importjobmysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormmodel"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/importjob"
)

// Repository persists import jobs with GORM.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a GORM import job repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts an import job.
func (r *Repository) Create(ctx context.Context, job *importjob.Job) error {
	model, err := toModel(*job)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create import job %q: %w", job.ID, err)
	}

	*job = toDomain(model)

	return nil
}

// Update persists changes to an import job.
func (r *Repository) Update(ctx context.Context, job *importjob.Job) error {
	model, err := toModel(*job)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return fmt.Errorf("update import job %q: %w", job.ID, err)
	}

	*job = toDomain(model)

	return nil
}

// FindByID returns an import job by id.
func (r *Repository) FindByID(ctx context.Context, id string) (importjob.Job, error) {
	var model gormmodel.ImportJob

	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return importjob.Job{}, fmt.Errorf("find import job %q: %w", id, importjob.ErrNotFound)
	}
	if err != nil {
		return importjob.Job{}, fmt.Errorf("find import job %q: %w", id, err)
	}

	return toDomain(model), nil
}

// FindProgress returns a lightweight import job status by id.
func (r *Repository) FindProgress(ctx context.Context, id string) (importjob.Progress, error) {
	var model gormmodel.ImportJob

	err := r.db.WithContext(ctx).
		Select(
			"id", "entity", "status", "dry_run", "total_rows", "processed_rows",
			"valid_rows", "imported_rows", "updated_rows", "failed_rows", "updated_at",
		).
		First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return importjob.Progress{}, fmt.Errorf("find import job progress %q: %w", id, importjob.ErrNotFound)
	}
	if err != nil {
		return importjob.Progress{}, fmt.Errorf("find import job progress %q: %w", id, err)
	}

	return importjob.Progress{
		ID:            model.ID,
		Entity:        importjob.Entity(model.Entity),
		Status:        importjob.Status(model.Status),
		DryRun:        model.DryRun,
		TotalRows:     model.TotalRows,
		ProcessedRows: model.ProcessedRows,
		ValidRows:     model.ValidRows,
		ImportedRows:  model.ImportedRows,
		UpdatedRows:   model.UpdatedRows,
		FailedRows:    model.FailedRows,
		UpdatedAt:     model.UpdatedAt,
	}, nil
}

// List returns a page of import jobs and the total number of matching records.
func (r *Repository) List(ctx context.Context, filter importjob.Filter) ([]importjob.Job, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&gormmodel.ImportJob{})
		if filter.Entity != "" {
			query = query.Where("entity = ?", filter.Entity)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}

		return query
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count import jobs: %w", err)
	}

	var models []gormmodel.ImportJob
	offset := (filter.Page - 1) * filter.PageSize
	if err := base().
		Order("created_at DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list import jobs: %w", err)
	}

	jobs := make([]importjob.Job, 0, len(models))
	for _, model := range models {
		jobs = append(jobs, toDomain(model))
	}

	return jobs, total, nil
}

// Pending returns jobs waiting to be processed.
func (r *Repository) Pending(ctx context.Context) ([]importjob.Job, error) {
	var models []gormmodel.ImportJob

	if err := r.db.WithContext(ctx).
		Where("status = ?", importjob.StatusPending).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list pending import jobs: %w", err)
	}

	jobs := make([]importjob.Job, 0, len(models))
	for _, model := range models {
		jobs = append(jobs, toDomain(model))
	}

	return jobs, nil
}

// FailRunning marks jobs left running by a previous process as failed.
func (r *Repository) FailRunning(ctx context.Context, message string) error {
	now := time.Now().UTC()

	if err := r.db.WithContext(ctx).
		Model(&gormmodel.ImportJob{}).
		Where("status = ?", importjob.StatusRunning).
		Updates(map[string]any{
			"status":        importjob.StatusFailed,
			"error_message": message,
			"finished_at":   now,
			"updated_at":    now,
		}).Error; err != nil {
		return fmt.Errorf("fail running import jobs: %w", err)
	}

	return nil
}

func toModel(job importjob.Job) (gormmodel.ImportJob, error) {
	errorsJSON, err := json.Marshal(job.Errors)
	if err != nil {
		return gormmodel.ImportJob{}, fmt.Errorf("marshal import job errors: %w", err)
	}

	return gormmodel.ImportJob{
		ID:            job.ID,
		Entity:        string(job.Entity),
		Status:        string(job.Status),
		Mode:          job.Mode,
		DryRun:        job.DryRun,
		FileName:      job.FileName,
		FilePath:      job.FilePath,
		TotalRows:     job.TotalRows,
		ProcessedRows: job.ProcessedRows,
		ValidRows:     job.ValidRows,
		ImportedRows:  job.ImportedRows,
		UpdatedRows:   job.UpdatedRows,
		FailedRows:    job.FailedRows,
		ErrorsJSON:    string(errorsJSON),
		ErrorMessage:  job.ErrorMessage,
		CreatedAt:     job.CreatedAt,
		StartedAt:     job.StartedAt,
		FinishedAt:    job.FinishedAt,
		UpdatedAt:     job.UpdatedAt,
	}, nil
}

func toDomain(model gormmodel.ImportJob) importjob.Job {
	errorsList := make([]importjob.Error, 0)
	_ = json.Unmarshal([]byte(model.ErrorsJSON), &errorsList)
	if errorsList == nil {
		errorsList = make([]importjob.Error, 0)
	}

	return importjob.Job{
		ID:            model.ID,
		Entity:        importjob.Entity(model.Entity),
		Status:        importjob.Status(model.Status),
		Mode:          model.Mode,
		DryRun:        model.DryRun,
		FileName:      model.FileName,
		FilePath:      model.FilePath,
		TotalRows:     model.TotalRows,
		ProcessedRows: model.ProcessedRows,
		ValidRows:     model.ValidRows,
		ImportedRows:  model.ImportedRows,
		UpdatedRows:   model.UpdatedRows,
		FailedRows:    model.FailedRows,
		Errors:        errorsList,
		ErrorMessage:  model.ErrorMessage,
		CreatedAt:     model.CreatedAt,
		StartedAt:     model.StartedAt,
		FinishedAt:    model.FinishedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

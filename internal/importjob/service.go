package importjob

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service creates and reads import jobs.
type Service struct {
	repo    Repository
	storage *Storage
	queue   chan<- string
	logger  *slog.Logger
}

// NewService creates an import job service.
func NewService(repo Repository, storage *Storage, queue chan<- string, logger *slog.Logger) *Service {
	return &Service{repo: repo, storage: storage, queue: queue, logger: logger}
}

// Create stores the spreadsheet, creates a pending job and enqueues it.
func (s *Service) Create(ctx context.Context, entity Entity, mode string, dryRun bool, fileName string, file io.Reader) (Job, error) {
	if entity != EntityStations && entity != EntityRadioLinks {
		return Job{}, fmt.Errorf("%w: entity must be stations or radio_links", ErrValidation)
	}

	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "insert"
	}
	if mode != "insert" && mode != "upsert" {
		return Job{}, fmt.Errorf("%w: mode must be insert or upsert", ErrValidation)
	}

	id := uuid.NewString()
	path, err := s.storage.Save(id, file)
	if err != nil {
		return Job{}, err
	}

	job := Job{
		ID:       id,
		Entity:   entity,
		Status:   StatusPending,
		Mode:     mode,
		DryRun:   dryRun,
		FileName: fileName,
		FilePath: path,
		Errors:   make([]Error, 0),
	}

	if err := s.repo.Create(ctx, &job); err != nil {
		s.storage.Remove(path)
		return Job{}, fmt.Errorf("create import job: %w", err)
	}

	select {
	case s.queue <- job.ID:
	case <-ctx.Done():
		now := time.Now().UTC()
		job.Status = StatusFailed
		job.ErrorMessage = ctx.Err().Error()
		job.FinishedAt = &now
		_ = s.repo.Update(context.Background(), &job)
		return Job{}, ctx.Err()
	}

	return job, nil
}

// Get returns an import job by id.
func (s *Service) Get(ctx context.Context, id string) (Job, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Job{}, fmt.Errorf("%w: job id is required", ErrValidation)
	}

	return s.repo.FindByID(ctx, id)
}

// List returns a page of import jobs.
func (s *Service) List(ctx context.Context, filter Filter) (Page, error) {
	filter = filter.Normalize()

	items, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return Page{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	}

	return Page{
		Items:      items,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// Package importprocessor executes asynchronous Excel import jobs.
package importprocessor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/spreadsheet"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/importjob"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

// StationStore is the station persistence required by the import processor.
type StationStore interface {
	InsertBatch(ctx context.Context, rows []station.StationData) error
	UpsertBatch(ctx context.Context, rows []station.StationData) (int, int, error)
	Create(ctx context.Context, data station.StationData) (station.Station, error)
	FindByID(ctx context.Context, siteID string) (station.Station, error)
	Update(ctx context.Context, data station.StationData) (station.Station, error)
}

// RadioLinkStore is the radio link persistence required by the import processor.
type RadioLinkStore interface {
	InsertBatch(ctx context.Context, rows []radiolink.RadioLinkData) error
	UpsertBatch(ctx context.Context, rows []radiolink.RadioLinkData) (int, int, error)
	Create(ctx context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error)
	FindByID(ctx context.Context, linkID string) (radiolink.RadioLink, error)
	Update(ctx context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error)
}

// Processor executes import jobs in batches.
type Processor struct {
	jobs       importjob.Repository
	stations   StationStore
	radioLinks RadioLinkStore
	batchSize  int
	errorLimit int
	logger     *slog.Logger
}

// New creates an import processor.
func New(
	jobs importjob.Repository,
	stations StationStore,
	radioLinks RadioLinkStore,
	batchSize int,
	errorLimit int,
	logger *slog.Logger,
) *Processor {
	if batchSize < 1 {
		batchSize = 1000
	}
	if errorLimit < 1 {
		errorLimit = 1000
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Processor{
		jobs:       jobs,
		stations:   stations,
		radioLinks: radioLinks,
		batchSize:  batchSize,
		errorLimit: errorLimit,
		logger:     logger,
	}
}

// Process executes a job and persists its final status.
func (p *Processor) Process(ctx context.Context, job importjob.Job) error {
	now := time.Now().UTC()
	job.Status = importjob.StatusRunning
	job.StartedAt = &now
	job.FinishedAt = nil
	job.ErrorMessage = ""
	job.TotalRows = 0
	job.ProcessedRows = 0
	job.ValidRows = 0
	job.ImportedRows = 0
	job.UpdatedRows = 0
	job.FailedRows = 0
	job.Errors = make([]importjob.Error, 0)

	if err := p.jobs.Update(ctx, &job); err != nil {
		return fmt.Errorf("mark import job running: %w", err)
	}

	var processErr error
	switch job.Entity {
	case importjob.EntityStations:
		processErr = p.processStations(ctx, &job)
	case importjob.EntityRadioLinks:
		processErr = p.processRadioLinks(ctx, &job)
	default:
		processErr = fmt.Errorf("unsupported import entity %q", job.Entity)
	}

	finished := time.Now().UTC()
	job.FinishedAt = &finished
	if processErr != nil {
		job.Status = importjob.StatusFailed
		job.ErrorMessage = processErr.Error()
	} else {
		job.Status = importjob.StatusCompleted
	}

	updateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.jobs.Update(updateCtx, &job); err != nil {
		p.logger.Error("persist import job final status", "job_id", job.ID, "error", err)
	}

	return processErr
}

func (p *Processor) processStations(ctx context.Context, job *importjob.Job) error {
	batch := make([]station.ImportRow, 0, p.batchSize)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		validRows := make([]station.ImportRow, 0, len(batch))
		dataRows := make([]station.StationData, 0, len(batch))
		for _, row := range batch {
			data := row.Data
			data.Normalize()
			if err := data.Validate(); err != nil {
				job.AddError(p.errorLimit, row.Row, data.SiteID, err.Error())
				continue
			}

			job.ValidRows++
			if !job.DryRun {
				validRows = append(validRows, station.ImportRow{Row: row.Row, Data: data})
				dataRows = append(dataRows, data)
			}
		}

		if !job.DryRun && len(dataRows) > 0 {
			switch job.Mode {
			case "upsert":
				inserted, updated, err := p.stations.UpsertBatch(ctx, dataRows)
				if err != nil {
					p.upsertStationsIndividually(ctx, job, validRows)
				} else {
					job.ImportedRows += int64(inserted)
					job.UpdatedRows += int64(updated)
				}
			default:
				if err := p.stations.InsertBatch(ctx, dataRows); err != nil {
					p.insertStationsIndividually(ctx, job, validRows)
				} else {
					job.ImportedRows += int64(len(dataRows))
				}
			}
		}

		job.ProcessedRows += int64(len(batch))
		batch = batch[:0]

		return p.jobs.Update(ctx, job)
	}

	total, err := spreadsheet.StreamStations(job.FilePath, func(row station.ImportRow) error {
		batch = append(batch, row)
		if len(batch) >= p.batchSize {
			return flush()
		}

		return nil
	}, func(issue spreadsheet.Issue) error {
		job.ProcessedRows++
		job.AddError(p.errorLimit, issue.Row, "", issue.Message)
		return nil
	})
	if err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}

	job.TotalRows = int64(total)

	return nil
}

func (p *Processor) insertStationsIndividually(ctx context.Context, job *importjob.Job, rows []station.ImportRow) {
	for _, row := range rows {
		if _, err := p.stations.Create(ctx, row.Data); err != nil {
			job.AddError(p.errorLimit, row.Row, row.Data.SiteID, err.Error())
			continue
		}
		job.ImportedRows++
	}
}

func (p *Processor) upsertStationsIndividually(ctx context.Context, job *importjob.Job, rows []station.ImportRow) {
	for _, row := range rows {
		data := row.Data
		_, err := p.stations.FindByID(ctx, data.SiteID)
		switch {
		case err == nil:
			if _, err := p.stations.Update(ctx, data); err != nil {
				job.AddError(p.errorLimit, row.Row, data.SiteID, err.Error())
				continue
			}
			job.UpdatedRows++
		case errors.Is(err, station.ErrNotFound):
			if _, err := p.stations.Create(ctx, data); err != nil {
				job.AddError(p.errorLimit, row.Row, data.SiteID, err.Error())
				continue
			}
			job.ImportedRows++
		default:
			job.AddError(p.errorLimit, row.Row, data.SiteID, err.Error())
		}
	}
}

func (p *Processor) processRadioLinks(ctx context.Context, job *importjob.Job) error {
	batch := make([]radiolink.ImportRow, 0, p.batchSize)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		validRows := make([]radiolink.ImportRow, 0, len(batch))
		dataRows := make([]radiolink.RadioLinkData, 0, len(batch))
		for _, row := range batch {
			data := row.Data
			data.Normalize()
			data.EnsureGenerated()
			if err := data.Validate(); err != nil {
				job.AddError(p.errorLimit, row.Row, data.LinkID, err.Error())
				continue
			}

			job.ValidRows++
			if !job.DryRun {
				validRows = append(validRows, radiolink.ImportRow{Row: row.Row, Data: data})
				dataRows = append(dataRows, data)
			}
		}

		if !job.DryRun && len(dataRows) > 0 {
			switch job.Mode {
			case "upsert":
				inserted, updated, err := p.radioLinks.UpsertBatch(ctx, dataRows)
				if err != nil {
					p.upsertRadioLinksIndividually(ctx, job, validRows)
				} else {
					job.ImportedRows += int64(inserted)
					job.UpdatedRows += int64(updated)
				}
			default:
				if err := p.radioLinks.InsertBatch(ctx, dataRows); err != nil {
					p.insertRadioLinksIndividually(ctx, job, validRows)
				} else {
					job.ImportedRows += int64(len(dataRows))
				}
			}
		}

		job.ProcessedRows += int64(len(batch))
		batch = batch[:0]

		return p.jobs.Update(ctx, job)
	}

	total, err := spreadsheet.StreamRadioLinks(job.FilePath, func(row radiolink.ImportRow) error {
		batch = append(batch, row)
		if len(batch) >= p.batchSize {
			return flush()
		}

		return nil
	}, func(issue spreadsheet.Issue) error {
		job.ProcessedRows++
		job.AddError(p.errorLimit, issue.Row, "", issue.Message)
		return nil
	})
	if err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}

	job.TotalRows = int64(total)

	return nil
}

func (p *Processor) insertRadioLinksIndividually(ctx context.Context, job *importjob.Job, rows []radiolink.ImportRow) {
	for _, row := range rows {
		if _, err := p.radioLinks.Create(ctx, row.Data); err != nil {
			job.AddError(p.errorLimit, row.Row, row.Data.LinkID, err.Error())
			continue
		}
		job.ImportedRows++
	}
}

func (p *Processor) upsertRadioLinksIndividually(ctx context.Context, job *importjob.Job, rows []radiolink.ImportRow) {
	for _, row := range rows {
		data := row.Data
		_, err := p.radioLinks.FindByID(ctx, data.LinkID)
		switch {
		case err == nil:
			if _, err := p.radioLinks.Update(ctx, data); err != nil {
				job.AddError(p.errorLimit, row.Row, data.LinkID, err.Error())
				continue
			}
			job.UpdatedRows++
		case errors.Is(err, radiolink.ErrNotFound):
			if _, err := p.radioLinks.Create(ctx, data); err != nil {
				job.AddError(p.errorLimit, row.Row, data.LinkID, err.Error())
				continue
			}
			job.ImportedRows++
		default:
			job.AddError(p.errorLimit, row.Row, data.LinkID, err.Error())
		}
	}
}

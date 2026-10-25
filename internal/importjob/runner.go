package importjob

import (
	"context"
	"log/slog"
)

// Processor executes an import job.
type Processor interface {
	Process(ctx context.Context, job Job) error
}

// Runner consumes the import job queue with a fixed number of workers.
type Runner struct {
	repo      Repository
	processor Processor
	queue     chan string
	workers   int
	logger    *slog.Logger
}

// NewRunner creates an import job runner.
func NewRunner(repo Repository, processor Processor, queue chan string, workers int, logger *slog.Logger) *Runner {
	if workers < 1 {
		workers = 1
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Runner{repo: repo, processor: processor, queue: queue, workers: workers, logger: logger}
}

// Start recovers pending jobs and starts the workers.
func (r *Runner) Start(ctx context.Context) {
	if err := r.repo.FailRunning(ctx, "import interrupted by application restart"); err != nil {
		r.logger.Error("fail interrupted import jobs", "error", err)
	}

	for index := 0; index < r.workers; index++ {
		go r.worker(ctx)
	}

	pending, err := r.repo.Pending(ctx)
	if err != nil {
		r.logger.Error("list pending import jobs", "error", err)
		return
	}

	for _, job := range pending {
		select {
		case r.queue <- job.ID:
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-r.queue:
			job, err := r.repo.FindByID(ctx, id)
			if err != nil {
				r.logger.Error("load import job", "job_id", id, "error", err)
				continue
			}
			if job.Status != StatusPending {
				continue
			}
			if err := r.processor.Process(ctx, job); err != nil {
				r.logger.Error("process import job", "job_id", job.ID, "error", err)
			}
		}
	}
}

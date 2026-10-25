package importjob

import "context"

// Repository is the persistence port for import jobs.
type Repository interface {
	Create(ctx context.Context, job *Job) error
	Update(ctx context.Context, job *Job) error
	FindByID(ctx context.Context, id string) (Job, error)
	FindProgress(ctx context.Context, id string) (Progress, error)
	List(ctx context.Context, filter Filter) ([]Job, int64, error)
	Pending(ctx context.Context) ([]Job, error)
	FailRunning(ctx context.Context, message string) error
}

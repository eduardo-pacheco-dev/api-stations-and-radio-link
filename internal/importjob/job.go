// Package importjob contains the asynchronous import job domain and services.
package importjob

import (
	"errors"
	"strings"
	"time"
)

// Entity identifies what an import job writes.
type Entity string

const (
	EntityStations   Entity = "stations"
	EntityRadioLinks Entity = "radio_links"
)

// Status is the lifecycle state of an import job.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

// Error describes a row that failed during an import job.
type Error struct {
	Row     int    `json:"row" example:"2"`
	Key     string `json:"key,omitempty" example:"TSHG01"`
	Message string `json:"message" example:"invalid latitude \"abc\""`
}

// Job is a persisted bulk import job.
type Job struct {
	ID            string     `json:"id" example:"6f0e6e6e-..." `
	Entity        Entity     `json:"entity" example:"stations"`
	Status        Status     `json:"status" example:"running"`
	Mode          string     `json:"mode" example:"upsert"`
	DryRun        bool       `json:"dry_run" example:"false"`
	FileName      string     `json:"file_name" example:"stations.xlsx"`
	FilePath      string     `json:"-"`
	TotalRows     int64      `json:"total_rows" example:"700000"`
	ProcessedRows int64      `json:"processed_rows" example:"150000"`
	ValidRows     int64      `json:"valid_rows" example:"149500"`
	ImportedRows  int64      `json:"imported_rows" example:"100000"`
	UpdatedRows   int64      `json:"updated_rows" example:"49500"`
	FailedRows    int64      `json:"failed_rows" example:"500"`
	Errors        []Error    `json:"errors"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AddError increments the failure counter and stores the error up to limit.
func (j *Job) AddError(limit int, row int, key, message string) {
	j.FailedRows++
	if len(j.Errors) < limit {
		j.Errors = append(j.Errors, Error{Row: row, Key: key, Message: message})
	}
}

// Progress is a lightweight import job status used for polling.
type Progress struct {
	ID            string    `json:"id" example:"6f0e6e6e-..."`
	Entity        Entity    `json:"entity" example:"stations"`
	Status        Status    `json:"status" example:"running"`
	DryRun        bool      `json:"dry_run" example:"false"`
	TotalRows     int64     `json:"total_rows" example:"700000"`
	ProcessedRows int64     `json:"processed_rows" example:"150000"`
	ValidRows     int64     `json:"valid_rows" example:"149500"`
	ImportedRows  int64     `json:"imported_rows" example:"100000"`
	UpdatedRows   int64     `json:"updated_rows" example:"49500"`
	FailedRows    int64     `json:"failed_rows" example:"500"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Filter contains job list filters.
type Filter struct {
	Page     int
	PageSize int
	Entity   string
	Status   string
}

// Normalize applies pagination defaults and limits.
func (f Filter) Normalize() Filter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = DefaultPageSize
	}
	if f.PageSize > MaxPageSize {
		f.PageSize = MaxPageSize
	}
	f.Entity = strings.TrimSpace(f.Entity)
	f.Status = strings.TrimSpace(f.Status)

	return f
}

// Page is a paginated list of import jobs.
type Page struct {
	Items      []Job `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

const (
	// DefaultPageSize is used when the client does not request a page size.
	DefaultPageSize = 20
	// MaxPageSize limits the number of jobs returned by a list request.
	MaxPageSize = 100
)

// ErrNotFound indicates that an import job does not exist.
var ErrNotFound = errors.New("import job not found")

// ErrValidation indicates invalid import job input.
var ErrValidation = errors.New("validation failed")

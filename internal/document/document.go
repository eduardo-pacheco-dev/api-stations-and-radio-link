// Package document contains the document domain and use cases.
package document

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// OwnerType identifies the entity a document belongs to.
type OwnerType string

const (
	OwnerTypeStation   OwnerType = "station"
	OwnerTypeRadioLink OwnerType = "radio_link"
)

// Valid reports whether the owner type is supported.
func (t OwnerType) Valid() bool {
	return t == OwnerTypeStation || t == OwnerTypeRadioLink
}

// Document is a file attached to a station or radio link.
type Document struct {
	ID          string    `json:"id" example:"6f0e6e6e-..."`
	OwnerType   OwnerType `json:"owner_type" enums:"station,radio_link" example:"station"`
	OwnerID     string    `json:"owner_id" example:"TSHG01"`
	Title       string    `json:"title" example:"Licença de operação"`
	Category    string    `json:"category" example:"licenca"`
	Description string    `json:"description" example:"Licença válida até 2027"`
	FileName    string    `json:"file_name" example:"licenca.pdf"`
	ContentType string    `json:"content_type" example:"application/pdf"`
	SizeBytes   int64     `json:"size_bytes" example:"102400"`
	FilePath    string    `json:"-"`
	DownloadURL string    `json:"download_url,omitempty" example:"/api/v1/documents/6f0e6e6e-.../download"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateInput contains the metadata required to create a document.
type CreateInput struct {
	OwnerType   OwnerType
	OwnerID     string
	Title       string
	Category    string
	Description string
	FileName    string
	ContentType string
}

// UpdateInput contains the editable document metadata.
type UpdateInput struct {
	Title       string
	Category    string
	Description string
}

// ListFilter contains document list filters.
type ListFilter struct {
	Page      int
	PageSize  int
	OwnerType string
	OwnerID   string
	Search    string
}

// Normalize applies pagination defaults and limits.
func (f ListFilter) Normalize() ListFilter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = DefaultPageSize
	}
	if f.PageSize > MaxPageSize {
		f.PageSize = MaxPageSize
	}
	f.OwnerType = strings.TrimSpace(f.OwnerType)
	f.OwnerID = strings.TrimSpace(f.OwnerID)
	f.Search = strings.TrimSpace(f.Search)

	return f
}

// Page is a paginated document list.
type Page struct {
	Items      []Document `json:"items"`
	Page       int        `json:"page" example:"1"`
	PageSize   int        `json:"page_size" example:"20"`
	Total      int64      `json:"total" example:"1"`
	TotalPages int        `json:"total_pages" example:"1"`
}

const (
	// DefaultPageSize is used when the client does not request a page size.
	DefaultPageSize = 20
	// MaxPageSize limits the number of documents returned.
	MaxPageSize = 100
)

// ErrNotFound indicates that a document does not exist.
var ErrNotFound = errors.New("document not found")

// ErrValidation indicates invalid document input.
var ErrValidation = errors.New("validation failed")

// ErrOwnerNotFound indicates that the referenced station or radio link does not exist.
var ErrOwnerNotFound = errors.New("document owner not found")

// Validate checks document creation input.
func (in CreateInput) Validate() error {
	var problems []string

	if !in.OwnerType.Valid() {
		problems = append(problems, "owner_type must be station or radio_link")
	}
	if in.OwnerID == "" {
		problems = append(problems, "owner_id is required")
	}
	if in.Title == "" {
		problems = append(problems, "title is required")
	}
	if in.FileName == "" {
		problems = append(problems, "file is required")
	}

	limits := []struct {
		field string
		value string
		max   int
	}{
		{"owner_id", in.OwnerID, 50},
		{"title", in.Title, 200},
		{"category", in.Category, 100},
		{"description", in.Description, 2000},
		{"file_name", in.FileName, 255},
	}
	for _, limit := range limits {
		if len([]rune(limit.value)) > limit.max {
			problems = append(problems, fmt.Sprintf("%s must have at most %d characters", limit.field, limit.max))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(problems, "; "))
	}

	return nil
}

// Validate checks document update input.
func (in UpdateInput) Validate() error {
	var problems []string

	limits := []struct {
		field string
		value string
		max   int
	}{
		{"title", in.Title, 200},
		{"category", in.Category, 100},
		{"description", in.Description, 2000},
	}
	for _, limit := range limits {
		if len([]rune(limit.value)) > limit.max {
			problems = append(problems, fmt.Sprintf("%s must have at most %d characters", limit.field, limit.max))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(problems, "; "))
	}

	return nil
}

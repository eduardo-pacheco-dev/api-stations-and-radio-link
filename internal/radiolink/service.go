package radiolink

import (
	"context"
	"fmt"
	"strings"
)

// Service implements the radio link use cases.
type Service struct {
	repo Repository
}

// NewService creates a radio link service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create validates and persists a radio link.
func (s *Service) Create(ctx context.Context, data RadioLinkData) (RadioLink, error) {
	data.Normalize()
	if err := data.Validate(); err != nil {
		return RadioLink{}, err
	}

	return s.repo.Create(ctx, data)
}

// Get returns a radio link by link_id.
func (s *Service) Get(ctx context.Context, linkID string) (RadioLink, error) {
	linkID = strings.TrimSpace(linkID)
	if linkID == "" {
		return RadioLink{}, fmt.Errorf("%w: link_id is required", ErrValidation)
	}

	return s.repo.FindByID(ctx, linkID)
}

// List returns a paginated list of radio links.
func (s *Service) List(ctx context.Context, filter ListFilter) (Page, error) {
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

// Update validates and updates a radio link by link_id.
func (s *Service) Update(ctx context.Context, linkID string, data RadioLinkData) (RadioLink, error) {
	data.LinkID = strings.TrimSpace(linkID)
	data.Normalize()
	if err := data.Validate(); err != nil {
		return RadioLink{}, err
	}

	return s.repo.Update(ctx, data)
}

// Delete removes a radio link by link_id.
func (s *Service) Delete(ctx context.Context, linkID string) error {
	linkID = strings.TrimSpace(linkID)
	if linkID == "" {
		return fmt.Errorf("%w: link_id is required", ErrValidation)
	}

	return s.repo.Delete(ctx, linkID)
}

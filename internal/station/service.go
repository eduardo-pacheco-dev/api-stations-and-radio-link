package station

import (
	"context"
	"fmt"
	"strings"
)

// Service implements the station use cases.
type Service struct {
	repo Repository
}

// NewService creates a station service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create validates and persists a station.
func (s *Service) Create(ctx context.Context, data StationData) (Station, error) {
	data.Normalize()
	if err := data.Validate(); err != nil {
		return Station{}, err
	}

	return s.repo.Create(ctx, data)
}

// Get returns a station by site_id.
func (s *Service) Get(ctx context.Context, siteID string) (Station, error) {
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		return Station{}, fmt.Errorf("%w: site_id is required", ErrValidation)
	}

	return s.repo.FindByID(ctx, siteID)
}

// List returns a paginated list of stations.
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

// Update validates and updates a station by site_id.
func (s *Service) Update(ctx context.Context, siteID string, data StationData) (Station, error) {
	data.SiteID = strings.TrimSpace(siteID)
	data.Normalize()
	if err := data.Validate(); err != nil {
		return Station{}, err
	}

	return s.repo.Update(ctx, data)
}

// Delete removes a station by site_id.
func (s *Service) Delete(ctx context.Context, siteID string) error {
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		return fmt.Errorf("%w: site_id is required", ErrValidation)
	}

	return s.repo.Delete(ctx, siteID)
}

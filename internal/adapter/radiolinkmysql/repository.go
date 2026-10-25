// Package radiolinkmysql implements the radio link repository with GORM.
package radiolinkmysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormerr"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormmodel"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
)

// Repository persists radio links with GORM.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a GORM radio link repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a radio link and returns the persisted record.
func (r *Repository) Create(ctx context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	model := toModel(data)

	if err := r.db.WithContext(ctx).Omit(clause.Associations).Create(&model).Error; err != nil {
		return radiolink.RadioLink{}, mapWriteError(err, data.LinkID)
	}

	return r.FindByID(ctx, data.LinkID)
}

// FindByID returns a radio link by link_id.
func (r *Repository) FindByID(ctx context.Context, linkID string) (radiolink.RadioLink, error) {
	var model gormmodel.RadioLink

	err := r.db.WithContext(ctx).First(&model, "link_id = ?", linkID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return radiolink.RadioLink{}, fmt.Errorf("find radio link %q: %w", linkID, radiolink.ErrNotFound)
	}
	if err != nil {
		return radiolink.RadioLink{}, fmt.Errorf("find radio link %q: %w", linkID, err)
	}

	return toDomain(model), nil
}

// List returns a page of radio links and the total number of matching records.
func (r *Repository) List(ctx context.Context, filter radiolink.ListFilter) ([]radiolink.RadioLink, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&gormmodel.RadioLink{})
		if filter.Search != "" {
			term := "%" + filter.Search + "%"
			query = query.Where(
				"(link_id LIKE ? OR name LIKE ? OR station_a_site_id LIKE ? OR station_b_site_id LIKE ? OR technology LIKE ?)",
				term, term, term, term, term,
			)
		}
		if filter.StationID != "" {
			query = query.Where("(station_a_site_id = ? OR station_b_site_id = ?)", filter.StationID, filter.StationID)
		}

		return query
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count radio links: %w", err)
	}

	var models []gormmodel.RadioLink
	offset := (filter.Page - 1) * filter.PageSize
	if err := base().
		Order("link_id ASC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list radio links: %w", err)
	}

	records := make([]radiolink.RadioLink, 0, len(models))
	for _, model := range models {
		records = append(records, toDomain(model))
	}

	return records, total, nil
}

// Update replaces the mutable fields of an existing radio link.
func (r *Repository) Update(ctx context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	updates := map[string]any{
		"name":              data.Name,
		"station_a_site_id": data.StationASiteID,
		"station_b_site_id": data.StationBSiteID,
		"technology":        data.Technology,
		"frequency_mhz":     nullableFloat(data.FrequencyMHz),
		"capacity_mbps":     nullableFloat(data.CapacityMbps),
		"distance_km":       nullableFloat(data.DistanceKM),
		"status":            data.Status,
		"observation":       data.Observation,
		"updated_at":        time.Now().UTC(),
	}

	result := r.db.WithContext(ctx).
		Model(&gormmodel.RadioLink{}).
		Where("link_id = ?", data.LinkID).
		Updates(updates)
	if result.Error != nil {
		return radiolink.RadioLink{}, mapWriteError(result.Error, data.LinkID)
	}
	if result.RowsAffected == 0 {
		return r.FindByID(ctx, data.LinkID)
	}

	return r.FindByID(ctx, data.LinkID)
}

// Delete removes a radio link by link_id.
func (r *Repository) Delete(ctx context.Context, linkID string) error {
	result := r.db.WithContext(ctx).
		Where("link_id = ?", linkID).
		Delete(&gormmodel.RadioLink{})
	if result.Error != nil {
		return fmt.Errorf("delete radio link %q: %w", linkID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete radio link %q: %w", linkID, radiolink.ErrNotFound)
	}

	return nil
}

func mapWriteError(err error, linkID string) error {
	switch {
	case gormerr.IsDuplicate(err):
		return fmt.Errorf("save radio link %q: %w", linkID, radiolink.ErrAlreadyExists)
	case gormerr.IsForeignKey(err):
		return fmt.Errorf("save radio link %q: %w", linkID, radiolink.ErrStationNotFound)
	default:
		return fmt.Errorf("save radio link %q: %w", linkID, err)
	}
}

func toModel(data radiolink.RadioLinkData) gormmodel.RadioLink {
	return gormmodel.RadioLink{
		LinkID:         data.LinkID,
		Name:           data.Name,
		StationASiteID: data.StationASiteID,
		StationBSiteID: data.StationBSiteID,
		Technology:     data.Technology,
		FrequencyMHz:   data.FrequencyMHz,
		CapacityMbps:   data.CapacityMbps,
		DistanceKM:     data.DistanceKM,
		Status:         data.Status,
		Observation:    data.Observation,
	}
}

func toDomain(model gormmodel.RadioLink) radiolink.RadioLink {
	return radiolink.RadioLink{
		RadioLinkData: radiolink.RadioLinkData{
			LinkID:         model.LinkID,
			Name:           model.Name,
			StationASiteID: model.StationASiteID,
			StationBSiteID: model.StationBSiteID,
			Technology:     model.Technology,
			FrequencyMHz:   model.FrequencyMHz,
			CapacityMbps:   model.CapacityMbps,
			DistanceKM:     model.DistanceKM,
			Status:         model.Status,
			Observation:    model.Observation,
		},
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}

	return *value
}

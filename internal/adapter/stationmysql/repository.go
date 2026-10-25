// Package stationmysql implements the station repository with GORM.
package stationmysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormerr"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/gormmodel"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

// Repository persists stations with GORM.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a GORM station repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a station and returns the persisted record.
func (r *Repository) Create(ctx context.Context, data station.StationData) (station.Station, error) {
	model := toModel(data)

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		if gormerr.IsDuplicate(err) {
			return station.Station{}, fmt.Errorf("create station %q: %w", data.SiteID, station.ErrAlreadyExists)
		}

		return station.Station{}, fmt.Errorf("insert station %q: %w", data.SiteID, err)
	}

	return r.FindByID(ctx, data.SiteID)
}

// FindByID returns a station by site_id.
func (r *Repository) FindByID(ctx context.Context, siteID string) (station.Station, error) {
	var model gormmodel.Station

	err := r.db.WithContext(ctx).First(&model, "site_id = ?", siteID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return station.Station{}, fmt.Errorf("find station %q: %w", siteID, station.ErrNotFound)
	}
	if err != nil {
		return station.Station{}, fmt.Errorf("find station %q: %w", siteID, err)
	}

	return toDomain(model), nil
}

// List returns a page of stations and the total number of matching records.
func (r *Repository) List(ctx context.Context, filter station.ListFilter) ([]station.Station, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&gormmodel.Station{})
		if filter.Search != "" {
			term := "%" + filter.Search + "%"
			query = query.Where(
				"(site_id LIKE ? OR end_id LIKE ? OR station_id LIKE ? OR city LIKE ? OR address LIKE ?)",
				term, term, term, term, term,
			)
		}

		return query
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count stations: %w", err)
	}

	var models []gormmodel.Station
	offset := (filter.Page - 1) * filter.PageSize
	if err := base().
		Order("site_id ASC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list stations: %w", err)
	}

	records := make([]station.Station, 0, len(models))
	for _, model := range models {
		records = append(records, toDomain(model))
	}

	return records, total, nil
}

// Update replaces the mutable fields of an existing station.
func (r *Repository) Update(ctx context.Context, data station.StationData) (station.Station, error) {
	updates := map[string]any{
		"technology":          data.Technology,
		"end_id":              data.EndID,
		"contract_area_type":  data.ContractAreaType,
		"area_holder":         data.AreaHolder,
		"infrastructure_type": data.InfrastructureType,
		"ev_type":             data.EVType,
		"address":             data.Address,
		"neighborhood":        data.Neighborhood,
		"city":                data.City,
		"state":               data.State,
		"zip_code":            data.ZipCode,
		"regional":            data.Regional,
		"latitude":            nullableFloat(data.Latitude),
		"longitude":           nullableFloat(data.Longitude),
		"status":              data.Status,
		"tower_type":          data.TowerType,
		"nominal_aev":         data.NominalAEV,
		"station_id":          data.StationIDRef,
		"ev_supplier":         data.EVSupplier,
		"land_area":           nullableFloat(data.LandArea),
		"structure_height":    nullableFloat(data.StructureHeight),
		"observation":         data.Observation,
		"updated_at":          time.Now().UTC(),
	}

	result := r.db.WithContext(ctx).
		Model(&gormmodel.Station{}).
		Where("site_id = ?", data.SiteID).
		Updates(updates)
	if result.Error != nil {
		return station.Station{}, fmt.Errorf("update station %q: %w", data.SiteID, result.Error)
	}
	if result.RowsAffected == 0 {
		return r.FindByID(ctx, data.SiteID)
	}

	return r.FindByID(ctx, data.SiteID)
}

// Delete removes a station by site_id.
func (r *Repository) Delete(ctx context.Context, siteID string) error {
	result := r.db.WithContext(ctx).
		Where("site_id = ?", siteID).
		Delete(&gormmodel.Station{})
	if result.Error != nil {
		if gormerr.IsForeignKey(result.Error) {
			return fmt.Errorf("delete station %q: %w", siteID, station.ErrInUse)
		}

		return fmt.Errorf("delete station %q: %w", siteID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete station %q: %w", siteID, station.ErrNotFound)
	}

	return nil
}

// InsertBatch inserts a batch of stations.
func (r *Repository) InsertBatch(ctx context.Context, rows []station.StationData) error {
	if len(rows) == 0 {
		return nil
	}

	models := make([]gormmodel.Station, 0, len(rows))
	for _, data := range rows {
		models = append(models, toModel(data))
	}

	if err := r.db.WithContext(ctx).CreateInBatches(&models, len(models)).Error; err != nil {
		return fmt.Errorf("insert station batch: %w", err)
	}

	return nil
}

// UpsertBatch inserts or updates a batch of stations and returns inserted/updated counts.
func (r *Repository) UpsertBatch(ctx context.Context, rows []station.StationData) (int, int, error) {
	if len(rows) == 0 {
		return 0, 0, nil
	}

	models := make([]gormmodel.Station, 0, len(rows))
	ids := make([]string, 0, len(rows))
	for _, data := range rows {
		models = append(models, toModel(data))
		ids = append(ids, data.SiteID)
	}

	var existing []string
	if err := r.db.WithContext(ctx).
		Model(&gormmodel.Station{}).
		Where("site_id IN ?", ids).
		Pluck("site_id", &existing).Error; err != nil {
		return 0, 0, fmt.Errorf("load existing stations: %w", err)
	}

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "site_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"technology", "end_id", "contract_area_type", "area_holder",
				"infrastructure_type", "ev_type", "address", "neighborhood",
				"city", "state", "zip_code", "regional", "latitude", "longitude",
				"status", "tower_type", "nominal_aev", "station_id", "ev_supplier",
				"land_area", "structure_height", "observation", "updated_at",
			}),
		}).
		CreateInBatches(&models, len(models)).Error
	if err != nil {
		return 0, 0, fmt.Errorf("upsert station batch: %w", err)
	}

	updated := len(existing)
	inserted := len(models) - updated

	return inserted, updated, nil
}

func toModel(data station.StationData) gormmodel.Station {
	return gormmodel.Station{
		SiteID:             data.SiteID,
		Technology:         data.Technology,
		EndID:              data.EndID,
		ContractAreaType:   data.ContractAreaType,
		AreaHolder:         data.AreaHolder,
		InfrastructureType: data.InfrastructureType,
		EVType:             data.EVType,
		Address:            data.Address,
		Neighborhood:       data.Neighborhood,
		City:               data.City,
		State:              data.State,
		ZipCode:            data.ZipCode,
		Regional:           data.Regional,
		Latitude:           data.Latitude,
		Longitude:          data.Longitude,
		Status:             data.Status,
		TowerType:          data.TowerType,
		NominalAEV:         data.NominalAEV,
		StationID:          data.StationIDRef,
		EVSupplier:         data.EVSupplier,
		LandArea:           data.LandArea,
		StructureHeight:    data.StructureHeight,
		Observation:        data.Observation,
	}
}

func toDomain(model gormmodel.Station) station.Station {
	return station.Station{
		StationData: station.StationData{
			SiteID:             model.SiteID,
			Technology:         model.Technology,
			EndID:              model.EndID,
			ContractAreaType:   model.ContractAreaType,
			AreaHolder:         model.AreaHolder,
			InfrastructureType: model.InfrastructureType,
			EVType:             model.EVType,
			Address:            model.Address,
			Neighborhood:       model.Neighborhood,
			City:               model.City,
			State:              model.State,
			ZipCode:            model.ZipCode,
			Regional:           model.Regional,
			Latitude:           model.Latitude,
			Longitude:          model.Longitude,
			Status:             model.Status,
			TowerType:          model.TowerType,
			NominalAEV:         model.NominalAEV,
			StationIDRef:       model.StationID,
			EVSupplier:         model.EVSupplier,
			LandArea:           model.LandArea,
			StructureHeight:    model.StructureHeight,
			Observation:        model.Observation,
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

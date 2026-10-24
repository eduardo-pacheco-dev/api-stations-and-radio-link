// Package stationmysql implements the station repository with MySQL.
package stationmysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/station"
)

const stationColumns = `
	site_id, technology, end_id, contract_area_type, area_holder,
	infrastructure_type, ev_type, address, neighborhood, city, state,
	zip_code, regional, latitude, longitude, status, tower_type, nominal_aev,
	station_id, ev_supplier, land_area, structure_height, observation,
	created_at, updated_at`

const insertStation = `
INSERT INTO stations (
	site_id, technology, end_id, contract_area_type, area_holder,
	infrastructure_type, ev_type, address, neighborhood, city, state,
	zip_code, regional, latitude, longitude, status, tower_type, nominal_aev,
	station_id, ev_supplier, land_area, structure_height, observation
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const updateStation = `
UPDATE stations SET
	technology = ?, end_id = ?, contract_area_type = ?, area_holder = ?,
	infrastructure_type = ?, ev_type = ?, address = ?, neighborhood = ?,
	city = ?, state = ?, zip_code = ?, regional = ?, latitude = ?,
	longitude = ?, status = ?, tower_type = ?, nominal_aev = ?,
	station_id = ?, ev_supplier = ?, land_area = ?, structure_height = ?,
	observation = ?
WHERE site_id = ?`

// Repository persists stations in MySQL.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a MySQL station repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a station and returns the persisted record.
func (r *Repository) Create(ctx context.Context, data station.StationData) (station.Station, error) {
	_, err := r.db.ExecContext(ctx, insertStation,
		data.SiteID,
		data.Technology,
		data.EndID,
		data.ContractAreaType,
		data.AreaHolder,
		data.InfrastructureType,
		data.EVType,
		data.Address,
		data.Neighborhood,
		data.City,
		data.State,
		data.ZipCode,
		data.Regional,
		nullableFloat(data.Latitude),
		nullableFloat(data.Longitude),
		data.Status,
		data.TowerType,
		data.NominalAEV,
		data.StationIDRef,
		data.EVSupplier,
		nullableFloat(data.LandArea),
		nullableFloat(data.StructureHeight),
		data.Observation,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return station.Station{}, fmt.Errorf("create station %q: %w", data.SiteID, station.ErrAlreadyExists)
		}

		return station.Station{}, fmt.Errorf("insert station %q: %w", data.SiteID, err)
	}

	return r.FindByID(ctx, data.SiteID)
}

// FindByID returns a station by site_id.
func (r *Repository) FindByID(ctx context.Context, siteID string) (station.Station, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+stationColumns+" FROM stations WHERE site_id = ?",
		siteID,
	)

	record, err := scanStation(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return station.Station{}, fmt.Errorf("find station %q: %w", siteID, station.ErrNotFound)
	}
	if err != nil {
		return station.Station{}, fmt.Errorf("find station %q: %w", siteID, err)
	}

	return record, nil
}

// List returns a page of stations and the total number of matching records.
func (r *Repository) List(ctx context.Context, filter station.ListFilter) ([]station.Station, int64, error) {
	whereClause, args := buildWhere(filter.Search)

	var total int64
	countQuery := "SELECT COUNT(*) FROM stations" + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stations: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	query := "SELECT " + stationColumns + " FROM stations" + whereClause +
		" ORDER BY site_id LIMIT ? OFFSET ?"

	queryArgs := append(append([]any{}, args...), filter.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stations: %w", err)
	}
	defer rows.Close()

	records := make([]station.Station, 0, filter.PageSize)
	for rows.Next() {
		record, err := scanStation(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("scan station: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stations: %w", err)
	}

	return records, total, nil
}

// Update replaces the mutable fields of an existing station.
func (r *Repository) Update(ctx context.Context, data station.StationData) (station.Station, error) {
	_, err := r.db.ExecContext(ctx, updateStation,
		data.Technology,
		data.EndID,
		data.ContractAreaType,
		data.AreaHolder,
		data.InfrastructureType,
		data.EVType,
		data.Address,
		data.Neighborhood,
		data.City,
		data.State,
		data.ZipCode,
		data.Regional,
		nullableFloat(data.Latitude),
		nullableFloat(data.Longitude),
		data.Status,
		data.TowerType,
		data.NominalAEV,
		data.StationIDRef,
		data.EVSupplier,
		nullableFloat(data.LandArea),
		nullableFloat(data.StructureHeight),
		data.Observation,
		data.SiteID,
	)
	if err != nil {
		return station.Station{}, fmt.Errorf("update station %q: %w", data.SiteID, err)
	}

	return r.FindByID(ctx, data.SiteID)
}

// Delete removes a station by site_id.
func (r *Repository) Delete(ctx context.Context, siteID string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM stations WHERE site_id = ?", siteID)
	if err != nil {
		return fmt.Errorf("delete station %q: %w", siteID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete station %q: %w", siteID, err)
	}
	if affected == 0 {
		return fmt.Errorf("delete station %q: %w", siteID, station.ErrNotFound)
	}

	return nil
}

func buildWhere(search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", nil
	}

	term := "%" + search + "%"

	return " WHERE (site_id LIKE ? OR end_id LIKE ? OR station_id LIKE ? OR city LIKE ? OR address LIKE ?)",
		[]any{term, term, term, term, term}
}

func scanStation(scan func(dest ...any) error) (station.Station, error) {
	var (
		record          station.Station
		latitude        sql.NullFloat64
		longitude       sql.NullFloat64
		landArea        sql.NullFloat64
		structureHeight sql.NullFloat64
	)

	err := scan(
		&record.SiteID,
		&record.Technology,
		&record.EndID,
		&record.ContractAreaType,
		&record.AreaHolder,
		&record.InfrastructureType,
		&record.EVType,
		&record.Address,
		&record.Neighborhood,
		&record.City,
		&record.State,
		&record.ZipCode,
		&record.Regional,
		&latitude,
		&longitude,
		&record.Status,
		&record.TowerType,
		&record.NominalAEV,
		&record.StationIDRef,
		&record.EVSupplier,
		&landArea,
		&structureHeight,
		&record.Observation,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return station.Station{}, err
	}

	record.Latitude = floatPointer(latitude)
	record.Longitude = floatPointer(longitude)
	record.LandArea = floatPointer(landArea)
	record.StructureHeight = floatPointer(structureHeight)

	return record, nil
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}

	return *value
}

func floatPointer(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}

	converted := value.Float64

	return &converted
}

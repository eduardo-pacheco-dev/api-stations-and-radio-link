// Package radiolinkmysql implements the radio link repository with MySQL.
package radiolinkmysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/radiolink"
)

const radioLinkColumns = `
	link_id, name, station_a_site_id, station_b_site_id, technology,
	frequency_mhz, capacity_mbps, distance_km, status, observation,
	created_at, updated_at`

const insertRadioLink = `
INSERT INTO radio_links (
	link_id, name, station_a_site_id, station_b_site_id, technology,
	frequency_mhz, capacity_mbps, distance_km, status, observation
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const updateRadioLink = `
UPDATE radio_links SET
	name = ?, station_a_site_id = ?, station_b_site_id = ?, technology = ?,
	frequency_mhz = ?, capacity_mbps = ?, distance_km = ?, status = ?,
	observation = ?
WHERE link_id = ?`

// Repository persists radio links in MySQL.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a MySQL radio link repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a radio link and returns the persisted record.
func (r *Repository) Create(ctx context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	_, err := r.db.ExecContext(ctx, insertRadioLink,
		data.LinkID,
		data.Name,
		data.StationASiteID,
		data.StationBSiteID,
		data.Technology,
		nullableFloat(data.FrequencyMHz),
		nullableFloat(data.CapacityMbps),
		nullableFloat(data.DistanceKM),
		data.Status,
		data.Observation,
	)
	if err != nil {
		return radiolink.RadioLink{}, mapWriteError(err, data.LinkID)
	}

	return r.FindByID(ctx, data.LinkID)
}

// FindByID returns a radio link by link_id.
func (r *Repository) FindByID(ctx context.Context, linkID string) (radiolink.RadioLink, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+radioLinkColumns+" FROM radio_links WHERE link_id = ?",
		linkID,
	)

	record, err := scanRadioLink(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return radiolink.RadioLink{}, fmt.Errorf("find radio link %q: %w", linkID, radiolink.ErrNotFound)
	}
	if err != nil {
		return radiolink.RadioLink{}, fmt.Errorf("find radio link %q: %w", linkID, err)
	}

	return record, nil
}

// List returns a page of radio links and the total number of matching records.
func (r *Repository) List(ctx context.Context, filter radiolink.ListFilter) ([]radiolink.RadioLink, int64, error) {
	whereClause, args := buildWhere(filter)

	var total int64
	countQuery := "SELECT COUNT(*) FROM radio_links" + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count radio links: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	query := "SELECT " + radioLinkColumns + " FROM radio_links" + whereClause +
		" ORDER BY link_id LIMIT ? OFFSET ?"

	queryArgs := append(append([]any{}, args...), filter.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list radio links: %w", err)
	}
	defer rows.Close()

	records := make([]radiolink.RadioLink, 0, filter.PageSize)
	for rows.Next() {
		record, err := scanRadioLink(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("scan radio link: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate radio links: %w", err)
	}

	return records, total, nil
}

// Update replaces the mutable fields of an existing radio link.
func (r *Repository) Update(ctx context.Context, data radiolink.RadioLinkData) (radiolink.RadioLink, error) {
	_, err := r.db.ExecContext(ctx, updateRadioLink,
		data.Name,
		data.StationASiteID,
		data.StationBSiteID,
		data.Technology,
		nullableFloat(data.FrequencyMHz),
		nullableFloat(data.CapacityMbps),
		nullableFloat(data.DistanceKM),
		data.Status,
		data.Observation,
		data.LinkID,
	)
	if err != nil {
		return radiolink.RadioLink{}, mapWriteError(err, data.LinkID)
	}

	return r.FindByID(ctx, data.LinkID)
}

// Delete removes a radio link by link_id.
func (r *Repository) Delete(ctx context.Context, linkID string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM radio_links WHERE link_id = ?", linkID)
	if err != nil {
		return fmt.Errorf("delete radio link %q: %w", linkID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete radio link %q: %w", linkID, err)
	}
	if affected == 0 {
		return fmt.Errorf("delete radio link %q: %w", linkID, radiolink.ErrNotFound)
	}

	return nil
}

func buildWhere(filter radiolink.ListFilter) (string, []any) {
	var conditions []string
	var args []any

	if filter.Search != "" {
		term := "%" + filter.Search + "%"
		conditions = append(conditions,
			"(link_id LIKE ? OR name LIKE ? OR station_a_site_id LIKE ? OR station_b_site_id LIKE ? OR technology LIKE ?)")
		args = append(args, term, term, term, term, term)
	}

	if filter.StationID != "" {
		conditions = append(conditions, "(station_a_site_id = ? OR station_b_site_id = ?)")
		args = append(args, filter.StationID, filter.StationID)
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanRadioLink(scan func(dest ...any) error) (radiolink.RadioLink, error) {
	var (
		record    radiolink.RadioLink
		frequency sql.NullFloat64
		capacity  sql.NullFloat64
		distance  sql.NullFloat64
	)

	err := scan(
		&record.LinkID,
		&record.Name,
		&record.StationASiteID,
		&record.StationBSiteID,
		&record.Technology,
		&frequency,
		&capacity,
		&distance,
		&record.Status,
		&record.Observation,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return radiolink.RadioLink{}, err
	}

	record.FrequencyMHz = floatPointer(frequency)
	record.CapacityMbps = floatPointer(capacity)
	record.DistanceKM = floatPointer(distance)

	return record, nil
}

func mapWriteError(err error, linkID string) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062:
			return fmt.Errorf("save radio link %q: %w", linkID, radiolink.ErrAlreadyExists)
		case 1452:
			return fmt.Errorf("save radio link %q: %w", linkID, radiolink.ErrStationNotFound)
		}
	}

	return fmt.Errorf("save radio link %q: %w", linkID, err)
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

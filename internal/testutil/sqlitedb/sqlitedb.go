// Package sqlitedb provides an in-memory SQLite database for tests.
package sqlitedb

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

var schema = []string{
	`CREATE TABLE stations (
		site_id TEXT PRIMARY KEY,
		technology TEXT NOT NULL DEFAULT '',
		end_id TEXT NOT NULL DEFAULT '',
		contract_area_type TEXT NOT NULL DEFAULT '',
		area_holder TEXT NOT NULL DEFAULT '',
		infrastructure_type TEXT NOT NULL DEFAULT '',
		ev_type TEXT NOT NULL DEFAULT '',
		address TEXT NOT NULL DEFAULT '',
		neighborhood TEXT NOT NULL DEFAULT '',
		city TEXT NOT NULL DEFAULT '',
		state TEXT NOT NULL DEFAULT '',
		zip_code TEXT NOT NULL DEFAULT '',
		regional TEXT NOT NULL DEFAULT '',
		latitude REAL NULL,
		longitude REAL NULL,
		status TEXT NOT NULL DEFAULT '',
		tower_type TEXT NOT NULL DEFAULT '',
		nominal_aev TEXT NOT NULL DEFAULT '',
		station_id TEXT NOT NULL DEFAULT '',
		ev_supplier TEXT NOT NULL DEFAULT '',
		land_area REAL NULL,
		structure_height REAL NULL,
		observation TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE radio_links (
		link_id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		station_a_site_id TEXT NOT NULL,
		station_b_site_id TEXT NOT NULL,
		technology TEXT NOT NULL DEFAULT '',
		frequency_mhz REAL NULL,
		capacity_mbps REAL NULL,
		distance_km REAL NULL,
		status TEXT NOT NULL DEFAULT '',
		observation TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (station_a_site_id) REFERENCES stations (site_id) ON DELETE RESTRICT,
		FOREIGN KEY (station_b_site_id) REFERENCES stations (site_id) ON DELETE RESTRICT,
		CHECK (station_a_site_id <> station_b_site_id)
	)`,
}

// Open creates a unique in-memory SQLite database with the application schema.
func Open(t *testing.T) *sql.DB {
	t.Helper()

	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", name)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable sqlite foreign keys: %v", err)
	}

	for _, statement := range schema {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create sqlite schema: %v", err)
		}
	}

	return db
}

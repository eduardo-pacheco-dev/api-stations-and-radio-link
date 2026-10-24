// Package database opens and configures a database/sql handle.
//
// The concrete driver is selected by configuration and registered by the
// composition root (cmd/api). No other package depends on a specific driver.
package database

import (
	"database/sql"
	"fmt"
	"time"
)

// Config configures a database/sql handle.
type Config struct {
	Driver          string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Open creates a database/sql handle and configures its connection pool.
//
// It does not open a connection: connectivity is verified by the status check,
// which lets the API start and report a degraded state while the database is
// unavailable.
func Open(cfg Config) (*sql.DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return db, nil
}

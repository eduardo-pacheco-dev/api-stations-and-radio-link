// Package sqlpinger adapts a database/sql handle to the status.DatabasePinger
// port. It works with any database/sql driver.
package sqlpinger

import (
	"context"
	"database/sql"
)

// Pinger checks database connectivity.
type Pinger struct {
	db     *sql.DB
	driver string
}

// New creates a Pinger for the given database handle and driver name.
func New(db *sql.DB, driver string) *Pinger {
	return &Pinger{db: db, driver: driver}
}

// Ping verifies that the database is reachable.
func (p *Pinger) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Driver returns the configured driver name.
func (p *Pinger) Driver() string {
	return p.driver
}

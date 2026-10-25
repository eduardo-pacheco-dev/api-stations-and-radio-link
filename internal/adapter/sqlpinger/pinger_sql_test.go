package sqlpinger_test

import (
	"context"
	"testing"
	"time"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/sqlpinger"
	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/testutil/sqlitedb"
)

func TestPinger_Ping(t *testing.T) {
	gormDB := sqlitedb.Open(t)
	db, err := gormDB.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}

	pinger := sqlpinger.New(db, "sqlite")

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := pinger.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if got := pinger.Driver(); got != "sqlite" {
		t.Errorf("Driver() = %q, want %q", got, "sqlite")
	}
}

func TestPinger_Ping_ClosedDatabase(t *testing.T) {
	gormDB := sqlitedb.Open(t)
	db, err := gormDB.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	pinger := sqlpinger.New(db, "sqlite")

	if err := pinger.Ping(t.Context()); err == nil {
		t.Fatal("Ping() error = nil, want error for closed database")
	}
}

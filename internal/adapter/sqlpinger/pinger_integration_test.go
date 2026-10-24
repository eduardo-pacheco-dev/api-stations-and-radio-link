//go:build integration

package sqlpinger_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/sqlpinger"
)

const defaultTestDSN = "root:admin@tcp(127.0.0.1:3306)/api_stations_radio_link?parseTime=true&charset=utf8mb4"

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("MySQL is not available: %v", err)
	}

	return db
}

func TestPinger_Ping_Integration(t *testing.T) {
	db := openTestDatabase(t)

	pinger := sqlpinger.New(db, "mysql")

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := pinger.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if got := pinger.Driver(); got != "mysql" {
		t.Errorf("Driver() = %q, want %q", got, "mysql")
	}
}

func TestPinger_Ping_Integration_ClosedDatabase(t *testing.T) {
	db := openTestDatabase(t)

	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	pinger := sqlpinger.New(db, "mysql")

	if err := pinger.Ping(t.Context()); err == nil {
		t.Fatal("Ping() error = nil, want error for closed database")
	}
}

package database_test

import (
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/adapter/database"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	db, err := database.Open(database.Config{
		Driver:          "mysql",
		DSN:             "root:admin@tcp(127.0.0.1:3306)/api_stations_radio_link?parseTime=true",
		MaxOpenConns:    7,
		MaxIdleConns:    3,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	if got := db.Stats().MaxOpenConnections; got != 7 {
		t.Errorf("MaxOpenConnections = %d, want 7", got)
	}
}

func TestOpen_UnknownDriver(t *testing.T) {
	t.Parallel()

	_, err := database.Open(database.Config{
		Driver: "unknown",
		DSN:    "ignored",
	})
	if err == nil {
		t.Fatal("Open() error = nil, want unknown driver error")
	}
}
